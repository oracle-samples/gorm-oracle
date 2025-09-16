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
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// Helper function to get Oracle array type for a field
func getOracleArrayType(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "TABLE OF NUMBER(1)"
	case schema.Int, schema.Uint:
		return "TABLE OF NUMBER"
	case schema.Float:
		return "TABLE OF NUMBER"
	case schema.String:
		if field.Size > 0 && field.Size <= 4000 {
			return fmt.Sprintf("TABLE OF VARCHAR2(%d)", field.Size)
		}
		return "TABLE OF VARCHAR2(4000)"
	case schema.Time:
		return "TABLE OF TIMESTAMP WITH TIME ZONE"
	case schema.Bytes:
		return "TABLE OF BLOB"
	default:
		return "TABLE OF VARCHAR2(4000)" // Safe default
	}
}

// Helper function to get all column names for a table
func getAllTableColumns(schema *schema.Schema) []string {
	var columns []string
	for _, field := range schema.Fields {
		if field.DBName != "" {
			columns = append(columns, field.DBName)
		}
	}
	return columns
}

// Helper to check if a variable is an OUT parameter
func isOutParam(v interface{}) bool {
	_, ok := v.(sql.Out)
	return ok
}

// Find field by database column name
func findFieldByDBName(schema *schema.Schema, dbName string) *schema.Field {
	for _, field := range schema.Fields {
		if field.DBName == dbName {
			return field
		}
	}
	return nil
}

// Create typed destination for OUT parameters
func createTypedDestination(f *schema.Field) interface{} {
	if f == nil {
		var s string
		return &s
	}

	ft := f.FieldType
	for ft.Kind() == reflect.Ptr {
		ft = ft.Elem()
	}

	if ft == reflect.TypeOf(gorm.DeletedAt{}) {
		return new(sql.NullTime)
	}
	if ft == reflect.TypeOf(time.Time{}) {
		if !f.NotNull { // nullable column => keep NULLs
			return new(sql.NullTime)
		}
		return new(time.Time)
	}

	switch ft {
	case reflect.TypeOf(sql.NullTime{}):
		return new(sql.NullTime)
	case reflect.TypeOf(sql.NullInt64{}):
		return new(sql.NullInt64)
	case reflect.TypeOf(sql.NullInt32{}):
		return new(sql.NullInt32)
	case reflect.TypeOf(sql.NullFloat64{}):
		return new(sql.NullFloat64)
	case reflect.TypeOf(sql.NullBool{}):
		return new(sql.NullBool)
	}

	switch ft.Kind() {
	case reflect.String:
		return new(string)

	case reflect.Bool:
		return new(int64)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return new(int64)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return new(uint64)

	case reflect.Float32, reflect.Float64:
		return new(float64)
	}

	// Fallback
	var s string
	return &s
}

// Convert values for Oracle-specific types
func convertValue(val interface{}) interface{} {
	if val == nil {
		return nil
	}

	// Dereference pointers
	v := reflect.ValueOf(val)
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
		val = v.Interface()
	}

	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}

	switch v := val.(type) {
	case bool:
		if v {
			return 1
		} else {
			return 0
		}
	case string:
		return v
	default:
		return val
	}
}

// Convert Oracle values back to Go types
func convertFromOracleToField(value interface{}, field *schema.Field) interface{} {
	if value == nil || field == nil {
		return nil
	}

	targetType := field.FieldType
	isPtr := targetType.Kind() == reflect.Ptr
	if isPtr {
		targetType = targetType.Elem()
	}
	if isJSONField(field) {
		switch v := value.(type) {
		case string:
			return datatypes.JSON([]byte(v))
		case []byte:
			return datatypes.JSON(v)
		}
	}
	var converted interface{}

	switch targetType {
	case reflect.TypeOf(gorm.DeletedAt{}):
		if nullTime, ok := value.(sql.NullTime); ok {
			converted = gorm.DeletedAt{Time: nullTime.Time, Valid: nullTime.Valid}
		} else {
			converted = gorm.DeletedAt{}
		}
	case reflect.TypeOf(time.Time{}):
		switch vv := value.(type) {
		case time.Time:
			converted = vv
		case sql.NullTime:
			if vv.Valid {
				converted = vv.Time
			} else {
				// DB returned NULL
				if isPtr {
					return nil // -> *time.Time(nil)
				}
				// non-pointer time.Time: represent NULL as zero time
				return time.Time{}
			}
		default:
			converted = value
		}

	case reflect.TypeOf(sql.NullTime{}):
		if nullTime, ok := value.(sql.NullTime); ok {
			converted = nullTime
		} else {
			converted = sql.NullTime{}
		}

	case reflect.TypeOf(sql.NullInt64{}):
		if nullInt, ok := value.(sql.NullInt64); ok {
			converted = nullInt
		} else {
			converted = sql.NullInt64{}
		}
	case reflect.TypeOf(sql.NullInt32{}):
		if nullInt, ok := value.(sql.NullInt32); ok {
			converted = nullInt
		} else {
			converted = sql.NullInt32{}
		}
	case reflect.TypeOf(sql.NullFloat64{}):
		if nullFloat, ok := value.(sql.NullFloat64); ok {
			converted = nullFloat
		} else {
			converted = sql.NullFloat64{}
		}
	case reflect.TypeOf(sql.NullBool{}):
		if nullBool, ok := value.(sql.NullBool); ok {
			converted = nullBool
		} else {
			converted = sql.NullBool{}
		}
	default:
		// primitives and everything else
		converted = convertPrimitiveType(value, targetType)
	}

	// Pointer targets: nil for "zero-ish", else allocate and set.
	if isPtr {
		if isZeroFor(targetType, converted) {
			return nil
		}
		ptr := reflect.New(targetType)
		ptr.Elem().Set(reflect.ValueOf(converted))
		return ptr.Interface()
	}

	return converted
}

func isJSONField(f *schema.Field) bool {
	// Schema says it's JSON
	if strings.EqualFold(string(f.DataType), "json") {
		return true
	}
	// Some drivers/taggers carry type hints
	if strings.Contains(strings.ToUpper(f.Tag.Get("TYPE")), "JSON") {
		return true
	}
	// Detect gorm.io/datatypes.JSON by reflected type
	if f.FieldType.Name() == "JSON" && f.FieldType.PkgPath() == "gorm.io/datatypes" {
		return true
	}
	return false
}

// Helper function to handle primitive type conversions
func convertPrimitiveType(value interface{}, targetType reflect.Type) interface{} {
	switch targetType.Kind() {
	case reflect.Bool:
		if v, ok := value.(int64); ok {
			return v != 0
		}
		return value
	case reflect.Int:
		if v, ok := value.(int64); ok {
			return int(v)
		}
		if v, ok := value.(uint64); ok {
			return int(v)
		}
		return value
	case reflect.Int8:
		if v, ok := value.(int64); ok {
			return int8(v)
		}
		if v, ok := value.(uint64); ok {
			return int8(v)
		}
		return value
	case reflect.Int16:
		if v, ok := value.(int64); ok {
			return int16(v)
		}
		if v, ok := value.(uint64); ok {
			return int16(v)
		}
		return value
	case reflect.Int32:
		if v, ok := value.(int64); ok {
			return int32(v)
		}
		if v, ok := value.(uint64); ok {
			return int32(v)
		}
		return value
	case reflect.Int64:
		if v, ok := value.(uint64); ok {
			return int64(v)
		}
		return value
	case reflect.Uint:
		if v, ok := value.(uint64); ok {
			return uint(v)
		}
		if v, ok := value.(int64); ok {
			return uint(v)
		}
		return value
	case reflect.Uint8:
		if v, ok := value.(uint64); ok {
			return uint8(v)
		}
		if v, ok := value.(int64); ok {
			return uint8(v)
		}
		return value
	case reflect.Uint16:
		if v, ok := value.(uint64); ok {
			return uint16(v)
		}
		if v, ok := value.(int64); ok {
			return uint16(v)
		}
		return value
	case reflect.Uint32:
		if v, ok := value.(uint64); ok {
			return uint32(v)
		}
		if v, ok := value.(int64); ok {
			return uint32(v)
		}
		return value
	case reflect.Uint64:
		if v, ok := value.(int64); ok {
			return uint64(v)
		}
		if v, ok := value.(uint64); ok {
			return v
		}
		return value
	case reflect.Float32:
		if v, ok := value.(float64); ok {
			return float32(v)
		}
		return value
	case reflect.Float64:
		return value
	case reflect.String:
		return value
	default:
		return value
	}
}

// Add quotes to the identifiers
func writeQuotedIdentifier(builder *strings.Builder, identifier string) {
	var (
		underQuoted, selfQuoted bool
		continuousBacktick      int8
		shiftDelimiter          int8
	)

	for _, v := range []byte(identifier) {
		switch v {
		case '"':
			continuousBacktick++
			if continuousBacktick == 2 {
				builder.WriteString(`""`)
				continuousBacktick = 0
			}
		case '.':
			if continuousBacktick > 0 || !selfQuoted {
				shiftDelimiter = 0
				underQuoted = false
				continuousBacktick = 0
				builder.WriteByte('"')
			}
			builder.WriteByte(v)
			continue
		default:
			if shiftDelimiter-continuousBacktick <= 0 && !underQuoted {
				builder.WriteByte('"')
				underQuoted = true
				if selfQuoted = continuousBacktick > 0; selfQuoted {
					continuousBacktick -= 1
				}
			}

			for ; continuousBacktick > 0; continuousBacktick -= 1 {
				builder.WriteString(`""`)
			}

			builder.WriteByte(v)
		}
		shiftDelimiter++
	}

	if continuousBacktick > 0 && !selfQuoted {
		builder.WriteString(`""`)
	}
	builder.WriteByte('"')
}

func QuoteIdentifier(identifier string) string {
	var builder strings.Builder
	writeQuotedIdentifier(&builder, identifier)
	return builder.String()
}

// writeTableRecordCollectionDecl writes the PL/SQL declarations needed to
// define a custom record type and a collection of that record type,
// based on the schema of the given table.
//
// Specifically, it generates:
//   - A RECORD type (`t_record`) with fields corresponding to the table's columns.
//   - A nested TABLE type (`t_records`) of `t_record`.
//
// The declarations are written into the provided strings.Builder in the
// correct PL/SQL syntax, so they can be used as part of a larger PL/SQL block.
//
// Example output:
//
//	TYPE t_record IS RECORD (
//	  "id" "users"."id"%TYPE,
//	  "created_at" "users"."created_at"%TYPE,
//	  ...
//	);
//	TYPE t_records IS TABLE OF t_record;
//
// Parameters:
//   - plsqlBuilder: The builder to write the PL/SQL code into.
//   - dbNames: The slice containing the column names.
//   - table: The table name
func writeTableRecordCollectionDecl(db *gorm.DB, plsqlBuilder *strings.Builder, dbNames []string, table string) {
	// Declare a record where each element has the same structure as a row from the given table
	plsqlBuilder.WriteString("  TYPE t_record IS RECORD (\n")
	for i, field := range dbNames {
		if i > 0 {
			plsqlBuilder.WriteString(",\n")
		}
		plsqlBuilder.WriteString("    ")
		db.QuoteTo(plsqlBuilder, field)
		plsqlBuilder.WriteString(" ")
		db.QuoteTo(plsqlBuilder, table)
		plsqlBuilder.WriteString(".")
		db.QuoteTo(plsqlBuilder, field)
		plsqlBuilder.WriteString("%TYPE")
	}
	plsqlBuilder.WriteString("\n")
	plsqlBuilder.WriteString("  );\n")
	plsqlBuilder.WriteString("  TYPE t_records IS TABLE OF t_record;\n")
}

// Helper function to check if a value represents NULL
func isNullValue(value interface{}) bool {
	if value == nil {
		return true
	}

	// Check for different NULL types
	switch v := value.(type) {
	case sql.NullString:
		return !v.Valid
	case sql.NullInt64:
		return !v.Valid
	case sql.NullInt32:
		return !v.Valid
	case sql.NullFloat64:
		return !v.Valid
	case sql.NullBool:
		return !v.Valid
	case sql.NullTime:
		return !v.Valid
	default:
		return false
	}
}

func isZeroFor(t reflect.Type, v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return true
	}
	// exact type match?
	if rv.Type() == t {
		// special-case time.Time
		if t == reflect.TypeOf(time.Time{}) {
			return rv.Interface().(time.Time).IsZero()
		}
		// generic zero check
		z := reflect.Zero(t)
		return reflect.DeepEqual(rv.Interface(), z.Interface())
	}
	// If types differ (e.g., sql.NullTime), treat invalid as zero
	if nt, ok := v.(sql.NullTime); ok {
		return !nt.Valid
	}
	return false
}
