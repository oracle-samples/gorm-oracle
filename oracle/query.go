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
	"reflect"
	"regexp"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

// Identifies the table name alias provided as
// "\"users\" \"u\"" and "\"users\" u". Gorm already handles
// the other formats like "users u", "users AS u" etc.
var tableRegexp = regexp.MustCompile(`^"(\w+)"\s+"?(\w+)"?$`)

func BeforeQuery(db *gorm.DB) {
	if db == nil || db.Statement == nil || db.Statement.TableExpr == nil {
		return
	}
	name := db.Statement.TableExpr.SQL
	if strings.Contains(name, " ") || strings.Contains(name, "`") {
		if results := tableRegexp.FindStringSubmatch(name); len(results) == 3 {
			db.Statement.Table = results[2]
		}
	}
}

func AfterQuery(db *gorm.DB) {
	if db == nil || db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	destinationStruct := reflect.ValueOf(db.Statement.Dest)
	for _, field := range db.Statement.Schema.Fields {
		if field.DataType == "uuid" {
			uuidDestField := reflect.Indirect(destinationStruct).FieldByName(field.Name)
			if uuidDestField.Kind() == reflect.Ptr && !uuidDestField.IsNil() && uuidDestField.Elem().IsZero() {
				// NULL UUIDs should be returned as nil if the field is a pointer type (as opposed to all-zero value)
				field.Set(db.Statement.Context, destinationStruct, nil)
			}
		}
	}
}

// MismatchedCaseHandler handles Oracle case insensitivity for unquoted identifiers.
// When identifiers are not quoted, Oracle returns selected columns in uppercase.
// This callback populates Statement.ColumnMapping for both base model fields and
// GORM-generated join aliases so scan can resolve them back to the expected field
// and nested relation names.
func MismatchedCaseHandler(gormDB *gorm.DB) {
	if gormDB.Statement == nil || gormDB.Statement.Schema == nil {
		return
	}
	if len(gormDB.Statement.Schema.Fields) > 0 && gormDB.Statement.ColumnMapping == nil {
		gormDB.Statement.ColumnMapping = map[string]string{}
	}

	for _, field := range gormDB.Statement.Schema.Fields {
		gormDB.Statement.ColumnMapping[strings.ToUpper(field.DBName)] = field.Name
	}

	addJoinColumnMappings(gormDB.Statement)
}

func addJoinColumnMappings(stmt *gorm.Statement) {
	if stmt == nil || stmt.Schema == nil || len(stmt.Joins) == 0 {
		return
	}

	for _, join := range stmt.Joins {
		relations, ok := resolveJoinRelations(stmt.Schema, join.Name)
		if !ok {
			continue
		}

		parentTableName := clause.CurrentTable
		for idx, rel := range relations {
			curAliasName := rel.Name
			if parentTableName != clause.CurrentTable {
				curAliasName = utils.NestedRelationName(parentTableName, curAliasName)
			}

			aliasName := curAliasName
			if idx == len(relations)-1 && join.Alias != "" {
				aliasName = join.Alias
			}

			addNestedFieldMappings(stmt.ColumnMapping, aliasName, rel.FieldSchema)
			parentTableName = curAliasName
		}
	}
}

func resolveJoinRelations(root *schema.Schema, joinName string) ([]*schema.Relationship, bool) {
	if rel, ok := root.Relationships.Relations[joinName]; ok {
		return []*schema.Relationship{rel}, true
	}

	names := strings.Split(joinName, ".")
	if len(names) <= 1 {
		return nil, false
	}

	relations := make([]*schema.Relationship, 0, len(names))
	currentRelations := root.Relationships.Relations
	for _, name := range names {
		rel, ok := currentRelations[name]
		if !ok {
			return nil, false
		}
		relations = append(relations, rel)
		currentRelations = rel.FieldSchema.Relationships.Relations
	}

	return relations, true
}

func addNestedFieldMappings(columnMapping map[string]string, aliasName string, joinSchema *schema.Schema) {
	if len(columnMapping) == 0 || aliasName == "" || joinSchema == nil {
		return
	}

	for _, dbName := range joinSchema.DBNames {
		nestedName := utils.NestedRelationName(aliasName, dbName)
		columnMapping[strings.ToUpper(nestedName)] = nestedName
	}
}
