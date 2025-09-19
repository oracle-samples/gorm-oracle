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
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"maps"

	_ "github.com/godror/godror"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

const DefaultDriverName string = "godror"

type Config struct {
	DriverName           string
	DataSourceName       string
	Conn                 *sql.DB
	DefaultStringSize    uint
	SkipQuoteIdentifiers bool
}

type Dialector struct {
	*Config
}

// Name returns the name of the database dialect
func (d Dialector) Name() string {
	return "oracle"
}

// Open creates a new godror Dialector with the given DSN
func Open(dsn string) gorm.Dialector {
	return &Dialector{Config: &Config{DataSourceName: dsn}}
}

// New creates a new Dialector with the given config
func New(config Config) gorm.Dialector {
	return &Dialector{Config: &config}
}

// Initializes the database connection
func (d Dialector) Initialize(db *gorm.DB) (err error) {
	if d.DriverName == "" {
		d.DriverName = DefaultDriverName
	}

	d.DefaultStringSize = 4000

	config := &callbacks.Config{
		CreateClauses: []string{"INSERT", "VALUES", "ON CONFLICT", "RETURNING"},
		UpdateClauses: []string{"UPDATE", "SET", "WHERE", "RETURNING"},
		DeleteClauses: []string{"DELETE", "FROM", "WHERE", "RETURNING"},
	}
	callbacks.RegisterDefaultCallbacks(db, config)

	callback := db.Callback()
	callback.Create().Replace("gorm:create", Create)
	callback.Delete().Replace("gorm:delete", Delete)
	callback.Update().Replace("gorm:update", Update)
	callback.Query().Before("gorm:query").Register("oracle:before_query", BeforeQuery)

	maps.Copy(db.ClauseBuilders, OracleClauseBuilders())

	if d.Conn == nil {
		db.ConnPool, err = sql.Open(d.DriverName, d.DataSourceName)
	} else {
		db.ConnPool = d.Conn
	}

	if err != nil {
		return err
	}

	return nil
}

// Migrator returns the migrator instance associated with the given gorm.DB
func (d Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{
		Migrator: migrator.Migrator{
			Config: migrator.Config{
				DB:                          db,
				Dialector:                   d,
				CreateIndexAfterCreateTable: true,
			},
		},
	}
}

// Determines the data type for a schema field
func (d Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return d.getBooleanType()
	case schema.Int, schema.Uint:
		return d.getIntegerType(field)
	case schema.Float:
		return d.getFloatType(field)
	case schema.String:
		return d.getStringType(field)
	case schema.Time:
		return d.getDataTimeType()
	case schema.Bytes:
		return d.getBLOBType()
	default:
		dataType := strings.ToUpper(string(field.DataType))
		if dataType == "" {
			panic("sql type cannot be empty")
		}
		return dataType
	}
}

func (d Dialector) getStringType(field *schema.Field) string {
	var sqlType string
	if field.Size > 0 && field.Size <= 4000 {
		sqlType = fmt.Sprintf("VARCHAR2(%d)", field.Size)
	} else if field.Size == 0 {
		sqlType = fmt.Sprintf("VARCHAR2(%d)", d.DefaultStringSize)
	} else {
		sqlType = "CLOB"
	}

	// Don't add NOT NULL here - let GORM handle it to avoid duplicates
	return sqlType
}

func (d Dialector) getBooleanType() string {
	// Oracle doesn't support BOOLEAN in CREATE TABLE, use NUMBER(1) instead
	return "NUMBER(1)"
}

func (d Dialector) getDataTimeType() string {
	sqlType := "TIMESTAMP WITH TIME ZONE"
	return sqlType
}

func (d Dialector) getBLOBType() string {
	sqlType := "BLOB"
	return sqlType
}

func (d Dialector) getIntegerType(field *schema.Field) string {
	var sqlType string
	if field.Size > 0 {
		// Get the n requried for NUMBER(n)
		// Maxinum value of the integer with size X <= Maximum value of NUMBER(n)
		// 2^X - 1 <= 10^n - 1
		// n = ceil(X × log10(2))
		n := int(math.Ceil(float64(field.Size) * math.Log10(2)))
		n = int(math.Min(math.Max(float64(n), 1), 38))
		sqlType = fmt.Sprintf("NUMBER(%d)", n)
	} else {
		sqlType = "NUMBER"
	}

	if field.AutoIncrement {
		sqlType += " GENERATED BY DEFAULT AS IDENTITY"
	}
	return sqlType
}

func (d Dialector) getFloatType(field *schema.Field) string {
	var sqlType string
	if field.Precision > 0 {
		if field.Scale > 0 {
			sqlType = fmt.Sprintf("NUMBER(%d, %d)", field.Precision, field.Scale)
		} else {
			sqlType = fmt.Sprintf("FLOAT(%d)", field.Precision)
		}
	} else {
		sqlType = "FLOAT"
	}

	if val, ok := field.TagSettings["AUTOINCREMENT"]; ok && utils.CheckTruth(val) {
		sqlType += " GENERATED BY DEFAULT AS IDENTITY"
	}
	return sqlType
}

func (d Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	// This method is required by the gorm.Dialector interface but isn't used during migration
	// The actual default value handling is done in the migrator's FullDataTypeOf method
	return clause.Expr{SQL: ""}
}

// Handles variable binding in SQL statements
func (d Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v interface{}) {
	writer.WriteByte(':')
	writer.WriteString(strconv.Itoa(len(stmt.Vars)))
}

// Manages quoting of identifiers
func (d Dialector) QuoteTo(writer clause.Writer, str string) {
	out := str
	if !d.SkipQuoteIdentifiers {
		var builder strings.Builder
		writeQuotedIdentifier(&builder, str)
		out = builder.String()
	}
	_, _ = writer.WriteString(out)
}

var numericPlaceholder = regexp.MustCompile(`:(\d+)`)

// Explain Formats SQL statements with variables, string literals will be encoded
// with in ”
func (d Dialector) Explain(sqlStr string, vars ...interface{}) string {
	clonedVars := make([]interface{}, len(vars))
	copy(clonedVars, vars)

	// Unwrap sql.Out vars to actual values in the cloned slice
	for i, val := range clonedVars {
		if out, ok := val.(sql.Out); ok {
			valPtr := reflect.ValueOf(out.Dest)
			if valPtr.Kind() == reflect.Ptr && !valPtr.IsNil() {
				clonedVars[i] = valPtr.Elem().Interface()
			}
		}
	}
	return logger.ExplainSQL(sqlStr, numericPlaceholder, "'", clonedVars...)
}

// SavePoint creates a save point with the given name
func (d Dialector) SavePoint(tx *gorm.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return tx.Error
}

// RollbackTo Rolls back to the given save point
func (d Dialector) RollbackTo(tx *gorm.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return tx.Error
}
