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
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// Identifies the table name alias provided as
// "\"users\" \"u\"" and "\"users\" u" and "\"users\"". Gorm already handles
// the other formats like "users u", "users AS u" etc.
var tableRegexp = regexp.MustCompile(`^"(\w+)"(?:\s+"?(\w+)"?)?$`)

func BeforeQuery(db *gorm.DB) {
	if db == nil || db.Statement == nil || db.Statement.TableExpr == nil {
		return
	}
	name := db.Statement.TableExpr.SQL
	if strings.Contains(name, " ") || strings.Contains(name, "`") {
		if results := tableRegexp.FindStringSubmatch(name); len(results) == 3 {
			if results[2] != "" {
				db.Statement.Table = results[2]
			} else {
				db.Statement.Table = results[1]
			}
		}
	}
	return
}

// MismatchedCaseHandler handles Oracle Case Insensitivity.
// When identifiers are not quoted, columns are returned by Oracle in uppercase.
// Fields in the models may be lower case for compatibility with other databases.
// Match them up with the fields using the column mapping.
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
}
