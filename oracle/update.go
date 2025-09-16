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
	"sort"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Update overrides GORM's update callback for Oracle.
//
// It builds Oracle-compatible UPDATE statements and supports:
//
//   - Standard updates without RETURNING using GORM’s default SQL build, with
//     bind variable conversion for Oracle types.
//   - Updates with RETURNING, emitting a PL/SQL block that performs
//     UPDATE … RETURNING BULK COLLECT INTO for multi-row updates,
//     wiring sql.Out binds for each returned column and row.
//   - UPDATE safety: checks for missing WHERE conditions and refuses to run
//     unless AllowGlobalUpdate is set or the WHERE clause has meaningful
//     conditions (beyond soft-delete filters).
//   - Primary key WHERE injection when the destination object or slice has
//     identifiable PK values, to avoid unintended mass updates.
//   - Soft-delete compatibility: conditions on deleted_at are ignored for the
//     safety check, but preserved in the WHERE for the actual SQL.
//
// For updates with RETURNING, OUT bind results are mapped back into the
// destination struct or slice using getUpdateReturningValues.
//
// Register with:
//
//	db.Callback().Update().Replace("gorm:update", oracle.Update)
func Update(db *gorm.DB) {
	if db.Error != nil || db.Statement == nil {
		return
	}

	stmt := db.Statement

	if stmt.Schema != nil {
		for _, c := range stmt.Schema.UpdateClauses {
			stmt.AddClause(c)
		}
	}

	if stmt.SQL.Len() == 0 {
		stmt.SQL.Grow(180)
		stmt.AddClauseIfNotExists(clause.Update{})

		// Build SET clause if not exists
		if _, ok := stmt.Clauses["SET"]; !ok {
			if set := convertToUpdateAssignments(stmt); len(set) != 0 {
				defer delete(stmt.Clauses, "SET")
				stmt.AddClause(set)
			} else {
				// No assignments to make, return early
				return
			}
		}

		// Check if we need RETURNING clause
		_, hasReturning := stmt.Clauses["RETURNING"]
		needsReturning := stmt.Schema != nil && hasReturning

		if needsReturning {
			// Always use PL/SQL for RETURNING, just like delete callback
			buildUpdatePLSQL(db)
		} else {
			// Use GORM's standard build for non-RETURNING updates
			stmt.Build("UPDATE", "SET", "WHERE")
			// Convert values for Oracle
			for i, val := range stmt.Vars {
				stmt.Vars[i] = convertValue(val)
			}
		}

		// Safety check for missing WHERE conditions
		checkMissingWhereConditions(db)
		if db.Error != nil {
			return
		}
	}

	executeUpdate(db)
}

// Check for missing WHERE conditions (safety check)
func checkMissingWhereConditions(db *gorm.DB) {
	if db.Error != nil {
		return
	}

	// Skip check if AllowGlobalUpdate is enabled
	if db.AllowGlobalUpdate {
		return
	}

	whereClause, hasWhere := db.Statement.Clauses["WHERE"]

	// If no WHERE clause at all, this is definitely unsafe
	if !hasWhere {
		db.AddError(gorm.ErrMissingWhereClause)
		return
	}

	// Check if the WHERE clause has meaningful conditions
	if where, ok := whereClause.Expression.(clause.Where); ok {
		if len(where.Exprs) > 0 {
			// Track if we found any meaningful (non-soft-delete) conditions
			hasMeaningfulConditions := false

			// Check if the WHERE clause has any meaningful conditions
			for _, expr := range where.Exprs {
				switch e := expr.(type) {
				case clause.Eq:
					// Check if this is a soft delete condition
					if isSoftDeleteCondition(e.Column, e.Value) {
						// This is a soft delete condition, skip it
						continue
					}
					// Has non-soft-delete equality condition, this is valid
					hasMeaningfulConditions = true
					break
				case clause.IN:
					// Has IN condition with values, this is valid
					if len(e.Values) > 0 {
						hasMeaningfulConditions = true
						break
					} else {
						// Empty IN clause means "update/delete nothing" - this is safe
						hasMeaningfulConditions = true
						break
					}
				case clause.Expr:
					// Check if this is just a soft delete condition
					if isSoftDeleteExprCondition(e.SQL) {
						// This is just the soft delete condition, not a real filter
						continue
					}
					// Has non-soft-delete expression condition, consider it valid
					hasMeaningfulConditions = true
					break
				case clause.AndConditions, clause.OrConditions:
					// Complex conditions are likely valid (but we could be more thorough here)
					hasMeaningfulConditions = true
					break
				case clause.Where:
					// Handle nested WHERE clauses - recursively check their expressions
					if len(e.Exprs) > 0 {
						// TODO: recurse here
						hasMeaningfulConditions = true
						break
					}
				case clause.NotConditions:
					// Handle NOT conditions
					if len(e.Exprs) > 0 {
						hasMeaningfulConditions = true
						break
					}
				default:
					// Unknown condition types - assume they're meaningful for safety
					hasMeaningfulConditions = true
					break
				}

				// If we found meaningful conditions, we can stop checking
				if hasMeaningfulConditions {
					break
				}
			}

			// If we found meaningful conditions, allow the operation
			if hasMeaningfulConditions {
				return
			}
		}
	}

	// If we get here, we don't have meaningful WHERE conditions
	db.AddError(gorm.ErrMissingWhereClause)
}

// Helper function to check if a condition is a soft delete condition
func isSoftDeleteCondition(column interface{}, value interface{}) bool {
	// Convert column to string
	var columnStr string
	switch c := column.(type) {
	case string:
		columnStr = c
	case clause.Column:
		columnStr = c.Name
	default:
		return false
	}

	// Check if column name contains "deleted_at" and value is NULL-like
	columnLower := strings.ToLower(columnStr)
	if strings.Contains(columnLower, "deleted_at") {
		// Check if value is nil, NULL, or similar
		return value == nil || isNullValue(value)
	}

	return false
}

// Helper function to check if an expression is a soft delete condition
func isSoftDeleteExprCondition(sql string) bool {
	sqlLower := strings.ToLower(sql)
	// Check for common soft delete patterns
	return strings.Contains(sqlLower, "deleted_at is null") ||
		strings.Contains(sqlLower, "deleted_at is not null") ||
		(strings.Contains(sqlLower, "deleted_at") && strings.Contains(sqlLower, "null"))
}

// Convert to update assignments (adapted from GORM's ConvertToAssignments)
func convertToUpdateAssignments(stmt *gorm.Statement) (set clause.Set) {
	var (
		selectColumns, restricted = stmt.SelectAndOmitColumns(false, true)
		assignValue               func(field *schema.Field, value interface{})
	)

	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		assignValue = func(field *schema.Field, value interface{}) {
			for i := 0; i < stmt.ReflectValue.Len(); i++ {
				if stmt.ReflectValue.CanAddr() {
					field.Set(stmt.Context, stmt.ReflectValue.Index(i), value)
				}
			}
		}
	case reflect.Struct:
		assignValue = func(field *schema.Field, value interface{}) {
			if stmt.ReflectValue.CanAddr() {
				field.Set(stmt.Context, stmt.ReflectValue, value)
			}
		}
	default:
		assignValue = func(field *schema.Field, value interface{}) {
		}
	}

	updatingValue := reflect.ValueOf(stmt.Dest)
	for updatingValue.Kind() == reflect.Ptr {
		updatingValue = updatingValue.Elem()
	}

	// Add WHERE clause based on primary keys if not already present
	if !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
		addPrimaryKeyWhereClauseForUpdate(stmt)
	}

	switch value := updatingValue.Interface().(type) {
	case map[string]interface{}:
		set = make([]clause.Assignment, 0, len(value))

		// Sort keys for consistent behavior
		keys := make([]string, 0, len(value))
		for k := range value {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			kv := value[k]
			if _, ok := kv.(*gorm.DB); ok {
				kv = []interface{}{kv}
			}

			if stmt.Schema != nil {
				if field := stmt.Schema.LookUpField(k); field != nil {
					if field.DBName != "" {
						if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: kv})
							assignValue(field, value[k])
						}
					} else if v, ok := selectColumns[field.Name]; (ok && v) || (!ok && !restricted) {
						assignValue(field, value[k])
					}
					continue
				}
			}

			if v, ok := selectColumns[k]; (ok && v) || (!ok && !restricted) {
				set = append(set, clause.Assignment{Column: clause.Column{Name: k}, Value: kv})
			}
		}

		// Handle auto-update time fields
		if !stmt.SkipHooks && stmt.Schema != nil {
			for _, dbName := range stmt.Schema.DBNames {
				field := stmt.Schema.LookUpField(dbName)
				if field.AutoUpdateTime > 0 && value[field.Name] == nil && value[field.DBName] == nil {
					if v, ok := selectColumns[field.DBName]; (ok && v) || !ok {
						now := stmt.DB.NowFunc()
						assignValue(field, now)

						if field.AutoUpdateTime == schema.UnixNanosecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.UnixNano()})
						} else if field.AutoUpdateTime == schema.UnixMillisecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.UnixMilli()})
						} else if field.AutoUpdateTime == schema.UnixSecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.Unix()})
						} else {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now})
						}
					}
				}
			}
		}

	default:
		updatingSchema := stmt.Schema
		var isDiffSchema bool
		if !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
			// Different schema case
			updatingStmt := &gorm.Statement{DB: stmt.DB}
			if err := updatingStmt.Parse(stmt.Dest); err == nil {
				updatingSchema = updatingStmt.Schema
				isDiffSchema = true
			}
		}

		switch updatingValue.Kind() {
		case reflect.Struct:
			set = make([]clause.Assignment, 0, len(stmt.Schema.FieldsByDBName))
			for _, dbName := range stmt.Schema.DBNames {
				if field := updatingSchema.LookUpField(dbName); field != nil {
					if !field.PrimaryKey || !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
						if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && (!restricted || (!stmt.SkipHooks && field.AutoUpdateTime > 0))) {
							value, isZero := field.ValueOf(stmt.Context, updatingValue)

							// Handle auto-update time fields
							if !stmt.SkipHooks && field.AutoUpdateTime > 0 {
								if field.AutoUpdateTime == schema.UnixNanosecond {
									value = stmt.DB.NowFunc().UnixNano()
								} else if field.AutoUpdateTime == schema.UnixMillisecond {
									value = stmt.DB.NowFunc().UnixMilli()
								} else if field.AutoUpdateTime == schema.UnixSecond {
									value = stmt.DB.NowFunc().Unix()
								} else {
									value = stmt.DB.NowFunc()
								}
								isZero = false
							}

							if (ok || !isZero) && field.Updatable {
								set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: value})
								assignField := field
								if isDiffSchema {
									if originField := stmt.Schema.LookUpField(dbName); originField != nil {
										assignField = originField
									}
								}
								assignValue(assignField, value)
							}
						}
					} else {
						// Add primary key to WHERE clause
						if value, isZero := field.ValueOf(stmt.Context, updatingValue); !isZero {
							stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
						}
					}
				}
			}
		default:
			stmt.AddError(gorm.ErrInvalidData)
		}
	}

	return
}

// Add primary key WHERE clause for update operations
func addPrimaryKeyWhereClauseForUpdate(stmt *gorm.Statement) {
	if stmt.Schema == nil {
		return
	}

	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		if size := stmt.ReflectValue.Len(); size > 0 {
			var isZero bool
			for i := 0; i < size; i++ {
				for _, field := range stmt.Schema.PrimaryFields {
					_, isZero = field.ValueOf(stmt.Context, stmt.ReflectValue.Index(i))
					if !isZero {
						break
					}
				}
			}

			if !isZero {
				_, primaryValues := schema.GetIdentityFieldValuesMap(stmt.Context, stmt.ReflectValue, stmt.Schema.PrimaryFields)
				column, values := schema.ToQueryValues("", stmt.Schema.PrimaryFieldDBNames, primaryValues)
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
			}
		}
	case reflect.Struct:
		for _, field := range stmt.Schema.PrimaryFields {
			if value, isZero := field.ValueOf(stmt.Context, stmt.ReflectValue); !isZero {
				stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
			}
		}
	}
}

// Build PL/SQL block for UPDATE with RETURNING
func buildUpdatePLSQL(db *gorm.DB) {
	stmt := db.Statement
	schema := stmt.Schema

	if schema == nil {
		db.AddError(fmt.Errorf("schema required for update with returning"))
		return
	}

	// Get SET and WHERE clauses
	setClause, hasSet := stmt.Clauses["SET"]
	whereClause, hasWhere := stmt.Clauses["WHERE"]

	if !hasSet {
		db.AddError(fmt.Errorf("SET clause required for update"))
		return
	}

	set, ok := setClause.Expression.(clause.Set)
	if !ok || len(set) == 0 {
		db.AddError(fmt.Errorf("invalid SET clause"))
		return
	}

	var plsqlBuilder strings.Builder

	// Start PL/SQL block
	plsqlBuilder.WriteString("DECLARE\n")
	writeTableRecordCollectionDecl(db, &plsqlBuilder, stmt.Schema.DBNames, stmt.Table)
	plsqlBuilder.WriteString("  l_updated_records t_records;\n")
	plsqlBuilder.WriteString("BEGIN\n")

	// Build UPDATE statement
	plsqlBuilder.WriteString("  UPDATE ")
	db.QuoteTo(&plsqlBuilder, stmt.Table)
	plsqlBuilder.WriteString(" SET ")

	// Add SET assignments - handle both regular values and expressions
	for i, assignment := range set {
		if i > 0 {
			plsqlBuilder.WriteString(", ")
		}
		db.QuoteTo(&plsqlBuilder, assignment.Column.Name)
		plsqlBuilder.WriteString(" = ")

		// Check if the value is a clause.Expr (like gorm.Expr)
		if expr, ok := assignment.Value.(clause.Expr); ok {
			// Handle expressions directly by building them into SQL
			exprSQL := expr.SQL
			varIndex := 0

			// Replace ? placeholders with proper parameter references
			for strings.Contains(exprSQL, "?") && varIndex < len(expr.Vars) {
				exprSQL = strings.Replace(exprSQL, "?", fmt.Sprintf(":%d", len(stmt.Vars)+1), 1)
				stmt.Vars = append(stmt.Vars, convertValue(expr.Vars[varIndex]))
				varIndex++
			}
			plsqlBuilder.WriteString(exprSQL)
		} else {
			// Handle regular values as parameters
			plsqlBuilder.WriteString(fmt.Sprintf(":%d", len(stmt.Vars)+1))
			stmt.Vars = append(stmt.Vars, convertValue(assignment.Value))
		}
	}

	// Add WHERE clause if present
	if hasWhere {
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
	plsqlBuilder.WriteString("\n  BULK COLLECT INTO l_updated_records;\n")

	// Create OUT parameters for each field and each row that might be updated
	outParamStartIndex := len(stmt.Vars)
	//TODO make it configurable
	estimatedRows := 100 // Estimate maximum rows to update (same as DELETE)

	// First, create all OUT parameters
	for rowIdx := 0; rowIdx < estimatedRows; rowIdx++ {
		for _, column := range allColumns {
			field := findFieldByDBName(schema, column)
			if field != nil {
				var dest interface{}
				if isJSONField(field) {
					dest = new(string) // JSON comes back serialized as text
				} else {
					dest = createTypedDestination(field)
				}
				stmt.Vars = append(stmt.Vars, sql.Out{Dest: dest})
			}
		}
	}

	// Then, generate PL/SQL assignments with correct parameter indices
	for rowIdx := 0; rowIdx < estimatedRows; rowIdx++ {
		for colIdx, column := range allColumns {
			field := findFieldByDBName(schema, column)
			if field != nil {
				paramIndex := outParamStartIndex + (rowIdx * len(allColumns)) + colIdx + 1

				plsqlBuilder.WriteString(fmt.Sprintf("  IF l_updated_records.COUNT > %d THEN ", rowIdx))
				plsqlBuilder.WriteString(fmt.Sprintf(":%d := ", paramIndex))

				if isJSONField(field) {
					// serialize JSON so it binds as text
					plsqlBuilder.WriteString("JSON_SERIALIZE(")
					plsqlBuilder.WriteString(fmt.Sprintf("l_updated_records(%d).", rowIdx+1))
					writeQuotedIdentifier(&plsqlBuilder, column)
					plsqlBuilder.WriteString(" RETURNING CLOB)")
				} else {
					plsqlBuilder.WriteString(fmt.Sprintf("l_updated_records(%d).", rowIdx+1))
					writeQuotedIdentifier(&plsqlBuilder, column)
				}

				plsqlBuilder.WriteString("; END IF;\n")
			}
		}
	}

	plsqlBuilder.WriteString("END;")

	stmt.SQL.Reset()
	stmt.SQL.WriteString(plsqlBuilder.String())
}

// Execute the update statement
func executeUpdate(db *gorm.DB) {
	if db.DryRun || db.Error != nil {
		return
	}

	stmt := db.Statement

	// Check if we have RETURNING clause
	_, hasReturning := stmt.Clauses["RETURNING"]

	if hasReturning {
		// Always use ExecContext for PL/SQL blocks with RETURNING
		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)

		if err == nil {
			db.RowsAffected, _ = result.RowsAffected()
			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
			// Process RETURNING values using the same logic as delete
			getUpdateReturningValues(db)
		} else {
			db.AddError(err)
		}
	} else {
		// Regular UPDATE without RETURNING - use standard GORM execution path
		result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
		if err == nil {
			db.RowsAffected, _ = result.RowsAffected()
			if stmt.Result != nil {
				stmt.Result.Result = result
				stmt.Result.RowsAffected = db.RowsAffected
			}
		} else {
			db.AddError(err)
		}
	}
}

// Handle UPDATE RETURNING results (based on delete callback pattern)
func getUpdateReturningValues(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}

	targetValue := db.Statement.ReflectValue

	if targetValue.Kind() == reflect.Ptr {
		targetValue = targetValue.Elem()
	}

	// Handle both single struct and slice cases
	isSlice := targetValue.Kind() == reflect.Slice
	isSingleStruct := targetValue.Kind() == reflect.Struct

	if !isSlice && !isSingleStruct {
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

	if len(allColumns) == 0 {
		return
	}
	if isSingleStruct {
		// For single struct, process row 0 only
		rowIdx := 0

		for colIdx, column := range allColumns {
			paramIndex := actualStartIndex + (rowIdx * len(allColumns)) + colIdx

			if paramIndex >= len(db.Statement.Vars) {
				continue
			}

			outParam, ok := db.Statement.Vars[paramIndex].(sql.Out)
			if !ok || outParam.Dest == nil {
				continue
			}

			destValue := reflect.ValueOf(outParam.Dest)
			if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
				continue
			}

			actualValue := destValue.Elem().Interface()

			// Find the field and set it directly on the target struct
			field := findFieldByDBName(db.Statement.Schema, column)
			if field == nil {
				continue
			}

			// Convert and set the value directly on the target struct
			convertedValue := convertFromOracleToField(actualValue, field)

			if convertedValue != nil {
				if err := field.Set(db.Statement.Context, targetValue, convertedValue); err != nil {
					db.AddError(fmt.Errorf("failed to set field %s: %w", field.Name, err))
				}
			}
		}

		return
	}

	// Count OUT parameters and calculate max rows
	outParamCount := 0
	for i := actualStartIndex; i < len(db.Statement.Vars); i++ {
		if _, ok := db.Statement.Vars[i].(sql.Out); ok {
			outParamCount++
		}
	}

	maxPossibleRows := outParamCount / len(allColumns)

	if maxPossibleRows == 0 {
		return
	}

	var validRows []reflect.Value

	// Process rows and collect valid data
	for rowIdx := 0; rowIdx < maxPossibleRows; rowIdx++ {
		var targetElement reflect.Value
		hasRealData := false

		// Determine if we're working with structs or maps
		elementType := targetValue.Type().Elem()

		if elementType.Kind() == reflect.Map {
			// Create a new map for this row
			targetElement = reflect.MakeMap(elementType)
		} else {
			// Create a new struct for this row
			targetElement = reflect.New(elementType).Elem()
		}

		for colIdx, column := range allColumns {
			paramIndex := actualStartIndex + (rowIdx * len(allColumns)) + colIdx

			if paramIndex >= len(db.Statement.Vars) {
				continue
			}

			outParam, ok := db.Statement.Vars[paramIndex].(sql.Out)
			if !ok {
				continue
			}

			if outParam.Dest == nil {
				continue
			}

			destValue := reflect.ValueOf(outParam.Dest)

			if destValue.Kind() != reflect.Ptr || destValue.IsNil() {
				continue
			}

			actualValue := destValue.Elem().Interface()

			if !reflect.ValueOf(actualValue).IsZero() {
				hasRealData = true
			}
			// Find the field in the schema
			field := findFieldByDBName(db.Statement.Schema, column)
			if field == nil {
				continue
			}

			// Convert and set the value
			convertedValue := convertFromOracleToField(actualValue, field)

			if convertedValue != nil {

				// Handle both map and struct cases
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

		if hasRealData {
			validRows = append(validRows, targetElement)
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
