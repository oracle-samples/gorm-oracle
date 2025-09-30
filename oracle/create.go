/*
** Copyright (c) 2025 Oracle and/or its affiliates.
**
** The Universal Permissive License (UPL), Version 1.0
**
** Subject to the condition set forth below, permission is hereby granted to any
** person obtaining a copy of this software, associated documentation and/or data
** (collectively the "Software"), free of charge and under any and all copyright
** rights in the Software, and any and all patent rights owned or freely
** licensable by each licensor hereunder covering either (i) the unmodified
** Software as contributed to or provided by such licensor, or (ii) the Larger
** Works (as defined below), to deal in both
**
** (a) the Software, and
** (b) any piece of software and/or hardware listed in the lrgrwrks.txt file if
** one is included with the Software (each a "Larger Work" to which the Software
** is contributed by such licensors),
**
** without restriction, including without limitation the rights to copy, create
** derivative works of, display, perform, and distribute the Software and make,
** use, sell, offer for sale, import, export, have made, and have sold the
** Software and the Larger Work(s), and to sublicense the foregoing rights on
** either these or other terms.
**
** This license is subject to the following condition:
** The above copyright notice and either this complete permission notice or at
** a minimum a reference to the UPL must be included in all copies or
** substantial portions of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
** IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
** FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
** AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
** LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
** OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
 */

package oracle

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Create overrides GORM's create callback for Oracle.
//
// Behavior:
//   - If the schema has fields with default DB values and only one row is
//     being inserted, it builds an INSERT ... RETURNING statement.
//   - If no RETURNING is needed, it emits a standard INSERT.
//   - If multiple rows require RETURNING, it builds a PL/SQL block using
//     FORALL and BULK COLLECT; if an ON CONFLICT clause is present and
//     resolvable, it emits a MERGE.
//   - For that last case, it validates Dest (non-nil, non-empty slice with
//     no nil elements), normalizes bind variables for Oracle, and populates
//     destinations from OUT parameters.
//
// Register with:
//
//	db.Callback().Create().Replace("gorm:create", oracle.Create)
func Create(db *gorm.DB) {
	if db.Error != nil || db.Statement == nil {
		return
	}

	stmt := db.Statement

	// Check for nil values in slices before processing
	if err := validateCreateData(stmt); err != nil {
		db.AddError(err)
		return
	}

	stmtSchema := stmt.Schema
	if stmtSchema != nil && !stmt.Unscoped {
		for _, c := range stmtSchema.CreateClauses {
			stmt.AddClause(c)
		}
	}

	// SkipDefaultTransaction is here to distinguish the usage of DB.ToSQL
	if stmtSchema != nil && len(stmtSchema.FieldsWithDefaultDBValue) > 0 && (!db.DryRun || (db.DryRun && db.SkipDefaultTransaction)) {
		if _, ok := stmt.Clauses["RETURNING"]; !ok {
			fromColumns := make([]clause.Column, 0, len(stmtSchema.FieldsWithDefaultDBValue))
			for _, field := range stmtSchema.FieldsWithDefaultDBValue {
				fromColumns = append(fromColumns, clause.Column{Name: field.DBName})
			}
			stmt.AddClause(clause.Returning{Columns: fromColumns})
		}
	}

	if stmt.SQL.Len() == 0 {
		createValues := callbacks.ConvertToCreateValues(stmt)

		// Early validation for invalid data
		if len(createValues.Values) == 0 {
			db.AddError(gorm.ErrInvalidData)
			return
		}

		// Allow empty columns only if we have auto-increment primary key
		if len(createValues.Columns) == 0 {
			hasAutoIncrementPK := stmt.Schema != nil &&
				stmt.Schema.PrioritizedPrimaryField != nil &&
				stmt.Schema.PrioritizedPrimaryField.AutoIncrement
			if !hasAutoIncrementPK {
				db.AddError(gorm.ErrInvalidData)
				return
			}
		}

		// Validate that all value rows have the same number of columns
		expectedColumnCount := len(createValues.Columns)
		for i, valueRow := range createValues.Values {
			if len(valueRow) != expectedColumnCount {
				db.AddError(fmt.Errorf("invalid data: row %d has %d values, expected %d", i, len(valueRow), expectedColumnCount))
				return
			}
		}

		// Check if we need RETURNING clause for fields with default values
		_, hasReturningClause := db.Statement.Clauses["RETURNING"]
		hasReturningInDryRun := db.DryRun && hasReturningClause
		needsReturning := stmtSchema != nil && len(stmtSchema.FieldsWithDefaultDBValue) > 0 && (!db.DryRun || hasReturningInDryRun)

		if needsReturning && len(createValues.Values) > 1 {
			// Multiple rows with RETURNING - use PL/SQL
			buildBulkInsertPLSQL(db, createValues)
		} else if needsReturning {
			// Single row with RETURNING - use regular SQL with RETURNING
			buildSingleInsertSQL(db, createValues)
		} else {
			// No RETURNING needed - use standard INSERT
			buildStandardInsertSQL(db, createValues)
		}
	}
}

// validateCreateData checks for invalid data in the destination before processing
func validateCreateData(stmt *gorm.Statement) error {
	if stmt.Dest == nil {
		return gorm.ErrInvalidData
	}

	destValue := reflect.ValueOf(stmt.Dest)
	if destValue.Kind() == reflect.Ptr {
		if destValue.IsNil() {
			return gorm.ErrInvalidData
		}
		destValue = destValue.Elem()
	}

	// Check for nil values in slices
	if destValue.Kind() == reflect.Slice {
		if destValue.Len() == 0 {
			return gorm.ErrEmptySlice
		}
		for i := 0; i < destValue.Len(); i++ {
			item := destValue.Index(i)
			if item.Kind() == reflect.Ptr && item.IsNil() {
				return gorm.ErrInvalidData
			}
		}
	}

	return nil
}

// Build PL/SQL block for bulk INSERT/MERGE with RETURNING
func buildBulkInsertPLSQL(db *gorm.DB, createValues clause.Values) {
	sanitizeCreateValuesForBulkArrays(db.Statement, &createValues)

	stmt := db.Statement
	schema := stmt.Schema

	if schema == nil {
		db.AddError(fmt.Errorf("schema required for bulk insert with returning"))
		return
	}

	// Add validation for empty columns or values
	if len(createValues.Columns) == 0 {
		db.AddError(fmt.Errorf("no columns found for bulk insert"))
		return
	}

	if len(createValues.Values) == 0 {
		db.AddError(fmt.Errorf("no values found for bulk insert"))
		return
	}

	// Validate that all value rows have the same number of columns
	expectedColumnCount := len(createValues.Columns)
	for i, valueRow := range createValues.Values {
		if len(valueRow) != expectedColumnCount {
			db.AddError(fmt.Errorf("row %d has %d values, expected %d", i, len(valueRow), expectedColumnCount))
			return
		}
	}

	// Check if we have OnConflict clause
	onConflictClause, hasOnConflict := stmt.Clauses["ON CONFLICT"]
	if hasOnConflict {
		onConflict, ok := onConflictClause.Expression.(clause.OnConflict)
		if !ok {
			db.AddError(fmt.Errorf("invalid OnConflict clause"))
			return
		}

		// Determine conflict columns (use primary key if not specified)
		conflictColumns := onConflict.Columns
		if len(conflictColumns) == 0 {
			if len(schema.PrimaryFields) == 0 {
				return
			}
			for _, primaryField := range schema.PrimaryFields {
				conflictColumns = append(conflictColumns, clause.Column{Name: primaryField.DBName})
			}
		}

		shouldUseMerge := ShouldUseRealConflict(createValues, onConflict, conflictColumns)

		if shouldUseMerge {
			buildBulkMergePLSQL(db, createValues, onConflictClause)
			return
		}
	}
	// Original INSERT logic for when there's no conflict handling needed
	buildBulkInsertOnlyPLSQL(db, createValues)
}

// Build PL/SQL block for bulk MERGE with RETURNING (OnConflict case)
func buildBulkMergePLSQL(db *gorm.DB, createValues clause.Values, onConflictClause clause.Clause) {
	sanitizeCreateValuesForBulkArrays(db.Statement, &createValues)

	stmt := db.Statement
	schema := stmt.Schema

	onConflict, ok := onConflictClause.Expression.(clause.OnConflict)
	if !ok {
		db.AddError(fmt.Errorf("invalid OnConflict clause"))
		return
	}

	// Determine conflict columns (use primary key if not specified)
	conflictColumns := onConflict.Columns
	if len(conflictColumns) == 0 {
		if schema == nil || len(schema.PrimaryFields) == 0 {
			return
		}
		for _, primaryField := range schema.PrimaryFields {
			conflictColumns = append(conflictColumns, clause.Column{Name: primaryField.DBName})
		}
	}

	// Filter conflict columns to only include those present in createValues
	var valuesColumnMap = make(map[string]bool)
	for _, column := range createValues.Columns {
		valuesColumnMap[strings.ToUpper(column.Name)] = true
	}

	// Filter conflict columns to remove non unique columns
	var filteredConflictColumns []clause.Column
	for _, conflictCol := range conflictColumns {
		field := stmt.Schema.LookUpField(conflictCol.Name)
		if valuesColumnMap[strings.ToUpper(conflictCol.Name)] && (field.Unique || field.AutoIncrement) {
			filteredConflictColumns = append(filteredConflictColumns, conflictCol)
		}
	}

	// Check if we have any usable conflict columns
	if len(filteredConflictColumns) == 0 {
		buildBulkInsertOnlyPLSQL(db, createValues)
		return
	}

	// Use filtered conflict columns from here on
	conflictColumns = filteredConflictColumns

	var plsqlBuilder strings.Builder

	// Start PL/SQL block
	plsqlBuilder.WriteString("DECLARE\n")
	writeTableRecordCollectionDecl(db, &plsqlBuilder, stmt.Schema.DBNames, stmt.Table)
	plsqlBuilder.WriteString("  l_affected_records t_records;\n")

	// Create array types and variables for each column
	for i, column := range createValues.Columns {
		var arrayType string
		if field := findFieldByDBName(schema, column.Name); field != nil {
			arrayType = getOracleArrayType(field)
		} else {
			arrayType = "TABLE OF VARCHAR2(4000)"
		}
		for i, ccolumn := range createValues.Columns {
			if strings.EqualFold(ccolumn.Name, column.Name) {
				for j := range createValues.Values {
					if strValue, ok := createValues.Values[j][i].(string); ok {
						if len(strValue) > 4000 {
							arrayType = "TABLE OF CLOB"
						}
					}
				}
			}
		}

		plsqlBuilder.WriteString(fmt.Sprintf("  TYPE t_col_%d_array IS %s;\n", i, arrayType))
		plsqlBuilder.WriteString(fmt.Sprintf("  l_col_%d_array t_col_%d_array;\n", i, i))
	}

	plsqlBuilder.WriteString("BEGIN\n")

	// Initialize arrays with values
	for i := range createValues.Columns {
		plsqlBuilder.WriteString(fmt.Sprintf("  l_col_%d_array := t_col_%d_array(", i, i))
		for j, values := range createValues.Values {
			if j > 0 {
				plsqlBuilder.WriteString(", ")
			}
			plsqlBuilder.WriteString(fmt.Sprintf(":%d", len(stmt.Vars)+1))
			stmt.Vars = append(stmt.Vars, convertValue(values[i]))
		}
		plsqlBuilder.WriteString(");\n")
	}

	// FORALL with MERGE and RETURNING BULK COLLECT INTO
	plsqlBuilder.WriteString(fmt.Sprintf("  FORALL i IN 1..%d\n", len(createValues.Values)))
	plsqlBuilder.WriteString("    MERGE INTO ")
	db.QuoteTo(&plsqlBuilder, stmt.Table)
	plsqlBuilder.WriteString(" t\n")
	// Build USING clause
	plsqlBuilder.WriteString("    USING (SELECT ")
	for idx, column := range createValues.Columns {
		if idx > 0 {
			plsqlBuilder.WriteString(", ")
		}
		plsqlBuilder.WriteString(fmt.Sprintf("l_col_%d_array(i) AS ", idx))
		db.QuoteTo(&plsqlBuilder, column.Name)
	}
	plsqlBuilder.WriteString(" FROM DUAL) s\n")

	// Build ON clause using conflict columns
	plsqlBuilder.WriteString("    ON (")

	for idx, conflictCol := range conflictColumns {
		if idx > 0 {
			plsqlBuilder.WriteString(" AND ")
		}
		plsqlBuilder.WriteString("t.")
		db.QuoteTo(&plsqlBuilder, conflictCol.Name)
		plsqlBuilder.WriteString(" = s.")
		db.QuoteTo(&plsqlBuilder, conflictCol.Name)
	}
	plsqlBuilder.WriteString(")\n")

	// WHEN MATCHED THEN UPDATE (if DoUpdates specified)
	if len(onConflict.DoUpdates) > 0 {
		plsqlBuilder.WriteString("    WHEN MATCHED THEN UPDATE SET ")

		// Build update assignments
		updateCount := 0
		for _, column := range createValues.Columns {
			// Skip conflict columns in updates
			isConflictColumn := false
			for _, conflictCol := range conflictColumns {
				if strings.EqualFold(column.Name, conflictCol.Name) {
					isConflictColumn = true
					break
				}
			}

			if !isConflictColumn {
				if updateCount > 0 {
					plsqlBuilder.WriteString(", ")
				}
				plsqlBuilder.WriteString("t.")
				db.QuoteTo(&plsqlBuilder, column.Name)
				plsqlBuilder.WriteString(" = s.")
				db.QuoteTo(&plsqlBuilder, column.Name)
				updateCount++
			}
		}
		plsqlBuilder.WriteString("\n")
	} else if !onConflict.DoNothing {
		// Default behavior: update all non-conflict columns
		plsqlBuilder.WriteString("    WHEN MATCHED THEN UPDATE SET ")

		updateCount := 0
		for _, column := range createValues.Columns {
			// Skip conflict columns and auto-increment fields
			isConflictColumn := false
			for _, conflictCol := range conflictColumns {
				if strings.EqualFold(column.Name, conflictCol.Name) {
					isConflictColumn = true
					break
				}
			}

			isAutoIncrement := false
			if schema.PrioritizedPrimaryField != nil &&
				schema.PrioritizedPrimaryField.AutoIncrement &&
				strings.EqualFold(schema.PrioritizedPrimaryField.DBName, column.Name) {
				isAutoIncrement = true
			} else if stmt.Schema.LookUpField(column.Name).AutoIncrement {
				isAutoIncrement = true
			}

			if !isConflictColumn && !isAutoIncrement {
				if updateCount > 0 {
					plsqlBuilder.WriteString(", ")
				}
				plsqlBuilder.WriteString("t.")
				db.QuoteTo(&plsqlBuilder, column.Name)
				plsqlBuilder.WriteString(" = s.")
				db.QuoteTo(&plsqlBuilder, column.Name)
				updateCount++
			}
		}
		plsqlBuilder.WriteString("\n")
	} else {
		onCols := map[string]struct{}{}
		for _, c := range conflictColumns {
			onCols[strings.ToUpper(c.Name)] = struct{}{}
		}

		// Picking the first non-ON column from the INSERT/MERGE columns
		var noopCol string
		for _, c := range createValues.Columns {
			if _, inOn := onCols[strings.ToUpper(c.Name)]; !inOn {
				noopCol = c.Name
				break
			}
		}
		plsqlBuilder.WriteString("    WHEN MATCHED THEN UPDATE SET t.")
		db.QuoteTo(&plsqlBuilder, noopCol)
		plsqlBuilder.WriteString(" = t.")
		db.QuoteTo(&plsqlBuilder, noopCol)
		plsqlBuilder.WriteString("\n")
	}

	// WHEN NOT MATCHED THEN INSERT (unless DoNothing for inserts)
	if !onConflict.DoNothing {
		plsqlBuilder.WriteString("    WHEN NOT MATCHED THEN INSERT (")

		// Add column names (excluding auto-increment primary key)
		insertCount := 0
		for _, column := range createValues.Columns {
			if shouldIncludeColumnInInsert(stmt, column.Name) {
				if insertCount > 0 {
					plsqlBuilder.WriteString(", ")
				}
				db.QuoteTo(&plsqlBuilder, column.Name)
				insertCount++
			}
		}

		plsqlBuilder.WriteString(") VALUES (")

		// Add values (excluding auto-increment primary key)
		insertCount = 0
		for _, column := range createValues.Columns {
			if shouldIncludeColumnInInsert(stmt, column.Name) {
				if insertCount > 0 {
					plsqlBuilder.WriteString(", ")
				}
				plsqlBuilder.WriteString("s.")
				db.QuoteTo(&plsqlBuilder, column.Name)
				insertCount++
			}
		}
		plsqlBuilder.WriteString(")\n")
	} else {
		// Add a minimal WHEN NOT MATCHED that effectively does nothing by only inserting required fields
		plsqlBuilder.WriteString("    WHEN NOT MATCHED THEN INSERT (")

		// Find at least one non-auto-increment column to satisfy Oracle syntax
		insertCount := 0
		for _, column := range createValues.Columns {
			if shouldIncludeColumnInInsert(stmt, column.Name) {
				if insertCount > 0 {
					plsqlBuilder.WriteString(", ")
				}
				db.QuoteTo(&plsqlBuilder, column.Name)
				insertCount++
			}
		}

		plsqlBuilder.WriteString(") VALUES (")

		insertCount = 0
		for _, column := range createValues.Columns {
			if shouldIncludeColumnInInsert(stmt, column.Name) {
				if insertCount > 0 {
					plsqlBuilder.WriteString(", ")
				}
				plsqlBuilder.WriteString("s.")
				db.QuoteTo(&plsqlBuilder, column.Name)
				insertCount++
			}
		}
		plsqlBuilder.WriteString(")\n")
	}

	// Add RETURNING clause with BULK COLLECT INTO
	plsqlBuilder.WriteString("    RETURNING ")
	allColumns := getAllTableColumns(schema)
	for i, column := range allColumns {
		if i > 0 {
			plsqlBuilder.WriteString(", ")
		}
		db.QuoteTo(&plsqlBuilder, column)
	}
	plsqlBuilder.WriteString("\n    BULK COLLECT INTO l_affected_records;\n")

	// Add OUT parameter population (JSON serialized to CLOB)
	outParamIndex := len(stmt.Vars)
	for rowIdx := 0; rowIdx < len(createValues.Values); rowIdx++ {
		for _, column := range allColumns {
			if field := findFieldByDBName(schema, column); field != nil {
				if isJSONField(field) {
					if isRawMessageField(field) {
						// Column is a BLOB, return raw bytes; no JSON_SERIALIZE
						stmt.Vars = append(stmt.Vars, sql.Out{Dest: new([]byte)})
						plsqlBuilder.WriteString(fmt.Sprintf(
							"  IF l_affected_records.COUNT > %d THEN :%d := l_affected_records(%d).",
							rowIdx, outParamIndex+1, rowIdx+1,
						))
						db.QuoteTo(&plsqlBuilder, column)
						plsqlBuilder.WriteString("; END IF;\n")
					} else {
						// datatypes.JSON (text-based) -> serialize to CLOB
						stmt.Vars = append(stmt.Vars, sql.Out{Dest: new(string)})
						plsqlBuilder.WriteString(fmt.Sprintf(
							"  IF l_affected_records.COUNT > %d THEN :%d := JSON_SERIALIZE(l_affected_records(%d).",
							rowIdx, outParamIndex+1, rowIdx+1,
						))
						db.QuoteTo(&plsqlBuilder, column)
						plsqlBuilder.WriteString(" RETURNING CLOB); END IF;\n")
					}
				} else {
					stmt.Vars = append(stmt.Vars, sql.Out{Dest: createTypedDestination(field)})
					plsqlBuilder.WriteString(fmt.Sprintf("  IF l_affected_records.COUNT > %d THEN :%d := l_affected_records(%d).", rowIdx, outParamIndex+1, rowIdx+1))
					db.QuoteTo(&plsqlBuilder, column)
					plsqlBuilder.WriteString("; END IF;\n")
				}
				outParamIndex++
			}
		}
	}

	plsqlBuilder.WriteString("END;")

	stmt.SQL.Reset()
	stmt.SQL.WriteString(plsqlBuilder.String())

	if !db.DryRun && db.Error == nil {
		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if db.AddError(err) == nil {
			db.RowsAffected = int64(len(createValues.Values))
			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
			getBulkReturningValues(db, len(createValues.Values))
		}
	}
}

// Build PL/SQL block for bulk INSERT only (no conflict handling)
func buildBulkInsertOnlyPLSQL(db *gorm.DB, createValues clause.Values) {
	stmt := db.Statement
	schema := stmt.Schema

	var plsqlBuilder strings.Builder

	// Start PL/SQL block
	plsqlBuilder.WriteString("DECLARE\n")
	writeTableRecordCollectionDecl(db, &plsqlBuilder, stmt.Schema.DBNames, stmt.Table)
	plsqlBuilder.WriteString("  l_inserted_records t_records;\n")

	// Create array types and variables for each column
	for i, column := range createValues.Columns {
		var arrayType string
		if field := findFieldByDBName(schema, column.Name); field != nil {
			arrayType = getOracleArrayType(field)
		} else {
			arrayType = "TABLE OF VARCHAR2(4000)"
		}
		for i, ccolumn := range createValues.Columns {
			if strings.EqualFold(ccolumn.Name, column.Name) {
				for j := range createValues.Values {
					if strValue, ok := createValues.Values[j][i].(string); ok {
						if len(strValue) > 4000 {
							arrayType = "TABLE OF CLOB"
						}
					}
				}
			}
		}

		plsqlBuilder.WriteString(fmt.Sprintf("  TYPE t_col_%d_array IS %s;\n", i, arrayType))
		plsqlBuilder.WriteString(fmt.Sprintf("  l_col_%d_array t_col_%d_array;\n", i, i))
	}

	plsqlBuilder.WriteString("BEGIN\n")

	// Initialize arrays with values
	for i := range createValues.Columns {
		plsqlBuilder.WriteString(fmt.Sprintf("  l_col_%d_array := t_col_%d_array(", i, i))
		for j, values := range createValues.Values {
			if j > 0 {
				plsqlBuilder.WriteString(", ")
			}
			plsqlBuilder.WriteString(fmt.Sprintf(":%d", len(stmt.Vars)+1))
			stmt.Vars = append(stmt.Vars, convertValue(values[i]))
		}
		plsqlBuilder.WriteString(");\n")
	}

	// FORALL with RETURNING BULK COLLECT INTO
	plsqlBuilder.WriteString(fmt.Sprintf("  FORALL i IN 1..%d\n", len(createValues.Values)))
	plsqlBuilder.WriteString("    INSERT INTO ")
	db.QuoteTo(&plsqlBuilder, stmt.Table)
	plsqlBuilder.WriteString(" (")
	// Add column names
	for i, column := range createValues.Columns {
		if i > 0 {
			plsqlBuilder.WriteString(", ")
		}
		db.QuoteTo(&plsqlBuilder, column.Name)
	}
	plsqlBuilder.WriteString(") VALUES (")

	// Add array references
	for i := range createValues.Columns {
		if i > 0 {
			plsqlBuilder.WriteString(", ")
		}
		plsqlBuilder.WriteString(fmt.Sprintf("l_col_%d_array(i)", i))
	}
	plsqlBuilder.WriteString(")\n")

	// Add RETURNING clause with BULK COLLECT INTO
	plsqlBuilder.WriteString("    RETURNING ")
	allColumns := getAllTableColumns(schema)
	for i, column := range allColumns {
		if i > 0 {
			plsqlBuilder.WriteString(", ")
		}
		db.QuoteTo(&plsqlBuilder, column)
	}
	plsqlBuilder.WriteString("\n    BULK COLLECT INTO l_inserted_records;\n")

	// Add OUT parameter population (JSON serialized to CLOB)
	outParamIndex := len(stmt.Vars)
	for rowIdx := 0; rowIdx < len(createValues.Values); rowIdx++ {
		for _, column := range allColumns {
			var columnBuilder strings.Builder
			db.QuoteTo(&columnBuilder, column)
			quotedColumn := columnBuilder.String()

			if field := findFieldByDBName(schema, column); field != nil {
				if isJSONField(field) {
					if isRawMessageField(field) {
						// Column is a BLOB, return raw bytes; no JSON_SERIALIZE
						stmt.Vars = append(stmt.Vars, sql.Out{Dest: new([]byte)})
						plsqlBuilder.WriteString(fmt.Sprintf(
							"  IF l_inserted_records.COUNT > %d THEN :%d := l_inserted_records(%d).%s; END IF;\n",
							rowIdx, outParamIndex+1, rowIdx+1, quotedColumn,
						))
					} else {
						// datatypes.JSON (text-based) -> serialize to CLOB
						stmt.Vars = append(stmt.Vars, sql.Out{Dest: new(string)})
						plsqlBuilder.WriteString(fmt.Sprintf(
							"  IF l_inserted_records.COUNT > %d THEN :%d := JSON_SERIALIZE(l_inserted_records(%d).%s RETURNING CLOB); END IF;\n",
							rowIdx, outParamIndex+1, rowIdx+1, quotedColumn,
						))
					}
				} else {
					stmt.Vars = append(stmt.Vars, sql.Out{Dest: createTypedDestination(field)})
					plsqlBuilder.WriteString(fmt.Sprintf(
						"  IF l_inserted_records.COUNT > %d THEN :%d := l_inserted_records(%d).%s; END IF;\n",
						rowIdx, outParamIndex+1, rowIdx+1, quotedColumn,
					))
				}
				outParamIndex++
			}
		}
	}

	plsqlBuilder.WriteString("END;")

	stmt.SQL.Reset()
	stmt.SQL.WriteString(plsqlBuilder.String())

	if !db.DryRun && db.Error == nil {
		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if db.AddError(err) == nil {
			db.RowsAffected = int64(len(createValues.Values))
			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
			getBulkReturningValues(db, len(createValues.Values))
		}
	}
}

// Helper function to determine if column should be included in INSERT
func shouldIncludeColumnInInsert(stmt *gorm.Statement, columnName string) bool {
	if stmt.Schema.PrioritizedPrimaryField != nil &&
		stmt.Schema.PrioritizedPrimaryField.AutoIncrement &&
		strings.EqualFold(stmt.Schema.PrioritizedPrimaryField.DBName, columnName) {
		return false
	} else if stmt.Schema.LookUpField(columnName).AutoIncrement {
		return false
	}
	return true
}

// Build single INSERT with RETURNING
func buildSingleInsertSQL(db *gorm.DB, createValues clause.Values) {
	stmt := db.Statement

	stmt.AddClauseIfNotExists(clause.Insert{})
	stmt.AddClause(clause.Values{
		Columns: createValues.Columns,
		Values:  createValues.Values,
	})

	// Add RETURNING clause for fields with default values
	// addReturningClause(db, schema.FieldsWithDefaultDBValue)
	stmt.Build("INSERT", "VALUES", "ON CONFLICT", "RETURNING")

	if !db.DryRun && db.Error == nil {
		// Convert values for Oracle
		for i, val := range stmt.Vars {
			if !isOutParam(stmt.Vars[i]) {
				stmt.Vars[i] = convertValue(val)
			}
		}

		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if db.AddError(err) == nil {
			db.RowsAffected, _ = result.RowsAffected()
			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
			if db.RowsAffected > 0 {
				// Something was inserted/updated, process RETURNING values
				handleSingleRowReturning(db)
			}
		}
	}
}

// Build standard INSERT without RETURNING
func buildStandardInsertSQL(db *gorm.DB, createValues clause.Values) {
	stmt := db.Statement

	stmt.AddClauseIfNotExists(clause.Insert{})
	stmt.AddClause(clause.Values{
		Columns: createValues.Columns,
		Values:  createValues.Values,
	})
	stmt.Build("INSERT", "VALUES", "ON CONFLICT")

	if !db.DryRun && db.Error == nil {
		// Convert values for Oracle
		for i, val := range stmt.Vars {
			stmt.Vars[i] = convertValue(val)
		}

		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if db.AddError(err) == nil {
			db.RowsAffected, _ = result.RowsAffected()
			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
			if db.RowsAffected > 0 {
				handleLastInsertId(db, result)
			}
		}
	}
}

// Handle single row RETURNING results
func handleSingleRowReturning(db *gorm.DB) {

	if db.Statement.Schema == nil {
		return
	}

	// Get the RETURNING clause to know which columns we're expecting
	returningClause, hasReturning := db.Statement.Clauses["RETURNING"]
	if !hasReturning {
		return
	}

	returning, ok := returningClause.Expression.(clause.Returning)
	if !ok || len(returning.Columns) == 0 {
		return
	}

	// Get target struct to populate
	targetValue := db.Statement.ReflectValue
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	// For single row operations, we expect a single struct
	var targetStruct reflect.Value
	switch targetValue.Kind() {
	case reflect.Slice, reflect.Array:
		if targetValue.Len() > 0 {
			targetStruct = targetValue.Index(0) // First element
		} else {
			return
		}
	case reflect.Struct:
		targetStruct = targetValue
	default:
		return
	}

	// Process the sql.Out parameters created by ReturningClauseBuilder
	outIndex := 0

	// Find where the OUT parameters start (after the data variables)
	dataVarCount := 0
	if valuesClause, hasValues := db.Statement.Clauses["VALUES"]; hasValues {
		if values, ok := valuesClause.Expression.(clause.Values); ok && len(values.Values) > 0 {
			dataVarCount = len(values.Values[0]) // Number of columns in first row
		}
	}

	// Process OUT parameters
	for i := dataVarCount; i < len(db.Statement.Vars); i++ {
		if outParam, ok := db.Statement.Vars[i].(sql.Out); ok {

			if outIndex < len(returning.Columns) {
				columnName := returning.Columns[outIndex].Name
				field := findFieldByDBName(db.Statement.Schema, columnName)

				if field != nil && outParam.Dest != nil {
					// Extract the actual value from the OUT parameter destination
					destValue := reflect.ValueOf(outParam.Dest)

					if destValue.Kind() == reflect.Ptr && !destValue.IsNil() {
						actualValue := destValue.Elem().Interface()

						// Convert Oracle-specific values back to Go types
						if convertedValue := convertFromOracleToField(actualValue, field); convertedValue != nil {

							// Log target struct before setting
							if targetStruct.Kind() == reflect.Ptr {
								targetStruct = targetStruct.Elem()
							}

							if err := field.Set(db.Statement.Context, targetStruct, convertedValue); err != nil {
								db.AddError(fmt.Errorf("failed to set field %s: %w", field.Name, err))
							}
						}
					}
				}
				outIndex++
			}
		}
	}
}

// Handle bulk RETURNING results for PL/SQL operations
func getBulkReturningValues(db *gorm.DB, rowCount int) {
	if db.Statement.Schema == nil {
		return
	}

	// Get target slice to populate
	targetValue := db.Statement.ReflectValue
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	if targetValue.Kind() != reflect.Slice {
		return
	}

	// Grow slice if needed
	actualRowsToProcess := rowCount
	if actualRowsToProcess > targetValue.Len() {
		newSlice := reflect.MakeSlice(targetValue.Type(), actualRowsToProcess, actualRowsToProcess)
		if targetValue.Len() > 0 {
			reflect.Copy(newSlice, targetValue)
		}
		targetValue.Set(newSlice)
	}

	// Get all table columns
	allColumns := getAllTableColumns(db.Statement.Schema)

	// Find the actual starting index of OUT parameters
	actualStartIndex := -1
	for i := 0; i < len(db.Statement.Vars); i++ {
		if _, ok := db.Statement.Vars[i].(sql.Out); ok {
			actualStartIndex = i
			break
		}
	}

	if actualStartIndex == -1 {
		return
	}

	// Process OUT parameters for each row
	for rowIdx := 0; rowIdx < actualRowsToProcess; rowIdx++ {
		targetElement := targetValue.Index(rowIdx)

		// Handle interface{} wrapper
		if targetElement.Kind() == reflect.Interface {
			targetElement = targetElement.Elem()
		}

		for colIdx, column := range allColumns {
			paramIndex := actualStartIndex + (rowIdx * len(allColumns)) + colIdx

			if paramIndex < len(db.Statement.Vars) {
				if outParam, ok := db.Statement.Vars[paramIndex].(sql.Out); ok {
					if field := findFieldByDBName(db.Statement.Schema, column); field != nil && outParam.Dest != nil {
						destValue := reflect.ValueOf(outParam.Dest)
						if destValue.Kind() == reflect.Ptr && !destValue.IsNil() {
							actualValue := destValue.Elem().Interface()

							if convertedValue := convertFromOracleToField(actualValue, field); convertedValue != nil {
								// Check if target is a map or struct and handle accordingly
								if targetElement.Kind() == reflect.Map {
									// Handle map: set using field name as key
									targetElement.SetMapIndex(reflect.ValueOf(field.Name), reflect.ValueOf(convertedValue))
								} else {
									// Handle struct
									if err := field.Set(db.Statement.Context, targetElement, convertedValue); err != nil {
										db.AddError(fmt.Errorf("failed to set field %s: %w", field.Name, err))
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// Handle LastInsertId for auto-increment primary keys
func handleLastInsertId(db *gorm.DB, result sql.Result) {
	stmt := db.Statement

	if stmt.Schema == nil {
		return
	}

	insertID, err := result.LastInsertId()
	if err != nil || insertID <= 0 {
		return
	}

	pkField := stmt.Schema.PrioritizedPrimaryField
	if pkField == nil || !pkField.HasDefaultValue {
		return
	}

	// Handle struct types (most common case)
	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < stmt.ReflectValue.Len(); i++ {
			rv := stmt.ReflectValue.Index(i)
			if reflect.Indirect(rv).Kind() != reflect.Struct {
				break
			}

			if _, isZero := pkField.ValueOf(stmt.Context, rv); isZero {
				db.AddError(pkField.Set(stmt.Context, rv, insertID))
				insertID += pkField.AutoIncrementIncrement
			}
		}
	case reflect.Struct:
		if _, isZero := pkField.ValueOf(stmt.Context, stmt.ReflectValue); isZero {
			db.AddError(pkField.Set(stmt.Context, stmt.ReflectValue, insertID))
		}
	}
}

// This replaces expressions (clause.Expr) in bulk insert values
// with appropriate NULL placeholders based on the column's data type. This ensures that
// PL/SQL array binding remains consistent and avoids unsupported expressions during
// FORALL bulk operations.
func sanitizeCreateValuesForBulkArrays(stmt *gorm.Statement, cv *clause.Values) {
	for r := range cv.Values {
		for c, col := range cv.Columns {
			v := cv.Values[r][c]
			switch v.(type) {
			case clause.Expr:
				if f := findFieldByDBName(stmt.Schema, col.Name); f != nil {
					switch f.DataType {
					case schema.Int, schema.Uint:
						cv.Values[r][c] = sql.NullInt64{}
					case schema.Float:
						cv.Values[r][c] = sql.NullFloat64{}
					case schema.String:
						cv.Values[r][c] = sql.NullString{}
					case schema.Time:
						cv.Values[r][c] = sql.NullTime{}
					default:
						cv.Values[r][c] = nil
					}
				} else {
					cv.Values[r][c] = nil
				}
			}
		}
	}
}
