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
	"testing"
)

// Struct mapping to phone_typ object
type Phone struct {
	CountryCode string `gorm:"column:country_code"`
	AreaCode    string `gorm:"column:area_code"`
	PhNumber    string `gorm:"column:ph_number"`
}

// Struct for table with VARRAY of OBJECT
type DeptPhoneList struct {
	DeptNo    int     `gorm:"column:dept_no;primaryKey"`
	PhoneList []Phone `gorm:"column:phone_list;type:\"phone_varray_typ\""`
}

// Struct for table with email VARRAY
type EmailVarrayTable struct {
	ID     uint     `gorm:"column:ID;primaryKey"`
	Emails []string `gorm:"column:EMAILS;type:\"email_list_arr\""`
}

func TestStringVarray(t *testing.T) {
	t.Skip("Skipping due to issue #87")
	dropTable := `
				BEGIN
				EXECUTE IMMEDIATE 'DROP TABLE "email_varray_tables"';
				EXCEPTION WHEN OTHERS THEN NULL; END;`
	if err := DB.Exec(dropTable).Error; err != nil {
		t.Fatalf("Failed to drop email_varray_tables: %v", err)
	}
	dropType := `
				BEGIN
				EXECUTE IMMEDIATE 'DROP TYPE "email_list_arr"';
				EXCEPTION WHEN OTHERS THEN NULL; END;`
	if err := DB.Exec(dropType).Error; err != nil {
		t.Fatalf("Failed to drop email_list_arr: %v", err)
	}
	createType := `CREATE OR REPLACE TYPE "email_list_arr" AS VARRAY(10) OF VARCHAR2(80)`
	if err := DB.Exec(createType).Error; err != nil {
		t.Fatalf("Failed to create email_list_arr: %v", err)
	}
	createTable := `CREATE TABLE "email_varray_tables" (
					"ID" NUMBER PRIMARY KEY,
					"EMAILS" "email_list_arr"
				)`
	if err := DB.Exec(createTable).Error; err != nil {
		t.Fatalf("Failed to create email_varray_tables: %v", err)
	}

	// Insert initial data via raw SQL
	insertRaw := `INSERT INTO "email_varray_tables" VALUES (1, "email_list_arr"('alice@example.com','bob@example.com','gorm@oracle.com'))`
	if err := DB.Exec(insertRaw).Error; err != nil {
		t.Fatalf("Failed to insert row via raw SQL: %v", err)
	}

	// Query
	var got EmailVarrayTable
	if err := DB.First(&got, 1).Error; err != nil {
		t.Fatalf("Failed to fetch varray: %v", err)
	}
	expected := []string{"alice@example.com", "bob@example.com", "gorm@oracle.com"}
	if !reflect.DeepEqual(expected, got.Emails) {
		t.Errorf("String VARRAY roundtrip failed: got %v, want %v", got.Emails, expected)
	}

	// Update
	newEmails := []string{"u1@ex.com", "u2@ex.com"}
	if err := DB.Model(&got).Update("Emails", newEmails).Error; err != nil {
		t.Fatalf("Failed to update emails: %v", err)
	}
	var updated EmailVarrayTable
	if err := DB.First(&updated, 1).Error; err != nil {
		t.Fatalf("Failed to reload updated EmailVarrayTable: %v", err)
	}
	if !reflect.DeepEqual(updated.Emails, newEmails) {
		t.Errorf("String VARRAY update failed: got %v, want %v", updated.Emails, newEmails)
	}

	// Insert
	item := EmailVarrayTable{
		ID:     1,
		Emails: []string{"alice_new@example.com", "bob_new@example.com", "gorm_new@oracle.com"},
	}
	if err := DB.Create(&item).Error; err != nil {
		t.Fatalf("Failed to insert varray: %v", err)
	}
}

func TestVarrayOfObject(t *testing.T) {
	t.Skip("Skipping due to issue #87")
	dropTable := `
				BEGIN
				EXECUTE IMMEDIATE 'DROP TABLE "dept_phone_lists"';
				EXCEPTION WHEN OTHERS THEN NULL; END;`
	if err := DB.Exec(dropTable).Error; err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}
	dropVarray := `
				BEGIN
				EXECUTE IMMEDIATE 'DROP TYPE "phone_varray_typ"';
				EXCEPTION WHEN OTHERS THEN NULL; END;`
	DB.Exec(dropVarray)
	dropObj := `
				BEGIN
				EXECUTE IMMEDIATE 'DROP TYPE "phone_typ"';
				EXCEPTION WHEN OTHERS THEN NULL; END;`
	DB.Exec(dropObj)

	createObj := `CREATE OR REPLACE TYPE "phone_typ" AS OBJECT (
  				  "country_code"   VARCHAR2(2), 
				  "area_code"      VARCHAR2(3),
				  "ph_number"      VARCHAR2(7))`
	if err := DB.Exec(createObj).Error; err != nil {
		t.Fatalf("Failed to create phone_typ: %v", err)
	}
	createVarray := `CREATE OR REPLACE TYPE "phone_varray_typ" AS VARRAY(5) OF "phone_typ"`
	if err := DB.Exec(createVarray).Error; err != nil {
		t.Fatalf("Failed to create phone_varray_typ: %v", err)
	}
	createTable := `CREATE TABLE "dept_phone_lists"("dept_no" NUMBER(5) PRIMARY KEY, "phone_list" "phone_varray_typ")`
	if err := DB.Exec(createTable).Error; err != nil {
		t.Fatalf("Failed to create dept_phone_lists: %v", err)
	}

	// Insert initial data using raw SQL
	insertRaw := `INSERT INTO "dept_phone_lists" VALUES (
				  100,
				  "phone_varray_typ"( "phone_typ" ('01', '650', '5550123'),
									  "phone_typ" ('01', '650', '5550148'),
				  				      "phone_typ" ('01', '650', '5550192')))
				  `
	if err := DB.Exec(insertRaw).Error; err != nil {
		t.Fatalf("Failed to insert example row: %v", err)
	}

	// Query
	var got DeptPhoneList
	if err := DB.First(&got, 100).Error; err != nil {
		t.Fatalf("failed to query: %v", err)
	}
	want := []Phone{
		{CountryCode: "01", AreaCode: "650", PhNumber: "5550123"},
		{CountryCode: "01", AreaCode: "650", PhNumber: "5550148"},
		{CountryCode: "01", AreaCode: "650", PhNumber: "5550192"},
	}
	if !reflect.DeepEqual(got.PhoneList, want) {
		t.Errorf("Select phone_varray_typ roundtrip failed: got %v, want %v", got.PhoneList, want)
	}

	// Insert
	item := DeptPhoneList{
		DeptNo: 200,
		PhoneList: []Phone{
			{CountryCode: "86", AreaCode: "10", PhNumber: "1234567"},
			{CountryCode: "49", AreaCode: "89", PhNumber: "7654321"},
		},
	}
	err := DB.Create(&item).Error
	if err != nil {
		t.Errorf("Failed to insert: %v", err)
	} else {
		var read DeptPhoneList
		if err := DB.First(&read, 200).Error; err != nil {
			t.Errorf("Could not fetch inserted row: %v", err)
		} else if !reflect.DeepEqual(read.PhoneList, item.PhoneList) {
			t.Errorf("Inserted VARRAY<OBJECT> mismatch: got %v want %v", read.PhoneList, item.PhoneList)
		}
	}

	// Update
	newPhones := []Phone{
		{CountryCode: "07", AreaCode: "312", PhNumber: "2000044"},
	}
	if err := DB.Model(&DeptPhoneList{}).Where("\"dept_no\" = ?", 200).
		Update("PhoneList", newPhones).Error; err != nil {
		t.Errorf("Failed to update varray<obj>: %v", err)
	}
	var updated DeptPhoneList
	if err := DB.First(&updated, 200).Error; err != nil {
		t.Errorf("Could not fetch updated row: %v", err)
	} else if !reflect.DeepEqual(updated.PhoneList, newPhones) {
		t.Errorf("Updated VARRAY<OBJECT> mismatch: got %v want %v", updated.PhoneList, newPhones)
	}
}
