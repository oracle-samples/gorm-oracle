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

	"github.com/oracle/gorm-oracle/oracle"
	"gorm.io/gorm"
)

func TestOpen(t *testing.T) {
	dsn := "gorm:gorm@tcp(localhost:9910)/gorm?loc=Asia%2FHongKong" // invalid loc
	_, err := gorm.Open(oracle.Open(dsn), &gorm.Config{})
	if err == nil {
		t.Fatalf("should returns error but got nil")
	}
}

func TestReturningWithNullToZeroValues(t *testing.T) {
	t.Skip()
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
	results := DB.First(&got, "id = ?", u1.ID)
	if results.Error != nil {
		t.Fatalf("errors happened on first: %v", results.Error)
	} else if results.RowsAffected != 1 {
		t.Fatalf("rows affected expects: %v, got %v", 1, results.RowsAffected)
	} else if got.ID != u1.ID {
		t.Fatalf("first expects: %v, got %v", u1, got)
	}

	results = DB.Select("id, name").Find(&got)
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
	results = DB.Where("id in (?, ?)", u1.ID, u2.ID).Order("id asc").Select("id, name").Find(&gotUsers)
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
