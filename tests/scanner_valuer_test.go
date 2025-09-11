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

package tests

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"testing"
	"time"
	"math"

	"github.com/godror/godror"
	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
)

func TestScannerValuer(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Fatalf("no error should happen when migrate scanner, valuer struct, got error %v", err)
	}

	data := ScannerValuerStruct{
		Name:     sql.NullString{String: "name", Valid: true},
		Gender:   &sql.NullString{String: "M", Valid: true},
		Age:      sql.NullInt64{Int64: 18, Valid: true},
		Male:     sql.NullBool{Bool: true, Valid: true},
		Height:   sql.NullFloat64{Float64: 1.8888, Valid: true},
		Birthday: sql.NullTime{Time: time.Now(), Valid: true},
		Allergen: NullString{sql.NullString{String: "Allergen", Valid: true}},
		Password: EncryptedData("pass1"),
		Bytes:    []byte("byte"),
		Num:      18,
		Strings:  StringsSlice{"a", "b", "c"},
		Structs: StructsSlice{
			{"name1", "value1"},
			{"name2", "value2"},
		},
		Role:             Role{Name: "admin"},
		ExampleStruct:    ExampleStruct{"name", "value1"},
		ExampleStructPtr: &ExampleStruct{"name", "value2"},
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("No error should happened when create scanner valuer struct, but got %v", err)
	}

	var result ScannerValuerStruct

	if err := DB.Find(&result, "\"id\" = ?", data.ID).Error; err != nil {
		t.Fatalf("no error should happen when query scanner, valuer struct, but got %v", err)
	}

	if result.ExampleStructPtr.Val != "value2" {
		t.Errorf(`ExampleStructPtr.Val should equal to "value2", but got %v`, result.ExampleStructPtr.Val)
	}

	if result.ExampleStruct.Val != "value1" {
		t.Errorf(`ExampleStruct.Val should equal to "value1", but got %#v`, result.ExampleStruct)
	}
	tests.AssertObjEqual(t, data, result, "Name", "Gender", "Age", "Male", "Height", "Birthday", "Password", "Bytes", "Num", "Strings", "Structs")
}

func TestScannerValuerWithFirstOrCreate(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Errorf("no error should happen when migrate scanner, valuer struct")
	}

	cond := ScannerValuerStruct{
		Name:   sql.NullString{String: "name", Valid: true},
		Gender: &sql.NullString{String: "M", Valid: true},
		Age:    sql.NullInt64{Int64: 18, Valid: true},
	}

	attrs := ScannerValuerStruct{
		ExampleStruct:    ExampleStruct{"name", "value1"},
		ExampleStructPtr: &ExampleStruct{"name", "value2"},
	}

	var result ScannerValuerStruct
	tx := DB.Where(cond).Attrs(attrs).FirstOrCreate(&result)

	if tx.RowsAffected != 1 {
		t.Errorf("RowsAffected should be 1 after create some record")
	}

	if tx.Error != nil {
		t.Errorf("Should not raise any error, but got %v", tx.Error)
	}

	tests.AssertObjEqual(t, result, cond, "Name", "Gender", "Age")

	if err := DB.Where(cond).Assign(ScannerValuerStruct{Age: sql.NullInt64{Int64: 18, Valid: true}}).FirstOrCreate(&result).Error; err != nil {
		t.Errorf("Should not raise any error, but got %v", err)
	}

	if result.Age.Int64 != 18 {
		t.Errorf("should update age to 18")
	}

	var result2 ScannerValuerStruct
	if err := DB.First(&result2, result.ID).Error; err != nil {
		t.Errorf("got error %v when query with %v", err, result.ID)
	}

	tests.AssertObjEqual(t, result2, result, "ID", "CreatedAt", "UpdatedAt", "Name", "Gender", "Age")
}

func TestInvalidValuer(t *testing.T) {
	DB.Migrator().DropTable(&ScannerValuerStruct{})
	if err := DB.Migrator().AutoMigrate(&ScannerValuerStruct{}); err != nil {
		t.Errorf("no error should happen when migrate scanner, valuer struct")
	}

	data := ScannerValuerStruct{
		Password:         EncryptedData("xpass1"),
		ExampleStruct:    ExampleStruct{"name", "value1"},
		ExampleStructPtr: &ExampleStruct{"name", "value2"},
	}

	if err := DB.Create(&data).Error; err == nil {
		t.Errorf("Should failed to create data with invalid data")
	}

	data.Password = EncryptedData("pass1")
	if err := DB.Create(&data).Error; err != nil {
		t.Errorf("Should got no error when creating data, but got %v", err)
	}

	if err := DB.Model(&data).Update("password", EncryptedData("xnewpass")).Error; err == nil {
		t.Errorf("Should failed to update data with invalid data")
	}

	if err := DB.Model(&data).Update("password", EncryptedData("newpass")).Error; err != nil {
		t.Errorf("Should got no error update data with valid data, but got %v", err)
	}

	tests.AssertEqual(t, data.Password, EncryptedData("newpass"))
}

type ScannerValuerStruct struct {
	gorm.Model
	Name             sql.NullString
	Gender           *sql.NullString
	Age              sql.NullInt64
	Male             sql.NullBool
	Height           sql.NullFloat64
	Birthday         sql.NullTime
	Allergen         NullString
	Password         EncryptedData
	Bytes            []byte
	Num              Num
	Strings          StringsSlice
	Structs          StructsSlice
	Role             Role
	UserID           *sql.NullInt64
	User             User
	EmptyTime        EmptyTime
	ExampleStruct    ExampleStruct
	ExampleStructPtr *ExampleStruct
}

func (StringsSlice) GormDataType() string { return "CLOB" }
func (l StringsSlice) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	v, err := l.Value()
	if err != nil {
		return gorm.Expr("?", err)
	}
	return gorm.Expr("?", v)
}

func (StructsSlice) GormDataType() string { return "CLOB" }
func (l StructsSlice) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	v, err := l.Value()
	if err != nil {
		return gorm.Expr("?", err)
	}
	return gorm.Expr("?", v)
}

type EncryptedData []byte

func (data *EncryptedData) Scan(value interface{}) error {
	if b, ok := value.([]byte); ok {
		if len(b) < 3 || b[0] != '*' || b[1] != '*' || b[2] != '*' {
			return errors.New("Too short")
		}

		*data = append((*data)[0:], b[3:]...)
		return nil
	} else if s, ok := value.(string); ok {
		*data = []byte(s[3:])
		return nil
	}

	return errors.New("Bytes expected")
}

func (data EncryptedData) Value() (driver.Value, error) {
	if len(data) > 0 && data[0] == 'x' {
		// needed to test failures
		return nil, errors.New("Should not start with 'x'")
	}

	// prepend asterisks
	return append([]byte("***"), data...), nil
}

type Num int64

func (n *Num) Scan(val interface{}) error {
	if val == nil {
		*n = 0
		return nil
	}

	switch x := val.(type) {
	case int64:
		*n = Num(x)
		return nil

	case godror.Number:
		i, err := strconv.ParseInt(string(x), 10, 64)
		if err != nil {
			return fmt.Errorf("Num.Scan: cannot parse godror.Number %q: %w", string(x), err)
		}
		*n = Num(i)
		return nil

	case string:
		i, err := strconv.ParseInt(x, 10, 64)
		if err != nil {
			return fmt.Errorf("Num.Scan: cannot parse string %q: %w", x, err)
		}
		*n = Num(i)
		return nil

	case []byte:
		i, err := strconv.ParseInt(string(x), 10, 64)
		if err != nil {
			return fmt.Errorf("Num.Scan: cannot parse []byte %q: %w", string(x), err)
		}
		*n = Num(i)
		return nil

	default:
		return fmt.Errorf("Num.Scan: unsupported type %T", val)
	}
}

type StringsSlice []string

func (l StringsSlice) Value() (driver.Value, error) {
	bytes, err := json.Marshal(l)
	return string(bytes), err
}

func (l *StringsSlice) Scan(input interface{}) error {
	switch value := input.(type) {
	case string:
		return json.Unmarshal([]byte(value), l)
	case []byte:
		return json.Unmarshal(value, l)
	default:
		return errors.New("not supported")
	}
}

type ExampleStruct struct {
	Name string
	Val  string
}

func (ExampleStruct) GormDataType() string {
	return "bytes"
}

func (s ExampleStruct) Value() (driver.Value, error) {
	if len(s.Name) == 0 {
		return nil, nil
	}
	// for test, has no practical meaning
	s.Name = ""
	return json.Marshal(s)
}

func (s *ExampleStruct) Scan(src interface{}) error {
	switch value := src.(type) {
	case string:
		return json.Unmarshal([]byte(value), s)
	case []byte:
		return json.Unmarshal(value, s)
	default:
		return errors.New("not supported")
	}
}

type StructsSlice []ExampleStruct

func (l StructsSlice) Value() (driver.Value, error) {
	bytes, err := json.Marshal(l)
	return string(bytes), err
}

func (l *StructsSlice) Scan(input interface{}) error {
	switch value := input.(type) {
	case string:
		return json.Unmarshal([]byte(value), l)
	case []byte:
		return json.Unmarshal(value, l)
	default:
		return errors.New("not supported")
	}
}

type Role struct {
	Name string `gorm:"size:256"`
}

func (role *Role) Scan(value interface{}) error {
	if b, ok := value.([]uint8); ok {
		role.Name = string(b)
	} else {
		role.Name = value.(string)
	}
	return nil
}

func (role Role) Value() (driver.Value, error) {
	return role.Name, nil
}

func (role Role) IsAdmin() bool {
	return role.Name == "admin"
}

type EmptyTime struct {
	time.Time
}

func (t *EmptyTime) Scan(v interface{}) error {
	nullTime := sql.NullTime{}
	err := nullTime.Scan(v)
	t.Time = nullTime.Time
	return err
}

func (t EmptyTime) Value() (driver.Value, error) {
	return time.Now() /* pass tests, mysql 8 doesn't support 0000-00-00 by default */, nil
}

type NullString struct {
	sql.NullString
}

type Point struct {
	X, Y int
}

func (point Point) GormDataType() string {
	return "geo"
}

func (point Point) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	return clause.Expr{
		SQL:  "ST_PointFromText(?)",
		Vars: []interface{}{fmt.Sprintf("POINT(%d %d)", point.X, point.Y)},
	}
}

func TestGORMValuer(t *testing.T) {
	type UserWithPoint struct {
		Name  string
		Point Point
	}

	dryRunDB := DB.Session(&gorm.Session{DryRun: true})

	stmt := dryRunDB.Create(&UserWithPoint{
		Name:  "jinzhu",
		Point: Point{X: 100, Y: 100},
	}).Statement

	if stmt.SQL.String() == "" || len(stmt.Vars) != 2 {
		t.Errorf("Failed to generate sql, got %v", stmt.SQL.String())
	}

	if !regexp.MustCompile(`INSERT INTO .user_with_points. \(.name.,.point.\) VALUES \(.+,ST_PointFromText\(.+\)\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("insert with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}

	stmt = dryRunDB.Model(UserWithPoint{}).Create(map[string]interface{}{
		"Name":  "jinzhu",
		"Point": clause.Expr{SQL: "ST_PointFromText(?)", Vars: []interface{}{"POINT(100 100)"}},
	}).Statement

	if !regexp.MustCompile(`INSERT INTO .user_with_points. \(.name.,.point.\) VALUES \(.+,ST_PointFromText\(.+\)\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("insert with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}

	stmt = dryRunDB.Table("user_with_points").Create(&map[string]interface{}{
		"Name":  "jinzhu",
		"Point": clause.Expr{SQL: "ST_PointFromText(?)", Vars: []interface{}{"POINT(100 100)"}},
	}).Statement

	if !regexp.MustCompile(`INSERT INTO .user_with_points. \(.Name.,.Point.\) VALUES \(.+,ST_PointFromText\(.+\)\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("insert with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}

	stmt = dryRunDB.Session(&gorm.Session{
		AllowGlobalUpdate: true,
	}).Model(&UserWithPoint{}).Updates(UserWithPoint{
		Name:  "jinzhu",
		Point: Point{X: 100, Y: 100},
	}).Statement

	if !regexp.MustCompile(`UPDATE .user_with_points. SET .name.=.+,.point.=ST_PointFromText\(.+\)`).MatchString(stmt.SQL.String()) {
		t.Errorf("update with sql.Expr, but got %v", stmt.SQL.String())
	}

	if !reflect.DeepEqual([]interface{}{"jinzhu", "POINT(100 100)"}, stmt.Vars) {
		t.Errorf("generated vars is not equal, got %v", stmt.Vars)
	}
}

func TestEncryptedDataScanValue(t *testing.T) {
	var ed EncryptedData

	if err := ed.Scan([]byte("***mypassword")); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if string(ed) != "mypassword" {
		t.Errorf("expected 'mypassword', got %s", string(ed))
	}

	if err := ed.Scan("***otherpass"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if string(ed) != "otherpass" {
		t.Errorf("expected 'otherpass', got %s", string(ed))
	}

	if err := ed.Scan([]byte("no")); err == nil {
		t.Errorf("expected error for too short input")
	}

	val, err := EncryptedData("mypassword").Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if string(val.([]byte)) != "***mypassword" {
		t.Errorf("expected '***mypassword', got %s", string(val.([]byte)))
	}

	_, err = EncryptedData("xpass").Value()
	if err == nil {
		t.Errorf("expected error when starting with 'x'")
	}
}

func TestNumScan(t *testing.T) {
	var n Num

	if err := n.Scan(int64(42)); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != 42 {
		t.Errorf("expected 42, got %d", n)
	}

	if err := n.Scan("99"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != 99 {
		t.Errorf("expected 99, got %d", n)
	}

	if err := n.Scan([]byte("123")); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != 123 {
		t.Errorf("expected 123, got %d", n)
	}

	if err := n.Scan(godror.Number("456")); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != 456 {
		t.Errorf("expected 456, got %d", n)
	}

	if err := n.Scan(nil); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 after nil scan, got %d", n)
	}

	if err := n.Scan(3.14); err == nil {
		t.Errorf("expected error for unsupported type")
	}

	if err := n.Scan(int64(0)); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}

	if err := n.Scan(int64(-123)); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if n != -123 {
		t.Errorf("expected -123, got %d", n)
	}

	large := int64(math.MaxInt64)
	if err := n.Scan(large); err != nil {
		t.Errorf("expected no error for large int, got %v", err)
	}
	if n != Num(large) {
		t.Errorf("expected %d, got %d", large, n)
	}

	small := int64(math.MinInt64)
	if err := n.Scan(small); err != nil {
		t.Errorf("expected no error for small int, got %v", err)
	}
	if n != Num(small) {
		t.Errorf("expected %d, got %d", small, n)
	}

	if err := n.Scan("   77  "); err == nil {
    	t.Errorf("expected error for spaced string")
	}

	if err := n.Scan(""); err == nil {
		t.Errorf("expected error for empty string")
	}

	if err := n.Scan([]byte("")); err == nil {
		t.Errorf("expected error for empty byte slice")
	}

	if err := n.Scan("not-a-number"); err == nil {
		t.Errorf("expected error for invalid string")
	}

	if err := n.Scan(godror.Number("abc")); err == nil {
		t.Errorf("expected error for invalid godror.Number")
	}

	if err := n.Scan(uint64(123)); err == nil {
		t.Errorf("expected error for unsupported uint64 type")
	}

	if err := n.Scan("9223372036854775808"); err == nil {
    	t.Errorf("expected error for overflow string")
	}

	if err := n.Scan(`"42"`); err == nil {
    	t.Errorf("expected error for quoted JSON string")
	}

	if err := n.Scan(true); err == nil {
    	t.Errorf("expected error for bool input")
	}
}

func TestStringsSliceScanValue(t *testing.T) {
	original := StringsSlice{"a", "b"}
	val, err := original.Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var parsed StringsSlice
	if err := parsed.Scan(val.(string)); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(parsed) != 2 || parsed[0] != "a" || parsed[1] != "b" {
		t.Errorf("unexpected parsed result: %#v", parsed)
	}

	if err := parsed.Scan(123); err == nil {
		t.Errorf("expected error for unsupported type")
	}
}

func TestStructsSliceScanValue(t *testing.T) {
	original := StructsSlice{
		{Name: "n1", Val: "v1"},
		{Name: "n2", Val: "v2"},
	}
	val, err := original.Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var parsed StructsSlice
	if err := parsed.Scan(val.(string)); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if len(parsed) != 2 || parsed[0].Name != "n1" || parsed[1].Val != "v2" {
		t.Errorf("unexpected parsed result: %#v", parsed)
	}
}

func TestExampleStructScanValue(t *testing.T) {
	orig := ExampleStruct{Name: "foo", Val: "bar"}
	val, err := orig.Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var parsed ExampleStruct
	if err := parsed.Scan(val.([]byte)); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if parsed.Val != "bar" {
		t.Errorf("expected Val 'bar', got %s", parsed.Val)
	}

	if err := parsed.Scan(123); err == nil {
		t.Errorf("expected error for unsupported type")
	}
}

func TestRoleScanValue(t *testing.T) {
	var r Role
	if err := r.Scan([]byte("admin")); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !r.IsAdmin() {
		t.Errorf("expected role to be admin")
	}

	if err := r.Scan("user"); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if r.IsAdmin() {
		t.Errorf("expected role to be not admin")
	}

	val, err := r.Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if val != "user" {
		t.Errorf("expected 'user', got %v", val)
	}
}

func TestEmptyTimeScanValue(t *testing.T) {
	var et EmptyTime
	now := time.Now()
	if err := et.Scan(now); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !et.Time.Equal(now) {
		t.Errorf("expected %v, got %v", now, et.Time)
	}

	val, err := et.Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if _, ok := val.(time.Time); !ok {
		t.Errorf("expected time.Time, got %T", val)
	}
}

func TestStringsSliceEmptySlice(t *testing.T) {
	empty := StringsSlice{}
	val, err := empty.Value()
	if err != nil {
		t.Errorf("expected no error for empty slice, got %v", err)
	}

	var parsed StringsSlice
	if err := parsed.Scan(val.(string)); err != nil {
		t.Errorf("expected no error scanning empty slice, got %v", err)
	}
	if len(parsed) != 0 {
		t.Errorf("expected empty slice, got %#v", parsed)
	}
}

func TestStringsSliceNilSlice(t *testing.T) {
	var nilSlice StringsSlice
	val, err := nilSlice.Value()
	if err != nil {
		t.Errorf("expected no error for nil slice, got %v", err)
	}
	if val.(string) != "null" && val.(string) != "[]" {
		t.Errorf("expected JSON null or [], got %v", val)
	}
}

func TestStringsSliceSpecialCharacters(t *testing.T) {
	special := StringsSlice{"a,b", "c\nd", "e\"f"}
	val, err := special.Value()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var parsed StringsSlice
	if err := parsed.Scan(val.(string)); err != nil {
		t.Errorf("expected no error scanning special chars, got %v", err)
	}
	if parsed[0] != "a,b" || parsed[1] != "c\nd" || parsed[2] != "e\"f" {
		t.Errorf("unexpected parsed result with special chars: %#v", parsed)
	}
}

func TestStringsSliceInvalidJSON(t *testing.T) {
	var parsed StringsSlice
	err := parsed.Scan("{bad json}")
	if err == nil {
		t.Errorf("expected error for malformed JSON, got nil")
	}
}

func TestStringsSliceWrongType(t *testing.T) {
	slice := StringsSlice{"x"}
	err := slice.Scan(123)
	if err == nil {
		t.Errorf("expected error for wrong type, got nil")
	}
}
