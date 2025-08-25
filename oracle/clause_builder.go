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
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ClauseInsert     = "INSERT"
	ClauseUpdate     = "UPDATE"
	ClauseDelete     = "DELETE"
	ClauseLimit      = "LIMIT"
	ClauseOnConflict = "ON CONFLICT"
	ClauseValues     = "VALUES"
	ClauseReturning  = "RETURNING"
)

// Returns the clause builders that are used to generate clauses for Oracle DB
func OracleClauseBuilders() map[string]clause.ClauseBuilder {
	return map[string]clause.ClauseBuilder{
		ClauseInsert:     InsertClauseBuilder,
		ClauseUpdate:     UpdateClauseBuilder,
		ClauseDelete:     DeleteClauseBuilder,
		ClauseLimit:      LimitClauseBuilder,
		ClauseOnConflict: OnConflictClauseBuilder,
		ClauseValues:     ValuesClauseBuilder,
		ClauseReturning:  ReturningClauseBuilder,
	}
}

// InsertClauseBuilder builds the INSERT INTO cluase
func InsertClauseBuilder(c clause.Clause, builder clause.Builder) {

	if insert, ok := c.Expression.(clause.Insert); ok {
		builder.WriteString("INSERT INTO ")

		// If the table name is empty in the clause, get it from the statement
		if insert.Table.Name == "" {
			if stmt, ok := builder.(*gorm.Statement); ok {
				builder.WriteQuoted(stmt.Table)
			}
		} else {
			builder.WriteQuoted(insert.Table)
		}
	}
	// Modifier field is intentionally ignored for Oracle
}

// UpdateClauseBuilder builds the UPDATE clause
func UpdateClauseBuilder(c clause.Clause, builder clause.Builder) {
	if update, ok := c.Expression.(clause.Update); ok {
		builder.WriteString("UPDATE ")

		// If the table name is empty in the clause, get it from the statement
		if update.Table.Name == "" {
			if stmt, ok := builder.(*gorm.Statement); ok {
				builder.WriteQuoted(stmt.Table)
			}
		} else {
			builder.WriteQuoted(update.Table)
		}
	}
	// Modifier field is intentionally ignored for Oracle
}

// DeleteClauseBuilder builds the DELETE clause
func DeleteClauseBuilder(c clause.Clause, builder clause.Builder) {
	if _, ok := c.Expression.(clause.Delete); ok {
		builder.WriteString("DELETE")
	}
	// Modifier field is intentionally ignored for Oracle
}

// Enhanced ReturningClauseBuilder that handles both RETURNING and INTO
func ReturningClauseBuilder(c clause.Clause, builder clause.Builder) {
	if returning, ok := c.Expression.(clause.Returning); ok {
		// Handle the case where RETURNING clause is empty - populate it with all columns
		if len(returning.Columns) == 0 {
			if stmt, ok := builder.(*gorm.Statement); ok && stmt.Schema != nil {
				// Auto-populate with all columns
				var returningColumns []clause.Column
				for _, field := range stmt.Schema.Fields {
					if field.DBName != "" {
						returningColumns = append(returningColumns, clause.Column{Name: field.DBName})
					}
				}

				if len(returningColumns) > 0 {
					// Update the local returning variable directly instead of trying to update the statement
					returning = clause.Returning{Columns: returningColumns}

					// Also update the clause in the statement for later use
					stmt.Clauses["RETURNING"] = clause.Clause{
						Name:       "RETURNING",
						Expression: returning,
					}
				}
			}
		}

		// Only build RETURNING clause if we have columns
		if len(returning.Columns) > 0 {
			builder.WriteString("RETURNING ")
			for idx, column := range returning.Columns {
				if idx > 0 {
					builder.WriteByte(',')
				}
				builder.WriteQuoted(column)
			}

			// Handle the INTO part here
			if stmt, ok := builder.(*gorm.Statement); ok {
				builder.WriteString(" INTO ")

				// Add sql.Out parameters for each returning column
				for idx, column := range returning.Columns {
					if idx > 0 {
						builder.WriteByte(',')
					}

					// Find the field by column name and create appropriate destination
					var dest interface{}
					if stmt.Schema != nil {
						if field := findFieldByDBName(stmt.Schema, column.Name); field != nil {
							dest = createTypedDestination(field)
						} else {
							dest = new(string) // Default to string for unknown fields
						}
					} else {
						dest = new(string) // Default to string if no schema
					}

					stmt.AddVar(stmt, sql.Out{Dest: dest})
				}
			}
		}
	}
}

// LimitClauseBuilder builds the Oracle FETCH clause instead of using the default LIMIT syntax
// The FETCH syntax is supported in Oracle 12c and later
func LimitClauseBuilder(c clause.Clause, builder clause.Builder) {
	if limit, ok := c.Expression.(clause.Limit); ok {
		// Convert LIMIT to Oracle FETCH syntax
		if stmt, ok := builder.(*gorm.Statement); ok {
			buildOracleFetchLimit(limit, builder, stmt)
		}
	}
}

// ValuesClauseBuilder builds the VALUES clause of an INSERT statement
func ValuesClauseBuilder(c clause.Clause, builder clause.Builder) {
	if values, ok := c.Expression.(clause.Values); ok {
		if len(values.Columns) > 0 {
			// Standard case: (col1, col2) VALUES (val1, val2)
			builder.WriteByte('(')
			for idx, column := range values.Columns {
				if idx > 0 {
					builder.WriteByte(',')
				}
				builder.WriteQuoted(column)
			}
			builder.WriteByte(')')

			builder.WriteString(" VALUES ")

			for idx, value := range values.Values {
				if idx > 0 {
					builder.WriteByte(',')
				}

				builder.WriteByte('(')
				builder.AddVar(builder, value...)
				builder.WriteByte(')')
			}
		} else {
			// Default case: (col1, col2) VALUES (DEFAULT, DEFAULT)
			if stmt, ok := builder.(*gorm.Statement); ok {
				if stmt.Schema != nil && len(stmt.Schema.Fields) > 0 {
					builder.WriteString("VALUES (")
					for idx := range stmt.Schema.Fields {
						if idx > 0 {
							builder.WriteByte(',')
						}
						builder.WriteString("DEFAULT")
					}
					builder.WriteByte(')')
				} else {
					// Error: Can't convert DEFAULT VALUES without schema information
					stmt.AddError(fmt.Errorf("Oracle doesn't support 'DEFAULT VALUES' syntax. Cannot convert to 'VALUES (DEFAULT, DEFAULT, ...)' without table schema information. Please ensure schema is available"))
				}
			}
		}
	}
}

// buildOracleFetchLimit builds Oracle FETCH clause (Oracle 12c+)
func buildOracleFetchLimit(limit clause.Limit, builder clause.Builder, stmt *gorm.Statement) {
	// Check if ORDER BY exists when we have LIMIT/OFFSET
	hasLimit := limit.Limit != nil && *limit.Limit >= 0
	hasOffset := limit.Offset > 0

	if hasLimit || hasOffset {
		if _, hasOrderBy := stmt.Clauses["ORDER BY"]; !hasOrderBy {
			builder.WriteString("ORDER BY ")
			if stmt.Schema != nil && stmt.Schema.PrioritizedPrimaryField != nil {
				builder.WriteQuoted(stmt.Schema.PrioritizedPrimaryField.DBName)
				builder.WriteString(" ")
			} else {
				builder.WriteString("1 ")
			}
		}
	}

	// Build OFFSET clause if specified
	if hasOffset {
		builder.WriteString("OFFSET ")
		builder.WriteString(strconv.Itoa(limit.Offset))
		if limit.Offset == 1 {
			builder.WriteString(" ROW ")
		} else {
			builder.WriteString(" ROWS ")
		}
	}

	// Build FETCH clause if limit is specified
	if hasLimit {
		builder.WriteString("FETCH NEXT ")
		builder.WriteString(strconv.Itoa(*limit.Limit))
		if *limit.Limit == 1 {
			builder.WriteString(" ROW ONLY")
		} else {
			builder.WriteString(" ROWS ONLY")
		}
	}
}

// OnConflictClauseBuilder builds MERGE statement directly
func OnConflictClauseBuilder(c clause.Clause, builder clause.Builder) {
	if onConflict, ok := c.Expression.(clause.OnConflict); ok {
		if stmt, ok := builder.(*gorm.Statement); ok {
			// Get the VALUES clause to build the USING subquery
			valuesClause, hasValues := stmt.Clauses["VALUES"]
			if !hasValues {
				stmt.AddError(fmt.Errorf("MERGE requires VALUES clause"))
				return
			}

			values, ok := valuesClause.Expression.(clause.Values)
			if !ok {
				stmt.AddError(fmt.Errorf("invalid VALUES clause for MERGE"))
				return
			}

			// Determine conflict columns
			conflictColumns := onConflict.Columns
			if len(conflictColumns) == 0 {
				// If no columns specified, try to use primary key fields as default
				if stmt.Schema == nil || len(stmt.Schema.PrimaryFields) == 0 {
					return
				}
				for _, primaryField := range stmt.Schema.PrimaryFields {
					conflictColumns = append(conflictColumns, clause.Column{Name: primaryField.DBName})
				}
			}

			// Validate that we actually need to use MERGE statement
			shouldUseMerge := ShouldUseRealConflict(values, onConflict, conflictColumns)
			if !shouldUseMerge {
				return // Leave the statement as-is (regular INSERT with RETURNING)
			}

			// Validate that all conflict columns exist in the VALUES clause
			// Use Map to optimize the performance
			selectedColumnSet := make(map[string]struct{}, len(values.Columns))
			for _, col := range values.Columns {
				selectedColumnSet[strings.ToLower(col.Name)] = struct{}{}
			}
			var missingColumns []string
			for _, conflictCol := range conflictColumns {
				if _, found := selectedColumnSet[strings.ToLower(conflictCol.Name)]; !found {
					missingColumns = append(missingColumns, conflictCol.Name)
				}
			}
			if len(missingColumns) > 0 {
				var selectedColumns []string
				for col := range selectedColumnSet {
					selectedColumns = append(selectedColumns, col)
				}
				stmt.AddError(fmt.Errorf("conflict columns %v are not present in the VALUES clause. Available columns: %v",
					missingColumns, selectedColumns))
				return
			}

			// exclude primary key, default value columns from merge update clause
			if len(onConflict.DoUpdates) > 0 {
				hasPrimaryKey := false

				for _, assignment := range onConflict.DoUpdates {
					field := stmt.Schema.LookUpField(assignment.Column.Name)
					if field != nil && field.PrimaryKey {
						hasPrimaryKey = true
						break
					}
				}

				if hasPrimaryKey {
					onConflict.DoUpdates = nil
					columns := make([]string, 0, len(values.Columns)-1)
					for _, col := range values.Columns {
						field := stmt.Schema.LookUpField(col.Name)

						if field != nil && !field.PrimaryKey && (!field.HasDefaultValue || field.DefaultValueInterface != nil ||
							strings.EqualFold(field.DefaultValue, "NULL")) && field.AutoCreateTime == 0 {
							columns = append(columns, col.Name)
						}

					}
					onConflict.DoUpdates = append(onConflict.DoUpdates, clause.AssignmentColumns(columns)...)
				}
			}

			// Build MERGE statement
			buildMergeInClause(stmt, onConflict, values, conflictColumns)
		}
	}
}

// Helper method to determine if we need MERGE
func ShouldUseRealConflict(values clause.Values, onConflict clause.OnConflict, conflictColumns []clause.Column) bool {
	var valuesColumns []string
	for _, column := range values.Columns {
		valuesColumns = append(valuesColumns, column.Name)
	}

	// Use the updated conflict columns instead of the original one (onConflict.Columns)
	meaningfulColumns := 0
	for _, column := range conflictColumns {
		// Check if this column is in VALUES
		for _, valCol := range valuesColumns {
			if strings.EqualFold(valCol, column.Name) {
				meaningfulColumns++
				break
			}
		}
	}

	// Only use MERGE if we have meaningful columns to conflict on
	if meaningfulColumns == 0 {
		return false
	}

	// If we have meaningful columns and any conflict action, use MERGE
	hasConflictAction := len(onConflict.DoUpdates) > 0 || onConflict.DoNothing
	result := hasConflictAction
	return result
}

// Build MERGE statement directly in the clause builder
func buildMergeInClause(stmt *gorm.Statement, onConflict clause.OnConflict, values clause.Values, conflictColumns []clause.Column) {
	dummyTable := getDummyTable()

	// Clear any existing SQL and start fresh with MERGE
	stmt.SQL.Reset()
	stmt.Vars = stmt.Vars[:0] // Clear variables

	// MERGE INTO table_name
	stmt.WriteString("MERGE INTO ")
	stmt.WriteQuoted(stmt.Table)
	stmt.WriteString(" USING (")

	// Build the USING subquery with UNION ALL
	for idx, value := range values.Values {
		if idx > 0 {
			stmt.WriteString(" UNION ALL ")
		}

		stmt.WriteString("SELECT ")
		for i, v := range value {
			if i > 0 {
				stmt.WriteByte(',')
			}
			column := values.Columns[i]
			stmt.AddVar(stmt, v)
			stmt.WriteString(" AS ")
			stmt.WriteQuoted(column.Name)
		}
		stmt.WriteString(" FROM ")
		stmt.WriteString(dummyTable)
	}

	// Close USING and add alias
	stmt.WriteString(") ")
	stmt.WriteQuoted(clause.Table{Name: "excluded"})
	stmt.WriteString(" ON (")

	// Build ON condition using the conflict columns
	for idx, column := range conflictColumns {
		if idx > 0 {
			stmt.WriteString(" AND ")
		}
		stmt.WriteQuoted(clause.Column{Table: stmt.Table, Name: column.Name})
		stmt.WriteString(" = ")
		stmt.WriteQuoted(clause.Column{Table: "excluded", Name: column.Name})
	}
	stmt.WriteByte(')')

	// WHEN MATCHED THEN UPDATE SET (if provided)
	if len(onConflict.DoUpdates) > 0 {
		stmt.WriteString(" WHEN MATCHED THEN UPDATE SET ")
		onConflict.DoUpdates.Build(stmt)
	}

	// WHEN NOT MATCHED THEN INSERT
	stmt.WriteString(" WHEN NOT MATCHED THEN INSERT (")

	// Add column names (excluding auto-increment primary key)
	written := false
	for _, column := range values.Columns {
		if shouldIncludeColumn(stmt, column.Name) {
			if written {
				stmt.WriteByte(',')
			}
			written = true
			stmt.WriteQuoted(column)
		}
	}

	stmt.WriteString(") VALUES (")

	// Add values (excluding auto-increment primary key)
	written = false
	for _, column := range values.Columns {
		if shouldIncludeColumn(stmt, column.Name) {
			if written {
				stmt.WriteByte(',')
			}
			written = true
			stmt.WriteQuoted(clause.Column{
				Table: "excluded",
				Name:  column.Name,
			})
		}
	}
	stmt.WriteByte(')')
}

// Check if column should be included (exclude auto-increment primary keys)
func shouldIncludeColumn(stmt *gorm.Statement, columnName string) bool {
	if stmt.Schema.PrioritizedPrimaryField != nil &&
		stmt.Schema.PrioritizedPrimaryField.AutoIncrement &&
		stmt.Schema.PrioritizedPrimaryField.DBName == columnName {
		return false
	}
	return true
}

func getDummyTable() string {
	return "DUAL"
}
