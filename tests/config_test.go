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
	"regexp"
	"testing"

	"github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
)

type Student struct {
	ID   uint
	Name string
}

func (s Student) TableName() string {
	return "STUDENTS"
}

func TestSkipQuoteIdentifiers(t *testing.T) {
	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	db.Migrator().DropTable(&Student{})
	db.Migrator().CreateTable(&Student{})

	if !db.Migrator().HasTable(&Student{}) {
		t.Errorf("Failed to get table: student")
	}

	if !db.Migrator().HasColumn(&Student{}, "ID") {
		t.Errorf("Failed to get column: id")
	}

	if !db.Migrator().HasColumn(&Student{}, "NAME") {
		t.Errorf("Failed to get column: name")
	}

	student := Student{ID: 1, Name: "John"}
	if err := db.Model(&Student{}).Create(&student).Error; err != nil {
		t.Errorf("Failed to insert student, got %v", err)
	}

	var result Student
	if err := db.First(&result).Error; err != nil {
		t.Errorf("Failed to query first student, got %v", err)
	}

	if result.ID != student.ID {
		t.Errorf("id should be %v, but got %v", student.ID, result.ID)
	}

	if result.Name != student.Name {
		t.Errorf("name should be %v, but got %v", student.Name, result.Name)
	}
}

func TestSkipQuoteIdentifiersSQL(t *testing.T) {
	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}
	dryrunDB := db.Session(&gorm.Session{DryRun: true})

	insertedStudent := Student{ID: 1, Name: "John"}
	result := dryrunDB.Model(&Student{}).Create(&insertedStudent)

	if !regexp.MustCompile(`^INSERT INTO STUDENTS \(name,id\) VALUES \(:1,:2\)$`).MatchString(result.Statement.SQL.String()) {
		t.Errorf("invalid insert SQL, got %v", result.Statement.SQL.String())
	}

	// Test First
	var firstStudent Student
	result = dryrunDB.First(&firstStudent)

	if !regexp.MustCompile(`^SELECT \* FROM STUDENTS ORDER BY STUDENTS\.id FETCH NEXT 1 ROW ONLY$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	// Test Find
	var foundStudent Student
	result = dryrunDB.Find(foundStudent, "id = ?", insertedStudent.ID)
	if !regexp.MustCompile(`^SELECT \* FROM STUDENTS WHERE id = :1$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	// Test Save
	result = dryrunDB.Save(&Student{ID: 2, Name: "Mary"})
	if !regexp.MustCompile(`^UPDATE STUDENTS SET name=:1 WHERE id = :2$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	// Update with conditions
	result = dryrunDB.Model(&Student{}).Where("id = ?", 1).Update("name", "hello")
	if !regexp.MustCompile(`^UPDATE STUDENTS SET name=:1 WHERE id = :2$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}
}
