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
	"strings"
)

type BooleanTest struct {
	ID       uint         `gorm:"column:ID;primaryKey"`
	Flag     bool         `gorm:"column:FLAG"`
	Nullable *bool        `gorm:"column:NULLABLE"`
	SQLBool  sql.NullBool `gorm:"column:SQL_BOOL"`
}

func (BooleanTest) TableName() string {
	return "BOOLEAN_TESTS"
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

	if len(trues) == 0 {
		t.Fatalf("expected at least 1 row, got 0")
	}

	for _, row := range trues {
		if !row.Flag {
			t.Errorf("expected only true rows, got false")
		}
	}
}

func TestBooleanNegativeInvalidDBValue(t *testing.T) {
    DB.Migrator().DropTable(&BooleanTest{})
    DB.AutoMigrate(&BooleanTest{})

    if err := DB.Exec(`INSERT INTO "BOOLEAN_TESTS" ("ID","FLAG") VALUES (2001, 2)`).Error; err != nil {
        t.Fatalf("failed to insert invalid bool: %v", err)
    }

    var got BooleanTest
    err := DB.First(&got, 2001).Error
    if err == nil {
        t.Fatal("expected invalid boolean scan error, got nil")
    }

    if !strings.Contains(err.Error(), "invalid") &&
       !strings.Contains(err.Error(), "convert") {
        t.Fatalf("expected boolean conversion error, got: %v", err)
    }
}

func TestBooleanInsertWithIntValues(t *testing.T) {
	DB.Migrator().DropTable(&BooleanTest{})
	DB.AutoMigrate(&BooleanTest{})

	if err := DB.Exec("INSERT INTO BOOLEAN_TESTS (ID, FLAG) VALUES (1001, 1)").Error; err != nil {
		t.Fatalf("failed to insert int 1 as boolean: %v", err)
	}
	if err := DB.Exec("INSERT INTO BOOLEAN_TESTS (ID, FLAG) VALUES (1002, 0)").Error; err != nil {
		t.Fatalf("failed to insert int 0 as boolean: %v", err)
	}

	var gotTrue, gotFalse BooleanTest
	if err := DB.First(&gotTrue, 1001).Error; err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if gotTrue.Flag != true {
		t.Errorf("expected true for 1, got %v", gotTrue.Flag)
	}

	if err := DB.First(&gotFalse, 1002).Error; err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if gotFalse.Flag != false {
		t.Errorf("expected false for 0, got %v", gotFalse.Flag)
	}
}

func TestBooleanSQLNullBool(t *testing.T) {
	DB.Migrator().DropTable(&BooleanTest{})
	DB.AutoMigrate(&BooleanTest{})

	bt := BooleanTest{SQLBool: sql.NullBool{Bool: true, Valid: true}}
	DB.Create(&bt)

	var got BooleanTest
	DB.First(&got, bt.ID)
	if !got.SQLBool.Valid || got.SQLBool.Bool != true {
		t.Errorf("expected sql.NullBool true/valid, got %+v", got.SQLBool)
	}
}

func TestBooleanDefaultValue(t *testing.T) {
	DB.Migrator().DropTable(&BooleanTest{})
	DB.AutoMigrate(&BooleanTest{})

	bt := BooleanTest{}
	if err := DB.Create(&bt).Error; err != nil {
		t.Fatalf("insert default failed: %v", err)
	}

	var got BooleanTest
	DB.First(&got, bt.ID)

	if got.Flag != false {
		t.Errorf("expected default false, got %v", got.Flag)
	}
}

func TestBooleanQueryMixedComparisons(t *testing.T) {
    DB.Migrator().DropTable(&BooleanTest{})
    DB.AutoMigrate(&BooleanTest{})

    DB.Create(&BooleanTest{Flag: true})
    DB.Create(&BooleanTest{Flag: false})

    var gotNum []BooleanTest

    // FILTER USING NUMBER
    if err := DB.Where("FLAG = 1").Find(&gotNum).Error; err != nil {
        t.Fatal(err)
    }
    if len(gotNum) == 0 {
        t.Errorf("expected at least 1 row for FLAG=1")
    }

    var gotStr []BooleanTest
    if err := DB.Where("FLAG = 'true'").Find(&gotStr).Error; err == nil {
        t.Errorf("expected ORA-01722 when comparing NUMBER to string literal")
    }
}

func TestBooleanStringCoercion(t *testing.T) {
	DB.Migrator().DropTable(&BooleanTest{})
	DB.AutoMigrate(&BooleanTest{})

	// Insert using string literals
	if err := DB.Exec("INSERT INTO BOOLEAN_TESTS (ID, FLAG) VALUES (2001, '1')").Error; err != nil {
		t.Fatalf("failed to insert '1': %v", err)
	}
	if err := DB.Exec("INSERT INTO BOOLEAN_TESTS (ID, FLAG) VALUES (2002, '0')").Error; err != nil {
		t.Fatalf("failed to insert '0': %v", err)
	}

	var got1, got2 BooleanTest
	DB.First(&got1, 2001)
	DB.First(&got2, 2002)

	if got1.Flag != true {
		t.Errorf("expected true for '1', got %v", got1.Flag)
	}
	if got2.Flag != false {
		t.Errorf("expected false for '0', got %v", got2.Flag)
	}
}


func TestBooleanNullableColumn(t *testing.T) {
	t.Skip("Skipping until nullable bool bug is resolved")
	DB.Migrator().DropTable(&BooleanTest{})
	DB.AutoMigrate(&BooleanTest{})

	// Insert a row with NULL value for Nullable column
	bt := BooleanTest{Flag: true, Nullable: nil}
	if err := DB.Create(&bt).Error; err != nil {
		t.Fatalf("failed to insert NULL bool: %v", err)
	}

	var got BooleanTest
	if err := DB.First(&got, bt.ID).Error; err != nil {
		t.Fatal(err)
	}

	if got.Nullable != nil {
		t.Errorf("expected NULL, got %v", *got.Nullable)
	}
}
