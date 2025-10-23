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
	"database/sql"
	"testing"
)

type BooleanTest struct {
	ID       uint        `gorm:"column:ID;primaryKey"`
	Flag     bool        `gorm:"column:FLAG"` 
	Nullable *bool       `gorm:"column:NULLABLE"` 
	SQLBool  sql.NullBool `gorm:"column:SQL_BOOL"`
}

func TestBooleanBasicInsert(t *testing.T) {
	DB.Migrator().DropTable(&BooleanTest{})
	if err := DB.AutoMigrate(&BooleanTest{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	valTrue := true
	valFalse := false

	// Insert true
	bt1 := BooleanTest{Flag: true, Nullable: &valTrue}
	if err := DB.Create(&bt1).Error; err != nil {
		t.Fatalf("insert true failed: %v", err)
	}

	// Insert false
	bt2 := BooleanTest{Flag: false, Nullable: &valFalse}
	if err := DB.Create(&bt2).Error; err != nil {
		t.Fatalf("insert false failed: %v", err)
	}

	// Verify fetch
	var got1, got2 BooleanTest
	if err := DB.First(&got1, bt1.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got1.Flag != true {
		t.Errorf("expected true, got %v", got1.Flag)
	}

	if err := DB.First(&got2, bt2.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got2.Flag != false {
		t.Errorf("expected false, got %v", got2.Flag)
	}
}

func TestBooleanUpdate(t *testing.T) {
	DB.Migrator().DropTable(&BooleanTest{})
	DB.AutoMigrate(&BooleanTest{})

	bt := BooleanTest{Flag: false}
	DB.Create(&bt)

	// Update false → true
	if err := DB.Model(&bt).Update("Flag", true).Error; err != nil {
		t.Fatalf("update failed: %v", err)
	}

	var got BooleanTest
	DB.First(&got, bt.ID)
	if got.Flag != true {
		t.Errorf("expected true after update, got %v", got.Flag)
	}
}

func TestBooleanQueryFilters(t *testing.T) {
	DB.Migrator().DropTable(&BooleanTest{})
	DB.AutoMigrate(&BooleanTest{})

	DB.Create(&BooleanTest{Flag: true})
	DB.Create(&BooleanTest{Flag: false})

	var trues []BooleanTest
	if err := DB.Where("FLAG = ?", true).Find(&trues).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range trues {
		if !row.Flag {
			t.Errorf("expected only true rows, got false")
		}
	}
}

func TestBooleanNegativeInvalidDBValue(t *testing.T) {
	// Insert invalid value directly (bypassing GORM)
	if err := DB.Exec("INSERT INTO BOOLEAN_TEST (ID, FLAG) VALUES (999, 2)").Error; err != nil {
		t.Logf("expected insert error: %v", err)
		return
	}

	var got BooleanTest
	err := DB.First(&got, 999).Error
	if err == nil {
		t.Errorf("expected scan error for invalid boolean mapping, got %+v", got)
	}
}
