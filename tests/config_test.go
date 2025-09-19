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
	"errors"
	"regexp"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
)

type FolderData struct {
	ID         string           `gorm:"primaryKey;column:folder_id"`
	Name       string           `gorm:"column:folder_nm"`
	Properties []FolderProperty `gorm:"foreignKey:ID;PRELOAD:false"`
}

func (FolderData) TableName() string {
	return "folder_data"
}

type FolderProperty struct {
	Seq   uint64 `gorm:"autoIncrement"`
	ID    string `gorm:"primaryKey;column:folder_id"`
	Key   string `gorm:"primaryKey;unique"`
	Value string
}

func (FolderProperty) TableName() string {
	return "folder_property"
}

func TestSkipQuoteIdentifiersMigrator(t *testing.T) {
	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	db.Migrator().DropTable(&FolderData{}, &FolderProperty{})
	db.Migrator().CreateTable(&FolderData{}, &FolderProperty{})

	folderDataTN := "FOLDER_DATA"
	if !db.Migrator().HasTable(folderDataTN) {
		t.Errorf("Failed to get table: %s", folderDataTN)
	}

	if !db.Migrator().HasColumn(folderDataTN, "FOLDER_ID") {
		t.Errorf("Failed to get column: FOLDER_ID")
	}

	if !db.Migrator().HasColumn(folderDataTN, "FOLDER_NM") {
		t.Errorf("Failed to get column: FOLDER_NM")
	}

	folderPropertyTN := "FOLDER_PROPERTY"
	if !db.Migrator().HasTable(folderPropertyTN) {
		t.Errorf("Failed to get table: %s", folderPropertyTN)
	}

	if !db.Migrator().HasColumn(folderPropertyTN, "SEQ") {
		t.Errorf("Failed to get column: SEQ")
	}

	if !db.Migrator().HasColumn(folderPropertyTN, "FOLDER_ID") {
		t.Errorf("Failed to get column: FOLDER_ID")
	}

	if !db.Migrator().HasColumn(folderPropertyTN, "KEY") {
		t.Errorf("Failed to get column: KEY")
	}

	if !db.Migrator().HasColumn(folderPropertyTN, "VALUE") {
		t.Errorf("Failed to get column: VALUE")
	}
}

func TestSkipQuoteIdentifiers(t *testing.T) {
	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	db.Migrator().DropTable(&FolderData{}, &FolderProperty{})
	db.Migrator().CreateTable(&FolderData{}, &FolderProperty{})

	id := uuid.New().String()
	folder := FolderData{
		ID:   id,
		Name: "My Folder",
		Properties: []FolderProperty{
			{
				ID:    id,
				Key:   "foo1",
				Value: "bar1",
			},
			{
				ID:    id,
				Key:   "foo2",
				Value: "bar2",
			},
		},
	}

	if err := db.Create(&folder).Error; err != nil {
		t.Errorf("Failed to insert data, got %v", err)
	}

	createdFolder := FolderData{}
	if err := db.Model(&FolderData{}).Preload("Properties").First(&createdFolder).Error; err != nil {
		t.Errorf("Failed to query data, got %v", err)
	}

	CheckFolderData(t, createdFolder, folder)

	createdFolder.Properties[1].Value = "baz1"
	createdFolder.Properties = append(createdFolder.Properties, FolderProperty{
		ID:    id,
		Key:   "foo3",
		Value: "bar3",
	})
	createdFolder.Properties = append(createdFolder.Properties, FolderProperty{
		ID:    id,
		Key:   "foo4",
		Value: "bar4",
	})
	db.Save(&createdFolder)

	updatedFolder := FolderData{}
	if err := db.Model(&FolderData{}).Preload("Properties").First(&updatedFolder).Error; err != nil {
		t.Errorf("Failed to query data, got %v", err)
	}

	CheckFolderData(t, updatedFolder, createdFolder)

	if err := db.Select(clause.Associations).Delete(&createdFolder).Error; err != nil {
		t.Errorf("Failed to delete data, got %v", err)
	}

	result := FolderData{}
	if err := db.Where("folder_id = ?", createdFolder.ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}
}

func CheckFolderData(t *testing.T, folderData FolderData, expect FolderData) {
	tests.AssertObjEqual(t, folderData, expect, "ID", "Name")
	t.Run("Properties", func(t *testing.T) {
		if len(folderData.Properties) != len(expect.Properties) {
			t.Fatalf("properties should equal, expect: %v, got %v", len(expect.Properties), len(folderData.Properties))
		}

		sort.Slice(folderData.Properties, func(i, j int) bool {
			return folderData.Properties[i].ID > folderData.Properties[j].ID
		})

		sort.Slice(expect.Properties, func(i, j int) bool {
			return expect.Properties[i].ID > expect.Properties[j].ID
		})

		for idx, property := range folderData.Properties {
			tests.AssertObjEqual(t, property, expect.Properties[idx], "Seq", "ID", "Key", "Value")
		}
	})
}

func TestSkipQuoteIdentifiersSQL(t *testing.T) {
	type Student struct {
		ID   uint
		Name string
	}

	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}
	dryrunDB := db.Session(&gorm.Session{DryRun: true})

	insertedStudent := Student{ID: 1, Name: "John"}
	result := dryrunDB.Model(&Student{}).Create(&insertedStudent)

	if !regexp.MustCompile(`^INSERT INTO students \(name,id\) VALUES \(:1,:2\)$`).MatchString(result.Statement.SQL.String()) {
		t.Errorf("invalid insert SQL, got %v", result.Statement.SQL.String())
	}

	// Test First
	var firstStudent Student
	result = dryrunDB.First(&firstStudent)

	if !regexp.MustCompile(`^SELECT \* FROM students ORDER BY students\.id FETCH NEXT 1 ROW ONLY$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	// Test Find
	var foundStudent Student
	result = dryrunDB.Find(foundStudent, "id = ?", insertedStudent.ID)
	if !regexp.MustCompile(`^SELECT \* FROM students WHERE id = :1$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	// Test Save
	result = dryrunDB.Save(&Student{ID: 2, Name: "Mary"})
	if !regexp.MustCompile(`^UPDATE students SET name=:1 WHERE id = :2$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	// Update with conditions
	result = dryrunDB.Model(&Student{}).Where("id = ?", 1).Update("name", "hello")
	if !regexp.MustCompile(`^UPDATE students SET name=:1 WHERE id = :2$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}
}
