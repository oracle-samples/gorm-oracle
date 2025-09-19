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
	"testing"

	"github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
)

func TestOpen(t *testing.T) {
	dsns := []string{
		"gorm@localhost:9910/invalid",
		"gorm:gorm@localhost:9910",
		"invalid_dsn_string",
		"gorm:gorm@tcp(localhost:9910)/gorm?loc=Asia%2FHongKong",
		"gorm@localhost:2300/invalid_service",
		"gorm@localhost:invalid_port/cdb1_pdb1",
		"gorm@invalid_host:1111/cdb1_pdb1",
	}
	for _, dsn := range dsns {
		_, err := gorm.Open(oracle.Open(dsn), &gorm.Config{})
		if err == nil {
			t.Errorf("should return error for dsn=%q but got nil", dsn)
		}
	}
}

func TestReturningWithNullToZeroValues(t *testing.T) {
	// This user struct will leverage the existing users table, but override
	// the Name field to default to null.
	type user struct {
		gorm.Model
		Name string `gorm:"default:null"`
	}
	u1 := user{}

	if results := DB.Create(&u1); results.Error != nil {
		t.Fatalf("errors happened on create: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if u1.ID == 0 {
		t.Fatalf("ID expects : not equal 0, got %v", u1.ID)
	}

	got := user{}
	results := DB.First(&got, "\"id\" = ?", u1.ID)
	if results.Error != nil {
		t.Fatalf("errors happened on first: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if got.ID != u1.ID {
		t.Fatalf("first expects: %v, got %v", u1, got)
	}

	results = DB.Select("\"id\", \"name\"").Find(&got)
	if results.Error != nil {
		t.Fatalf("errors happened on first: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if got.ID != u1.ID {
		t.Fatalf("select expects: %v, got %v", u1, got)
	}

	u1.Name = "jinzhu"
	if results := DB.Save(&u1); results.Error != nil {
		t.Fatalf("errors happened on update: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	}

	u1 = user{} // important to reinitialize this before creating it again
	u2 := user{}
	db := DB.Session(&gorm.Session{CreateBatchSize: 10})

	if results := db.Create([]*user{&u1, &u2}); results.Error != nil {
		t.Fatalf("errors happened on create: %v", results.Error)
	} else if results.RowsAffected != 2 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if u1.ID == 0 {
		t.Fatalf("ID expects : not equal 0, got %v", u1.ID)
	} else if u2.ID == 0 {
		t.Fatalf("ID expects : not equal 0, got %v", u2.ID)
	}

	var gotUsers []user
	results = DB.Where("\"id\" in (?, ?)", u1.ID, u2.ID).Order("\"id\" asc").Select("\"id\", \"name\"").Find(&gotUsers)
	if results.Error != nil {
		t.Fatalf("errors happened on first: %v", results.Error)
	} else if results.RowsAffected != 2 {
		t.Fatalf("rows affected expects: %v, got %v", 2, results.RowsAffected)
	} else if gotUsers[0].ID != u1.ID {
		t.Fatalf("select expects: %v, got %v", u1.ID, gotUsers[0].ID)
	} else if gotUsers[1].ID != u2.ID {
		t.Fatalf("select expects: %v, got %v", u2.ID, gotUsers[1].ID)
	}

	u1.Name = "Jinzhu"
	u2.Name = "Zhang"
	if results := DB.Save([]*user{&u1, &u2}); results.Error != nil {
		t.Fatalf("errors happened on update: %v", results.Error)
	} else if results.RowsAffected != 2 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	}
}

func TestReturningWithNullAndAdditionalFields(t *testing.T) {
	type userWithFields struct {
		gorm.Model
		Name    string `gorm:"default:noname"`
		Age     int
		Comment string
	}

	DB.Migrator().DropTable(&userWithFields{})
	DB.Migrator().AutoMigrate(&userWithFields{})

	// Insert a user and verify defaults
	u := userWithFields{}
	if results := DB.Create(&u); results.Error != nil {
		t.Fatalf("errors happened on create: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if u.ID == 0 || u.Name != "noname" || u.Age != 0 || u.Comment != "" {
		t.Fatalf("create expects ID!=0, Name='noname', Age=0, Comment='', got %+v", u)
	}

	// Update all fields and verify
	u.Name = "TestName"
	u.Age = 42
	u.Comment = "Hello"
	if results := DB.Save(&u); results.Error != nil {
		t.Fatalf("errors happened on update: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if u.Name != "TestName" || u.Age != 42 || u.Comment != "Hello" {
		t.Fatalf("update expects Name='TestName', Age=42, Comment='Hello', got %+v", u)
	}

	// Fetch and verify
	got := userWithFields{}
	results := DB.First(&got, "\"id\" = ?", u.ID)
	if results.Error != nil {
		t.Fatalf("errors happened on first: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if got.ID != u.ID || got.Name != "TestName" || got.Age != 42 || got.Comment != "Hello" {
		t.Fatalf("first expects %+v, got %+v", u, got)
	}

	// Batch create and check
	u1 := userWithFields{}
	u2 := userWithFields{Name: "foo"}
	u3 := userWithFields{Name: "bar", Age: 99, Comment: "bar-comment"}
	db := DB.Session(&gorm.Session{CreateBatchSize: 10})
	if results := db.Create([]*userWithFields{&u1, &u2, &u3}); results.Error != nil {
		t.Fatalf("errors happened on create: %v", results.Error)
	} else if results.RowsAffected != 3 {
		t.Fatalf("rows affected expects: %v, got %v", 3, results.RowsAffected)
	} else if u1.ID == 0 || u2.ID == 0 || u3.ID == 0 {
		t.Fatalf("ID expects : not equal 0, got %v,%v,%v", u1.ID, u2.ID, u3.ID)
	} else if u1.Name != "noname" || u2.Name != "foo" || u3.Name != "bar" {
		t.Fatalf("names expect: noname, foo, bar, got: %q,%q,%q", u1.Name, u2.Name, u3.Name)
	} else if u1.Age != 0 || u2.Age != 0 || u3.Age != 99 {
		t.Fatalf("ages expect: 0,0,99, got: %v,%v,%v", u1.Age, u2.Age, u3.Age)
	} else if u1.Comment != "" || u2.Comment != "" || u3.Comment != "bar-comment" {
		t.Fatalf("comments expect: '', '', 'bar-comment', got: %q,%q,%q", u1.Comment, u2.Comment, u3.Comment)
	}

	// Batch update and check
	u1.Name = "A"
	u2.Name = "B"
	u3.Comment = "updated"
	if results := DB.Save([]*userWithFields{&u1, &u2, &u3}); results.Error != nil {
		t.Fatalf("errors happened on update: %v", results.Error)
	} else if results.RowsAffected != 3 {
		t.Fatalf("rows affected expects: %v, got %v", 3, results.RowsAffected)
	} else if u1.Name != "A" || u2.Name != "B" || u3.Name != "bar" {
		t.Fatalf("names expect: A, B, bar, got: %q,%q,%q", u1.Name, u2.Name, u3.Name)
	} else if u1.Age != 0 || u2.Age != 0 || u3.Age != 99 {
		t.Fatalf("ages expect: 0,0,99, got: %v,%v,%v", u1.Age, u2.Age, u3.Age)
	} else if u1.Comment != "" || u2.Comment != "" || u3.Comment != "updated" {
		t.Fatalf("comments expect: '', '', 'updated', got: %q,%q,%q", u1.Comment, u2.Comment, u3.Comment)
	}

	// Batch fetch and verify
	updated := []userWithFields{}
	results = DB.Where("\"id\" in (?, ?, ?)", u1.ID, u2.ID, u3.ID).Order("\"id\" asc").Find(&updated)
	if results.Error != nil {
		t.Fatalf("errors happened on batch find: %v", results.Error)
	} else if results.RowsAffected != 3 {
		t.Fatalf("rows affected expects: %v, got %v", 3, results.RowsAffected)
	} else if len(updated) != 3 {
		t.Fatalf("batch find expects: %v records, got %v", 3, len(updated))
	} else if updated[0].ID != u1.ID || updated[1].ID != u2.ID || updated[2].ID != u3.ID {
		t.Fatalf("batch find expects IDs: %v,%v,%v, got: %v,%v,%v", u1.ID, u2.ID, u3.ID, updated[0].ID, updated[1].ID, updated[2].ID)
	} else if updated[0].Name != "A" || updated[1].Name != "B" || updated[2].Name != "bar" {
		t.Fatalf("batch find expects Names: A,B,bar, got: %q,%q,%q", updated[0].Name, updated[1].Name, updated[2].Name)
	} else if updated[0].Age != 0 || updated[1].Age != 0 || updated[2].Age != 99 {
		t.Fatalf("batch find expects Ages: 0,0,99, got: %v,%v,%v", updated[0].Age, updated[1].Age, updated[2].Age)
	} else if updated[0].Comment != "" || updated[1].Comment != "" || updated[2].Comment != "updated" {
		t.Fatalf("batch find expects Comments: '', '', 'updated', got: %q,%q,%q", updated[0].Comment, updated[1].Comment, updated[2].Comment)
	}

	// Delete and verify one user
	if err := DB.Delete(&userWithFields{}, u2.ID).Error; err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	var deleted userWithFields
	if err := DB.First(&deleted, u2.ID).Error; err == nil {
		t.Fatalf("Deleted user with ID=%d found", u2.ID)
	}

	DB.Migrator().DropTable(&userWithFields{})
}
