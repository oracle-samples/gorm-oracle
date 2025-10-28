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
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

type Migrator struct {
	migrator.Migrator
}

// RunWithValue runs migration for the given `value`
func (m Migrator) RunWithValue(value interface{}, fc func(*gorm.Statement) error) error {
	if table, ok := value.(string); ok {
		return m.Migrator.RunWithValue(table, fc)
	}
	return m.Migrator.RunWithValue(value, fc)
}

// CurrentDatabase returns the the name of the current Oracle database
func (m Migrator) CurrentDatabase() string {
	var name string
	m.DB.Raw("SELECT ora_database_name FROM dual").Scan(&name)
	return name
}

// CreateTable creates table in database for the given `values`
func (m Migrator) CreateTable(values ...interface{}) error {

	for _, value := range m.ReorderModels(values, false) {
		tx := m.DB.Session(&gorm.Session{})
		if err := m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {

			if stmt.Schema == nil {
				return errors.New("failed to get schema")
			}

			var (
				createTableSQL          = "CREATE TABLE ? ("
				values                  = []interface{}{m.CurrentTable(stmt)}
				hasPrimaryKeyInDataType bool
				// Stores the columns that have been referenced by foreign key constraints
				// The key is a slice containing the joined foreign key columns, referenced table, and referenced column
				fkReferenceMap = make(map[[3]string]bool)
			)

			for _, dbName := range stmt.Schema.DBNames {
				field := stmt.Schema.FieldsByDBName[dbName]
				if !field.IgnoreMigration {
					createTableSQL += "? ?"
					hasPrimaryKeyInDataType = hasPrimaryKeyInDataType || strings.Contains(m.DataTypeOf(field), "PRIMARY KEY")
					values = append(values, clause.Column{Name: dbName}, m.DB.Migrator().FullDataTypeOf(field))
					createTableSQL += ","
				}
			}

			if !hasPrimaryKeyInDataType && len(stmt.Schema.PrimaryFields) > 0 {
				createTableSQL += "PRIMARY KEY ?,"
				primaryKeys := make([]interface{}, 0, len(stmt.Schema.PrimaryFields))
				for _, field := range stmt.Schema.PrimaryFields {
					primaryKeys = append(primaryKeys, clause.Column{Name: field.DBName})
				}

				values = append(values, primaryKeys)
			}

			for _, idx := range stmt.Schema.ParseIndexes() {
				if m.CreateIndexAfterCreateTable {
					defer func(value interface{}, name string) {
						if err == nil {
							err = tx.Migrator().CreateIndex(value, name)
						}
					}(value, idx.Name)
				} else {
					if idx.Class != "" {
						createTableSQL += idx.Class + " "
					}
					createTableSQL += "INDEX ? ?"

					if idx.Comment != "" {
						createTableSQL += fmt.Sprintf(" COMMENT '%s'", idx.Comment)
					}

					if idx.Option != "" {
						createTableSQL += " " + idx.Option
					}

					createTableSQL += ","
					values = append(values, clause.Column{Name: idx.Name}, tx.Migrator().(migrator.BuildIndexOptionsInterface).BuildIndexOptions(idx.Fields, stmt))
				}
			}

			if !m.DB.DisableForeignKeyConstraintWhenMigrating && !m.DB.IgnoreRelationshipsWhenMigrating {
				for _, rel := range stmt.Schema.Relationships.Relations {
					if rel.Field.IgnoreMigration {
						continue
					}
					if constraint := rel.ParseConstraint(); constraint != nil {
						if constraint.Schema == stmt.Schema {
							// Oracle doesn’t support OnUpdate on foreign keys.
							// Use a trigger instead to propagate the update to the child table instead.
							if len(constraint.References) > 0 && constraint.OnUpdate != "" {
								defer func(tx *gorm.DB, table string, constraint *schema.Constraint, onUpdate string) {
									if err == nil {
										// retore the OnUpdate value
										constraint.OnUpdate = onUpdate
										err = m.createUpadateCascadeTrigger(tx, constraint)
									}
								}(tx, stmt.Table, constraint, constraint.OnUpdate)
								constraint.OnUpdate = ""
							}

							// If the same set of foreign keys already references the parent column,
							// remove duplicates to avoid ORA-02274: duplicate referential constraint specifications
							var foreignKeys []string
							for _, fk := range constraint.ForeignKeys {
								foreignKeys = append(foreignKeys, fk.DBName)
							}
							jointForeignKeys := strings.Join(foreignKeys, ",")

							for i, ref := range constraint.References {
								refTableColumn := [3]string{jointForeignKeys, constraint.ReferenceSchema.Table, ref.DBName}
								if fkReferenceMap[refTableColumn] {
									// If the target column has already been referenced, remove it from the constraints
									constraint.References = slices.Delete(constraint.References, i, i+1)
								} else {
									fkReferenceMap[refTableColumn] = true
								}
							}

							// Don't build the SQL string when there's no reference target
							if len(constraint.References) > 0 {
								sql, vars := constraint.Build()
								createTableSQL += sql + ","
								values = append(values, vars...)
							}
						}
					}
				}
			}

			for _, uni := range stmt.Schema.ParseUniqueConstraints() {
				createTableSQL += "CONSTRAINT ? UNIQUE (?),"
				values = append(values, clause.Column{Name: uni.Name}, clause.Expr{SQL: stmt.Quote(uni.Field.DBName)})
			}

			for _, chk := range stmt.Schema.ParseCheckConstraints() {
				createTableSQL += "CONSTRAINT ? CHECK (?),"
				values = append(values, clause.Column{Name: chk.Name}, clause.Expr{SQL: chk.Constraint})
			}

			createTableSQL = strings.TrimSuffix(createTableSQL, ",")

			createTableSQL += ")"

			if tableOption, ok := m.DB.Get("gorm:table_options"); ok {
				createTableSQL += " " + fmt.Sprint(tableOption)
			}

			err = tx.Exec(createTableSQL, values...).Error
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

// DropTable drops the table starting from the bottom of the dependency chain.
// The function returns an error when Oracle databases report a missing table.
// If multiple errors occur, it returns a combined (joint) error.
func (m Migrator) DropTable(values ...interface{}) error {
	var errorList []error
	values = m.ReorderModels(values, false)
	for i := len(values) - 1; i >= 0; i-- {
		tx := m.DB.Session(&gorm.Session{})
		if err := m.RunWithValue(values[i], func(stmt *gorm.Statement) error {
			return tx.Exec(
				"DROP TABLE ? CASCADE CONSTRAINTS",
				clause.Table{Name: stmt.Table}).Error
		}); err != nil {
			errorList = append(errorList, err)
		}
	}

	if len(errorList) > 0 {
		return errors.Join(errorList...)
	}

	return nil
}

// HasTable returns table exists or not for value, value could be a struct or string
func (m Migrator) HasTable(value interface{}) bool {
	var count int64

	m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
		return m.DB.Raw("SELECT COUNT(*) FROM USER_TABLES WHERE TABLE_NAME = ?", stmt.Table).Row().Scan(&count)
	})

	return count > 0
}

// RenameTable renames table from oldName to newName
func (m Migrator) RenameTable(oldName, newName interface{}) error {
	var oldTable, newTable interface{}
	if v, ok := oldName.(string); ok {
		oldTable = clause.Table{Name: v}
	} else {
		stmt := &gorm.Statement{DB: m.DB}
		if err := stmt.Parse(oldName); err == nil {
			oldTable = m.CurrentTable(stmt)
		} else {
			return err
		}
	}

	if v, ok := newName.(string); ok {
		newTable = clause.Table{Name: v}
	} else {
		stmt := &gorm.Statement{DB: m.DB}
		if err := stmt.Parse(newName); err == nil {
			newTable = m.CurrentTable(stmt)
		} else {
			return err
		}
	}

	return m.DB.Exec("RENAME ? TO ?", oldTable, newTable).Error
}

// GetTables returns tables
func (m Migrator) GetTables() (tableList []string, err error) {
	err = m.DB.Raw("SELECT TABLE_NAME FROM USER_TABLES").Scan(&tableList).Error

	return
}

// AddColumn creates `name` column for the given `value`
func (m Migrator) AddColumn(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		// Check if the column name is already used
		if f := stmt.Schema.LookUpField(name); f != nil {
			return m.DB.Exec(
				"ALTER TABLE ? ADD (? ?)",
				clause.Table{Name: stmt.Schema.Table},
				clause.Column{Name: f.DBName},
				m.DB.Migrator().FullDataTypeOf(f),
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", name)
	})
}

// DropColumn drops value's `name` column
func (m Migrator) DropColumn(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if field := stmt.Schema.LookUpField(name); field != nil {
				name = field.DBName
			}
		}

		return m.DB.Exec(
			"ALTER TABLE ? DROP COLUMN ?",
			clause.Table{Name: stmt.Schema.Table},
			clause.Column{Name: name},
		).Error
	})
}

// AlterColumn alters value's `field` column's type based on schema definition
func (m Migrator) AlterColumn(value interface{}, field string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if f := stmt.Schema.LookUpField(field); f != nil {
				columnTypes, err := m.ColumnTypes(value)
				if err != nil {
					return err
				}

				var currentNullable bool
				var currentType string
				for _, col := range columnTypes {
					if strings.EqualFold(col.Name(), f.DBName) {
						currentNullable, _ = col.Nullable()
						currentType = strings.ToUpper(col.DatabaseTypeName())
						break
					}
				}

				desiredNullable := !f.NotNull
				desiredType := strings.ToUpper(m.DataTypeOf(f))

				// nullable → non-nullable → skip
				if currentNullable && !desiredNullable {
					return nil
				}

				// same type + same nullability → skip
				if currentNullable == desiredNullable && strings.Contains(currentType, desiredType) {
					return nil
				}

				sql := "ALTER TABLE ? MODIFY ? " + m.DataTypeOf(f)
				if f.NotNull {
					sql += " NOT NULL"
				} else if !currentNullable && desiredNullable {
					sql += " NULL"
				}

				return m.DB.Exec(
					sql,
					clause.Table{Name: stmt.Schema.Table},
					clause.Column{Name: f.DBName},
				).Error
			}
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

// HasColumn checks whether the table for the given value contains the specified column `field`
func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64

	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw("SELECT COUNT(*) FROM USER_TAB_COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?",
			stmt.Table,
			field,
		).Row().Scan(&count)
	})

	return count > 0
}

// ColumnTypes returns the column types for the given value’s table and any error encountered during execution
func (m Migrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	columnTypes := make([]gorm.ColumnType, 0)
	execErr := m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
		// We only need 1 row to get the metadata
		rows, err := m.DB.Session(&gorm.Session{}).Table(stmt.Table).Where("ROWNUM = 1").Rows()
		if err != nil {
			return err
		}

		defer func() {
			err = rows.Close()
		}()

		var rawColumnTypes []*sql.ColumnType
		rawColumnTypes, err = rows.ColumnTypes()
		if err != nil {
			return err
		}

		for _, c := range rawColumnTypes {
			columnTypes = append(columnTypes, migrator.ColumnType{SQLColumnType: c})
		}

		return
	})

	return columnTypes, execErr
}

// CreateConstraint creates constraint based on the given 'value' and 'name'
func (m Migrator) CreateConstraint(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
		if constraint != nil {
			if c, ok := constraint.(*schema.Constraint); ok {
				// Oracle doesn’t support OnUpdate on foreign keys.
				// Use a trigger instead to propagate the update to the child table instead.
				if len(c.References) > 0 && c.OnUpdate != "" {
					m.createUpadateCascadeTrigger(m.DB, c)
					c.OnUpdate = ""
					constraint = c
				}
			}

			vars := []interface{}{clause.Table{Name: table}}
			if stmt.TableExpr != nil {
				vars[0] = stmt.TableExpr
			}
			sql, values := constraint.Build()
			return m.DB.Exec("ALTER TABLE ? ADD "+sql, append(vars, values...)...).Error
		}
		return nil
	})
}

// HasConstraint checks whether the table for the given `value` contains the specified constraint `name`
func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64

	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		constraint, table := m.GuessConstraintInterfaceAndTable(stmt, name)
		if constraint != nil {
			name = constraint.GetName()
		}

		return m.DB.Raw(
			"SELECT COUNT(*) FROM USER_CONSTRAINTS WHERE TABLE_NAME = ? AND CONSTRAINT_NAME = ?",
			table, name,
		).Row().Scan(&count)
	})

	return count > 0
}

// DropConstraint drops constraint based on the given 'value' and 'name'
func (m Migrator) DropConstraint(value interface{}, name string) error {
	if err := m.RunWithValue(value, func(stmt *gorm.Statement) error {

		constraint, _ := m.GuessConstraintInterfaceAndTable(stmt, name)

		if c, ok := constraint.(*schema.Constraint); ok && c != nil {
			if len(c.References) > 0 && c.OnUpdate != "" {
				for i, fk := range c.ForeignKeys {
					triggerName := m.FkTriggerName(
						c.ReferenceSchema.Table,
						c.References[i].DBName,
						c.Schema.Table,
						fk.DBName,
					)
					return m.DB.Exec("DROP TRIGGER ?", clause.Column{Name: triggerName}).Error
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return m.Migrator.DropConstraint(value, name)
}

// DropIndex drops the index with the specified `name` from the table associated with `value`
func (m Migrator) DropIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				name = idx.Name
			}
		}

		return m.DB.Exec("DROP INDEX ?", clause.Column{Name: name}).Error
	})
}

// HasIndex checks whether the table for the given `value` contains an index with the specified `name`
func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil {
			if idx := stmt.Schema.LookIndex(name); idx != nil {
				name = idx.Name
			}
		}

		return m.DB.Raw(
			"SELECT COUNT(*) AS INDEX_COUNT FROM USER_INDEXES WHERE TABLE_NAME = ? AND INDEX_NAME = ?",
			stmt.Table,
			name,
		).Row().Scan(&count)
	})

	return count > 0
}

// RenameIndex renames index from oldName to newName on the table for the given `value`
func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Exec(
			"ALTER INDEX ? RENAME TO ?",
			clause.Column{Name: oldName}, clause.Column{Name: newName},
		).Error
	})
}

func (m Migrator) FullDataTypeOf(field *schema.Field) (expr clause.Expr) {
	expr.SQL = m.DataTypeOf(field)

	// Handle Oracle-specific default values FIRST
	if field.DefaultValue != "" {
		defaultSQL := m.buildOracleDefault(field.DefaultValue)
		if defaultSQL != "" {
			expr.SQL += " " + defaultSQL
		}
	}

	// Handle Go default values (different from tag defaults)
	if field.HasDefaultValue && field.DefaultValueInterface != nil {
		defaultSQL := m.buildOracleDefaultFromInterface(field.DefaultValueInterface)
		if defaultSQL != "" && !strings.Contains(expr.SQL, "DEFAULT") {
			expr.SQL += " " + defaultSQL
		}
	}

	// Add NOT NULL after defaults
	if field.NotNull {
		expr.SQL += " NOT NULL"
	}

	return expr
}

// Builds Oracle-compatible default values from string
func (m Migrator) buildOracleDefault(defaultValue string) string {
	defaultValue = strings.TrimSpace(defaultValue)

	if defaultValue == "" {
		return ""
	}

	// Handle Oracle keywords and functions (case-insensitive)
	switch strings.ToUpper(defaultValue) {
	case "NULL":
		return "DEFAULT NULL"
	case "CURRENT_TIMESTAMP", "NOW()":
		return "DEFAULT CURRENT_TIMESTAMP"
	case "SYSDATE":
		return "DEFAULT SYSDATE"
	case "TRUE":
		return "DEFAULT 1"
	case "FALSE":
		return "DEFAULT 0"
	}

	// Handle sequence calls (contains .NEXTVAL)
	if strings.Contains(strings.ToUpper(defaultValue), ".NEXTVAL") {
		return "DEFAULT " + defaultValue
	}

	// Handle numeric values
	if m.isNumeric(defaultValue) {
		return "DEFAULT " + defaultValue
	}

	// Handle simple date format like "2000-01-02"
	if len(defaultValue) == 10 && strings.Count(defaultValue, "-") == 2 {
		if _, err := time.Parse("2006-01-02", defaultValue); err == nil {
			return "DEFAULT TO_DATE('" + defaultValue + "', 'YYYY-MM-DD')"
		}
	}

	// Handle already quoted strings
	if strings.HasPrefix(defaultValue, "'") && strings.HasSuffix(defaultValue, "'") {
		return "DEFAULT " + defaultValue
	}

	// Handle string values that need quoting
	return "DEFAULT '" + defaultValue + "'"
}

// Build Oracle-compatible default values from Go interface
func (m Migrator) buildOracleDefaultFromInterface(value interface{}) string {
	switch v := value.(type) {
	case bool:
		if v {
			return "DEFAULT 1"
		}
		return "DEFAULT 0"
	case string:
		if v == "" {
			return "DEFAULT ''"
		}
		return "DEFAULT '" + strings.ReplaceAll(v, "'", "''") + "'"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "DEFAULT " + fmt.Sprintf("%v", v)
	case float32, float64:
		return "DEFAULT " + fmt.Sprintf("%v", v)
	case time.Time:
		return "DEFAULT TO_TIMESTAMP('" + v.Format("2006-01-02 15:04:05") + "', 'YYYY-MM-DD HH24:MI:SS')"
	case nil:
		return "DEFAULT NULL"
	default:
		// For other types, convert to string and quote
		str := fmt.Sprintf("%v", v)
		return "DEFAULT '" + strings.ReplaceAll(str, "'", "''") + "'"
	}
}

// Helper function for numeric detection
func (m Migrator) isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func (m Migrator) FkTriggerName(refTable string, refField string, table string, field string) string {
	return fmt.Sprintf("fk_trigger_%s_%s_%s_%s", refTable, refField, table, field)
}

// Creates a trigger to cascade the update to the child table
func (m Migrator) createUpadateCascadeTrigger(tx *gorm.DB, constraint *schema.Constraint) error {
	onUpdate := strings.TrimSpace(strings.ToLower(constraint.OnUpdate))
	if onUpdate != "cascade" && onUpdate != "set null" && onUpdate != "set default" {
		return nil
	}

	parentTable := constraint.ReferenceSchema.Table
	quotedParentTable := QuoteIdentifier(parentTable)
	table := constraint.Schema.Table
	quotedTable := QuoteIdentifier(table)

	for i, fk := range constraint.ForeignKeys {
		parentField := constraint.References[i].DBName
		quotedParentField := QuoteIdentifier(parentField)
		field := fk.DBName
		quotedField := QuoteIdentifier(field)
		triggerName := m.FkTriggerName(parentTable, parentField, table, field)
		quotedTriggerName := QuoteIdentifier(triggerName)

		var updateValue string
		switch onUpdate {
		case "cascade":
			updateValue = ":NEW." + quotedParentField
		case "set null":
			updateValue = "NULL"
		case "set default":
			updateValue = "DEFAULT"
		}

		plsql := fmt.Sprintf(
			`CREATE OR REPLACE TRIGGER %s
AFTER UPDATE OF %s ON %s
FOR EACH ROW
BEGIN
  UPDATE %s
  SET %s = %s
  WHERE %s = :OLD.%s;
END;`,
			quotedTriggerName,
			quotedParentField,
			quotedParentTable,
			quotedTable,
			quotedField,
			updateValue,
			quotedField,
			quotedParentField,
		)

		if err := tx.Exec(plsql).Error; err != nil {
			return err
		}
	}

	return nil
}
