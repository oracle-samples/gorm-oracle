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
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Delete overrides GORM's delete callback for Oracle.
//
// Delete builds a safe, Oracle-compatible DELETE that supports soft deletes,
// hard deletes, and optional RETURNING of deleted rows.
//
// Behavior:
//   - DELETE safety: checks for missing WHERE conditions and refuses to run
//     unless AllowGlobalUpdate is set or the WHERE clause has meaningful
//     conditions.
//   - Soft delete: if the model has a soft-delete field and Unscoped is false,
//     it lets GORM emit the UPDATE that marks rows as deleted. If a RETURNING
//     clause is present with soft delete, it executes via QueryContext so the
//     returned columns can be scanned.
//   - Hard delete + RETURNING: it emits a PL/SQL block that performs DELETE …
//     RETURNING BULK COLLECT INTO, wiring per-row sql.Out destinations so the
//     deleted rows (or selected columns) can be populated back into the
//     destination slice.
//   - Hard delete (no RETURNING): it emits a standard DELETE and executes it
//     via ExecContext.
//   - Expressions: it expands WHERE expressions, including IN with slices, and
//     normalizes bind variables for Oracle.
//
// Register with:
//
//	db.Callback().Delete().Replace("gorm:delete", oracle.Delete)
func Delete(db *gorm.DB) {
	if db.Error != nil || db.Statement == nil {
		return
	}

	stmt := db.Statement

	if stmt.ReflectValue.IsValid() {
		modelValue := stmt.ReflectValue
		for modelValue.Kind() == reflect.Ptr {
			modelValue = modelValue.Elem()
		}
	}

	// Build WHERE clause based on primary keys FIRST
	if stmt.Schema != nil {
		addPrimaryKeyWhereClause(db)
	}

	// redirect soft-delete to update clause with bulk returning
	if stmt.Schema != nil {
		if deletedAtField := stmt.Schema.LookUpField("deleted_at"); deletedAtField != nil && !stmt.Unscoped {
			for _, c := range stmt.Schema.DeleteClauses {
				stmt.AddClause(c)
			}
			delete(stmt.Clauses, "DELETE")
			delete(stmt.Clauses, "FROM")
			stmt.SQL.Reset()
			stmt.Vars = stmt.Vars[:0]
			stmt.AddClauseIfNotExists(clause.Update{})
			Update(db)
			return
		}
	}

	// This prevents soft deletes from bypassing the safety check
	checkMissingWhereConditions(db)
	if db.Error != nil {
		return
	}

	// Add schema-defined delete clauses (like soft delete clauses) ONLY after safety checks pass
	if stmt.Schema != nil {
		for _, c := range stmt.Schema.DeleteClauses {
			// Only add soft delete clauses if not using Unscoped
			if stmt.Unscoped {
				// For hard delete, skip soft delete related clauses
				continue
			}
			stmt.AddClause(c)
		}
	}

	if stmt.SQL.Len() == 0 {
		stmt.SQL.Grow(100)
		stmt.AddClauseIfNotExists(clause.Delete{})

		// Build WHERE clause based on primary keys (like default callback)
		if stmt.Schema != nil {
			addPrimaryKeyWhereClause(db)
		}

		stmt.AddClauseIfNotExists(clause.From{})

		// Check if we need to capture deleted records
		_, hasReturning := db.Statement.Clauses["RETURNING"]
		needsReturning := stmt.Schema != nil && hasReturning

		if needsReturning {
			buildBulkDeletePLSQL(db)
		} else {
			buildStandardDeleteSQL(db)
		}

		// Safety check AFTER SQL is built
		checkMissingWhereConditions(db)
		if db.Error != nil {
			return
		}
	}

	// Execute the statement
	executeDelete(db)
}

// Add primary key WHERE clause like the default callback
func addPrimaryKeyWhereClause(db *gorm.DB) {
	stmt := db.Statement
	if stmt.Schema == nil {
		return
	}

	// First, try to extract from ReflectValue
	_, queryValues := schema.GetIdentityFieldValuesMap(
		stmt.Context,
		stmt.ReflectValue,
		stmt.Schema.PrimaryFields,
	)

	if len(queryValues) > 0 {
		column, values := schema.ToQueryValues(
			stmt.Table,
			stmt.Schema.PrimaryFieldDBNames,
			queryValues,
		)

		if len(values) > 0 {
			stmt.AddClause(clause.Where{
				Exprs: []clause.Expression{
					clause.IN{Column: column, Values: values},
				},
			})
		}
	}

	// Handle the dual model check from default GORM
	if stmt.ReflectValue.CanAddr() && stmt.Dest != stmt.Model && stmt.Model != nil {
		_, additionalQueryValues := schema.GetIdentityFieldValuesMap(
			stmt.Context,
			reflect.ValueOf(stmt.Model),
			stmt.Schema.PrimaryFields,
		)

		if len(additionalQueryValues) > 0 {
			column, values := schema.ToQueryValues(
				stmt.Table,
				stmt.Schema.PrimaryFieldDBNames,
				additionalQueryValues,
			)

			if len(values) > 0 {
				stmt.AddClause(clause.Where{
					Exprs: []clause.Expression{
						clause.IN{Column: column, Values: values},
					},
				})
			}
		}
	}
}

// Build standard DELETE without RETURNING
func buildStandardDeleteSQL(db *gorm.DB) {
	stmt := db.Statement

	// Check if this is a soft delete model and we're not using Unscoped
	if stmt.Schema != nil {
		if deletedAtField := stmt.Schema.LookUpField("deleted_at"); deletedAtField != nil && !stmt.Unscoped {
			// For soft delete, let GORM handle it normally - it will convert to UPDATE
			stmt.Build(stmt.BuildClauses...)
			return
		}
	}

	// For hard delete (either no soft delete field or Unscoped is true)
	// We don't call Build() yet - we let the safety check happen first
	// The actual building will happen in executeDelete()
}

// Build PL/SQL block for bulk DELETE with RETURNING
func buildBulkDeletePLSQL(db *gorm.DB) {
	stmt := db.Statement
	schema := stmt.Schema

	if schema == nil {
		db.AddError(fmt.Errorf("schema required for bulk delete with returning"))
		return
	}

	// Check if this is a soft delete model and we're not using Unscoped
	if deletedAtField := schema.LookUpField("deleted_at"); deletedAtField != nil && !stmt.Unscoped {
		// For soft delete with RETURNING, let GORM handle it normally
		stmt.Build(stmt.BuildClauses...)
		return
	}

	// For hard delete with RETURNING, use PL/SQL
	var plsqlBuilder strings.Builder

	// Start PL/SQL block
	plsqlBuilder.WriteString("DECLARE\n")
	writeTableRecordCollectionDecl(db, &plsqlBuilder, stmt.Schema.DBNames, stmt.Table)
	plsqlBuilder.WriteString("  l_deleted_records t_records;\n")
	plsqlBuilder.WriteString("BEGIN\n")

	// Build DELETE statement
	plsqlBuilder.WriteString("  DELETE FROM ")
	db.QuoteTo(&plsqlBuilder, stmt.Table)

	// Add WHERE clause if it exists
	if whereClause, hasWhere := stmt.Clauses["WHERE"]; hasWhere {
		plsqlBuilder.WriteString(" WHERE ")
		if where, ok := whereClause.Expression.(clause.Where); ok {
			buildWhereClause(db, &plsqlBuilder, where.Exprs)
		}
	}

	// Add RETURNING clause
	plsqlBuilder.WriteString("\n  RETURNING ")
	allColumns := getAllTableColumns(schema)
	for i, column := range allColumns {
		if i > 0 {
			plsqlBuilder.WriteString(", ")
		}
		db.QuoteTo(&plsqlBuilder, column)

	}
	plsqlBuilder.WriteString("\n  BULK COLLECT INTO l_deleted_records;\n")

	// Create OUT parameters for each field and each row that will be deleted
	outParamIndex := len(stmt.Vars)
	//TODO make it configurable
	estimatedRows := 100 // Estimate maximum rows to delete

	for rowIdx := 0; rowIdx < estimatedRows; rowIdx++ {
		for _, column := range allColumns {
			field := findFieldByDBName(schema, column)
			if field != nil {
				dest := createTypedDestination(field)
				stmt.Vars = append(stmt.Vars, sql.Out{Dest: dest})

				plsqlBuilder.WriteString(fmt.Sprintf("  IF l_deleted_records.COUNT > %d THEN\n", rowIdx))
				plsqlBuilder.WriteString(fmt.Sprintf("    :%d := l_deleted_records(%d).", outParamIndex+1, rowIdx+1))
				db.QuoteTo(&plsqlBuilder, column)
				plsqlBuilder.WriteString(";\n")
				plsqlBuilder.WriteString("  END IF;\n")
				outParamIndex++
			}
		}
	}
	plsqlBuilder.WriteString("END;")

	stmt.SQL.Reset()
	stmt.SQL.WriteString(plsqlBuilder.String())
}

// Build WHERE clause for DELETE operations
func buildWhereClause(db *gorm.DB, plsqlBuilder *strings.Builder, expressions []clause.Expression) {
	stmt := db.Statement

	for i, expr := range expressions {
		if i > 0 {
			plsqlBuilder.WriteString(" AND ")
		}

		// Handle different expression types
		switch e := expr.(type) {
		case clause.Eq:
			// Write the column name
			if columnName, ok := e.Column.(string); ok {
				db.QuoteTo(plsqlBuilder, columnName)
			} else if columnExpr, ok := e.Column.(clause.Column); ok {
				db.QuoteTo(plsqlBuilder, columnExpr.Name)
			} else {
				plsqlBuilder.WriteString(fmt.Sprintf("%v", e.Column))
			}

			// Check if the value is NULL and handle appropriately
			if isNullValue(e.Value) {
				plsqlBuilder.WriteString(" IS NULL")
				// Don't add the value to stmt.Vars for IS NULL
			} else {
				plsqlBuilder.WriteString(fmt.Sprintf(" = :%d", len(stmt.Vars)+1))
				stmt.Vars = append(stmt.Vars, convertValue(e.Value))
			}

		case clause.IN:
			if columnName, ok := e.Column.(string); ok {
				db.QuoteTo(plsqlBuilder, columnName)
			} else if columnExpr, ok := e.Column.(clause.Column); ok {
				db.QuoteTo(plsqlBuilder, columnExpr.Name)
			} else {
				plsqlBuilder.WriteString(fmt.Sprintf("%v", e.Column))
			}
			plsqlBuilder.WriteString(" IN (")
			for j, val := range e.Values {
				if j > 0 {
					plsqlBuilder.WriteString(", ")
				}
				plsqlBuilder.WriteString(fmt.Sprintf(":%d", len(stmt.Vars)+1))
				stmt.Vars = append(stmt.Vars, convertValue(val))
			}
			plsqlBuilder.WriteString(")")

		case clause.Expr:
			buildExpressionClause(db, plsqlBuilder, e)

		default:
			// Safe fallback for unknown expression types
			plsqlBuilder.WriteString("1=1")
		}
	}
}

// Build expression clause with proper handling of IN clauses
func buildExpressionClause(db *gorm.DB, plsqlBuilder *strings.Builder, e clause.Expr) {
	stmt := db.Statement
	exprSQL := e.SQL
	varIndex := 0

	for strings.Contains(exprSQL, "?") {
		if varIndex < len(e.Vars) {
			// Check if this is an IN expression with a slice
			if strings.Contains(exprSQL, "IN ?") && varIndex < len(e.Vars) {
				// Handle different slice types
				var values []interface{}

				switch v := e.Vars[varIndex].(type) {
				case []interface{}:
					values = v
				case []string:
					// Convert []string to []interface{}
					values = make([]interface{}, len(v))
					for i, s := range v {
						values[i] = s
					}
				default:
					// Fall back to regular parameter replacement
					exprSQL = strings.Replace(exprSQL, "?", fmt.Sprintf(":%d", len(stmt.Vars)+1), 1)
					stmt.Vars = append(stmt.Vars, convertValue(e.Vars[varIndex]))
					varIndex++
					continue
				}

				if len(values) > 0 {
					// Handle IN clause with multiple values
					inClause := "("
					for j, val := range values {
						if j > 0 {
							inClause += ", "
						}
						inClause += fmt.Sprintf(":%d", len(stmt.Vars)+1)
						stmt.Vars = append(stmt.Vars, convertValue(val))
					}
					inClause += ")"
					exprSQL = strings.Replace(exprSQL, "?", inClause, 1)
					varIndex++
				}
			} else {
				// Regular parameter replacement
				exprSQL = strings.Replace(exprSQL, "?", fmt.Sprintf(":%d", len(stmt.Vars)+1), 1)
				stmt.Vars = append(stmt.Vars, convertValue(e.Vars[varIndex]))
				varIndex++
			}
		} else {
			break
		}
	}

	plsqlBuilder.WriteString(exprSQL)
}

// Execute the delete statement
func executeDelete(db *gorm.DB) {
	if db.DryRun || db.Error != nil {
		return
	}

	stmt := db.Statement

	// Build SQL if not already built
	if stmt.SQL.Len() == 0 {
		stmt.Build("DELETE", "FROM", "WHERE")

		// Convert values for Oracle
		for i, val := range stmt.Vars {
			stmt.Vars[i] = convertValue(val)
		}
	}

	// Check if we have RETURNING clause to determine execution method
	_, hasReturning := stmt.Clauses["RETURNING"]

	if hasReturning {
		// Hard delete & soft delete with RETURNING - use ExecContext (for PL/SQL blocks)
		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if err == nil {
			db.RowsAffected, _ = result.RowsAffected()

			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
			getDeleteBulkReturningValues(db)
		} else {
			db.AddError(err)
		}
	} else {
		// Use ExecContext for regular DELETE
		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if err == nil {
			db.RowsAffected, _ = result.RowsAffected()

			// Set result information like default callback
			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
		} else {
			db.AddError(err)
		}
	}
}

// Handle bulk RETURNING results for DELETE operations
func getDeleteBulkReturningValues(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}

	targetValue := db.Statement.ReflectValue
	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	if targetValue.Kind() != reflect.Slice {
		return
	}

	// Find first OUT parameter
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

	allColumns := getAllTableColumns(db.Statement.Schema)

	// Count OUT parameters and calculate max rows
	outParamCount := 0
	for i := actualStartIndex; i < len(db.Statement.Vars); i++ {
		if _, ok := db.Statement.Vars[i].(sql.Out); ok {
			outParamCount++
		}
	}

	maxPossibleRows := outParamCount / len(allColumns)
	var validRows []reflect.Value

	// Process rows and collect valid data
	for rowIdx := 0; rowIdx < maxPossibleRows; rowIdx++ {
		targetStruct := reflect.New(targetValue.Type().Elem()).Elem()
		hasRealData := false

		for colIdx, column := range allColumns {
			paramIndex := actualStartIndex + (rowIdx * len(allColumns)) + colIdx

			if paramIndex < len(db.Statement.Vars) {
				if outParam, ok := db.Statement.Vars[paramIndex].(sql.Out); ok {
					if field := findFieldByDBName(db.Statement.Schema, column); field != nil && outParam.Dest != nil {
						destValue := reflect.ValueOf(outParam.Dest)
						if destValue.Kind() == reflect.Ptr && !destValue.IsNil() {
							actualValue := destValue.Elem().Interface()

							if actualValue != nil && !reflect.ValueOf(actualValue).IsZero() {
								hasRealData = true
							}

							if convertedValue := convertFromOracleToField(actualValue, field); convertedValue != nil {
								if err := field.Set(db.Statement.Context, targetStruct, convertedValue); err != nil {
									db.AddError(fmt.Errorf("failed to set field %s: %w", field.Name, err))
								}
							}
						}
					}
				}
			}
		}

		if hasRealData {
			validRows = append(validRows, targetStruct)
		}
	}

	// Set final results
	if len(validRows) > 0 {
		newSlice := reflect.MakeSlice(targetValue.Type(), len(validRows), len(validRows))
		for i, row := range validRows {
			newSlice.Index(i).Set(row)
		}
		targetValue.Set(newSlice)
	} else {
		targetValue.Set(reflect.MakeSlice(targetValue.Type(), 0, 0))
	}
}
