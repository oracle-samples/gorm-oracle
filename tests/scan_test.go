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
	"reflect"
	"sort"
	"strings"
	"testing"

	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

type PersonAddressInfo struct {
	Person  *Person  `gorm:"embedded"`
	Address *Address `gorm:"embedded"`
}

func TestScan(t *testing.T) {
	user1 := User{Name: "ScanUser1", Age: 1}
	user2 := User{Name: "ScanUser2", Age: 10}
	user3 := User{Name: "ScanUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	type result struct {
		ID   uint
		Name string
		Age  int
	}

	var res result
	DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" = ?", user3.ID).Scan(&res)
	if res.ID != user3.ID || res.Name != user3.Name || res.Age != int(user3.Age) {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", res, user3)
	}

	var resPointer *result
	if err := DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" = ?", user3.ID).Scan(&resPointer).Error; err != nil {
		t.Fatalf("Failed to query with pointer of value, got error %v", err)
	} else if resPointer.ID != user3.ID || resPointer.Name != user3.Name || resPointer.Age != int(user3.Age) {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", res, user3)
	}

	DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" = ?", user2.ID).Scan(&res)
	if res.ID != user2.ID || res.Name != user2.Name || res.Age != int(user2.Age) {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", res, user2)
	}

	DB.Model(&User{Model: gorm.Model{ID: user3.ID}}).Select("\"id\", \"name\", \"age\"").Scan(&res)
	if res.ID != user3.ID || res.Name != user3.Name || res.Age != int(user3.Age) {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", res, user3)
	}

	doubleAgeRes := &result{}
	if err := DB.Table("users").Select("\"age\" + \"age\" as \"age\"").Where("\"id\" = ?", user3.ID).Scan(&doubleAgeRes).Error; err != nil {
		t.Errorf("Scan to pointer of pointer")
	}

	if doubleAgeRes.Age != int(res.Age)*2 {
		t.Errorf("Scan double age as age, expect: %v, got %v", res.Age*2, doubleAgeRes.Age)
	}

	var results []result
	DB.Table("users").Select("\"name\", \"age\"").Where("\"id\" in ?", []uint{user2.ID, user3.ID}).Scan(&results)

	sort.Slice(results, func(i, j int) bool {
		return strings.Compare(results[i].Name, results[j].Name) <= -1
	})

	if len(results) != 2 || results[0].Name != user2.Name || results[1].Name != user3.Name {
		t.Errorf("Scan into struct map, got %#v", results)
	}

	type ID uint64
	var id ID
	DB.Raw("select \"id\" from \"users\" where \"id\" = ?", user2.ID).Scan(&id)
	if uint(id) != user2.ID {
		t.Errorf("Failed to scan to customized data type")
	}

	var resInt interface{}
	resInt = &User{}
	if err := DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" = ?", user3.ID).Find(&resInt).Error; err != nil {
		t.Fatalf("Failed to query with pointer of value, got error %v", err)
	} else if resInt.(*User).ID != user3.ID || resInt.(*User).Name != user3.Name || resInt.(*User).Age != user3.Age {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", resInt, user3)
	}

	var resInt2 interface{}
	resInt2 = &User{}
	if err := DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" = ?", user3.ID).Scan(&resInt2).Error; err != nil {
		t.Fatalf("Failed to query with pointer of value, got error %v", err)
	} else if resInt2.(*User).ID != user3.ID || resInt2.(*User).Name != user3.Name || resInt2.(*User).Age != user3.Age {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", resInt2, user3)
	}

	var resInt3 interface{}
	resInt3 = []User{}
	if err := DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" = ?", user3.ID).Find(&resInt3).Error; err != nil {
		t.Fatalf("Failed to query with pointer of value, got error %v", err)
	} else if rus := resInt3.([]User); len(rus) == 0 || rus[0].ID != user3.ID || rus[0].Name != user3.Name || rus[0].Age != user3.Age {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", resInt3, user3)
	}

	var resInt4 interface{}
	resInt4 = []User{}
	if err := DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" = ?", user3.ID).Scan(&resInt4).Error; err != nil {
		t.Fatalf("Failed to query with pointer of value, got error %v", err)
	} else if rus := resInt4.([]User); len(rus) == 0 || rus[0].ID != user3.ID || rus[0].Name != user3.Name || rus[0].Age != user3.Age {
		t.Fatalf("Scan into struct should work, got %#v, should %#v", resInt4, user3)
	}

	var resInt5 interface{}
	resInt5 = []User{}
	if err := DB.Table("users").Select("\"id\", \"name\", \"age\"").Where("\"id\" IN ?", []uint{user1.ID, user2.ID, user3.ID}).Find(&resInt5).Error; err != nil {
		t.Fatalf("Failed to query with pointer of value, got error %v", err)
	} else if rus := resInt5.([]User); len(rus) != 3 {
		t.Fatalf("Scan into struct should work, got %+v, len %v", resInt5, len(rus))
	}
}

func TestScanRows(t *testing.T) {
	user1 := User{Name: "ScanRowsUser1", Age: 1}
	user2 := User{Name: "ScanRowsUser2", Age: 10}
	user3 := User{Name: "ScanRowsUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	rows, err := DB.Table("users").Where("\"name\" = ? or \"name\" = ?", user2.Name, user3.Name).Select("\"name\", \"age\"").Rows()
	if err != nil {
		t.Errorf("No error should happen, got %v", err)
	}

	type Result struct {
		Name string
		Age  int
	}

	var results []Result
	for rows.Next() {
		var result Result
		if err := DB.ScanRows(rows, &result); err != nil {
			t.Errorf("should get no error, but got %v", err)
		}
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return strings.Compare(results[i].Name, results[j].Name) <= -1
	})

	if !reflect.DeepEqual(results, []Result{{Name: "ScanRowsUser2", Age: 10}, {Name: "ScanRowsUser3", Age: 20}}) {
		t.Errorf("Should find expected results, got %+v", results)
	}

	var ages int
	if err := DB.Table("users").Where("\"name\" = ? or \"name\" = ?", user2.Name, user3.Name).Select("SUM(\"age\")").Scan(&ages).Error; err != nil || ages != 30 {
		t.Fatalf("failed to scan ages, got error %v, ages: %v", err, ages)
	}

	var name string
	if err := DB.Table("users").Where("\"name\" = ?", user2.Name).Select("\"name\"").Scan(&name).Error; err != nil || name != user2.Name {
		t.Fatalf("failed to scan name, got error %v, name: %v", err, name)
	}
}

func TestScanRowsNullValuesScanToFieldDefault(t *testing.T) {
	DB.Save(&User{})

	rows, err := DB.Table("users").
		Select(`
			NULL AS bool_field,
			NULL AS int_field,
			NULL AS int8_field,
			NULL AS int16_field,
			NULL AS int32_field,
			NULL AS int64_field,
			NULL AS uint_field,
			NULL AS uint8_field,
			NULL AS uint16_field,
			NULL AS uint32_field,
			NULL AS uint64_field,
			NULL AS float32_field,
			NULL AS float64_field,
			NULL AS string_field,
			NULL AS time_field,
			NULL AS time_ptr_field,
			NULL AS embedded_int_field,
			NULL AS nested_embedded_int_field,
			NULL AS embedded_ptr_int_field
        `).Rows()
	if err != nil {
		t.Errorf("No error should happen, got %v", err)
	}

	type NestedEmbeddedStruct struct {
		NestedEmbeddedIntField            int
		NestedEmbeddedIntFieldWithDefault int `gorm:"default:2"`
	}

	type EmbeddedStruct struct {
		EmbeddedIntField     int
		NestedEmbeddedStruct `gorm:"embedded"`
	}

	type EmbeddedPtrStruct struct {
		EmbeddedPtrIntField   int
		*NestedEmbeddedStruct `gorm:"embedded"`
	}

	type Result struct {
		BoolField          bool
		IntField           int
		Int8Field          int8
		Int16Field         int16
		Int32Field         int32
		Int64Field         int64
		UIntField          uint
		UInt8Field         uint8
		UInt16Field        uint16
		UInt32Field        uint32
		UInt64Field        uint64
		Float32Field       float32
		Float64Field       float64
		StringField        string
		TimeField          time.Time
		TimePtrField       *time.Time
		EmbeddedStruct     `gorm:"embedded"`
		*EmbeddedPtrStruct `gorm:"embedded"`
	}

	currTime := time.Now()
	reusedVar := Result{
		BoolField:         true,
		IntField:          1,
		Int8Field:         1,
		Int16Field:        1,
		Int32Field:        1,
		Int64Field:        1,
		UIntField:         1,
		UInt8Field:        1,
		UInt16Field:       1,
		UInt32Field:       1,
		UInt64Field:       1,
		Float32Field:      1.1,
		Float64Field:      1.1,
		StringField:       "hello",
		TimeField:         currTime,
		TimePtrField:      &currTime,
		EmbeddedStruct:    EmbeddedStruct{EmbeddedIntField: 1, NestedEmbeddedStruct: NestedEmbeddedStruct{NestedEmbeddedIntField: 1, NestedEmbeddedIntFieldWithDefault: 2}},
		EmbeddedPtrStruct: &EmbeddedPtrStruct{EmbeddedPtrIntField: 1, NestedEmbeddedStruct: &NestedEmbeddedStruct{NestedEmbeddedIntField: 1, NestedEmbeddedIntFieldWithDefault: 2}},
	}

	for rows.Next() {
		if err := DB.ScanRows(rows, &reusedVar); err != nil {
			t.Errorf("should get no error, but got %v", err)
		}
	}

	if !reflect.DeepEqual(reusedVar, Result{}) {
		t.Errorf("Should find zero values in struct fields, got %+v\n", reusedVar)
	}
}

func TestScanToEmbedded(t *testing.T) {
	t.Skip()
	person1 := Person{Name: "person 1"}
	person2 := Person{Name: "person 2"}
	DB.Save(&person1).Save(&person2)

	address1 := Address{Name: "address 1"}
	address2 := Address{Name: "address 2"}
	DB.Save(&address1).Save(&address2)

	DB.Create(&PersonAddress{PersonID: person1.ID, AddressID: int(address1.ID)})
	DB.Create(&PersonAddress{PersonID: person1.ID, AddressID: int(address2.ID)})
	DB.Create(&PersonAddress{PersonID: person2.ID, AddressID: int(address1.ID)})

	var personAddressInfoList []*PersonAddressInfo
	if err := DB.Select("\"people\".*, \"addresses\".*").
		Table("people").
		Joins("inner join \"person_addresses\" on \"people\".\"id\" = \"person_addresses\".\"person_id\"").
		Joins("inner join \"addresses\" on \"person_addresses\".\"address_id\" = \"addresses\".\"id\"").
		Find(&personAddressInfoList).Error; err != nil {
		t.Errorf("Failed to run join query, got error: %v", err)
	}

	personMatched := false
	addressMatched := false

	for _, info := range personAddressInfoList {
		if info.Person == nil {
			t.Fatalf("Failed, expected not nil, got person nil")
		}
		if info.Address == nil {
			t.Fatalf("Failed, expected not nil, got address nil")
		}
		if info.Person.ID == person1.ID {
			personMatched = true
			if info.Person.Name != person1.Name {
				t.Errorf("Failed, expected %v, got %v", person1.Name, info.Person.Name)
			}
		}
		if info.Address.ID == address1.ID {
			addressMatched = true
			if info.Address.Name != address1.Name {
				t.Errorf("Failed, expected %v, got %v", address1.Name, info.Address.Name)
			}
		}
	}

	if !personMatched {
		t.Errorf("Failed, no person matched")
	}
	if !addressMatched {
		t.Errorf("Failed, no address matched")
	}

	personDupField := Person{ID: person1.ID}
	if err := DB.Select("\"people\".\"id\", \"people\".*").
		First(&personDupField).Error; err != nil {
		t.Errorf("Failed to run join query, got error: %v", err)
	}
	tests.AssertEqual(t, person1, personDupField)

	user := User{
		Name: "TestScanToEmbedded_1",
		Manager: &User{
			Name:    "TestScanToEmbedded_1_m1",
			Manager: &User{Name: "TestScanToEmbedded_1_m1_m1"},
		},
	}
	DB.Create(&user)

	type UserScan struct {
		ID        uint
		Name      string
		ManagerID *uint
	}
	var user2 UserScan
	err := DB.Raw("SELECT * FROM \"users\" INNER JOIN \"users\" \"manager\" ON \"users\".\"manager_id\" = \"manager\".\"id\" WHERE \"users\".\"id\" = ?", user.ID).Scan(&user2).Error
	tests.AssertEqual(t, err, nil)
}
