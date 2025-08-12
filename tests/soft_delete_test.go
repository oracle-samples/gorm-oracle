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
	"encoding/json"
	"errors"
	"regexp"
	"testing"
	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm/clause"

	"gorm.io/gorm"
)

func TestSoftDelete(t *testing.T) {
	user := *GetUser("SoftDelete", Config{})
	DB.Save(&user)

	var count int64
	var age uint

	if DB.Model(&User{}).Where("\"name\" = ?", user.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 1, count)
	}

	if DB.Model(&User{}).Select("age").Where("\"name\" = ?", user.Name).Scan(&age).Error != nil || age != user.Age {
		t.Errorf("Age soft deleted record, expects: %v, got: %v", 0, age)
	}

	if err := DB.Delete(&user).Error; err != nil {
		t.Fatalf("No error should happen when soft delete user, but got %v", err)
	}

	if sql.NullTime(user.DeletedAt).Time.IsZero() {
		t.Fatalf("user's deleted at is zero")
	}

	sql := DB.Session(&gorm.Session{DryRun: true}).Delete(&user).Statement.SQL.String()
	if !regexp.MustCompile(`UPDATE .users. SET .deleted_at.=.* WHERE .users.\..id. = .* AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = DB.Session(&gorm.Session{DryRun: true}).Table("user u").Select("name").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`SELECT .name. FROM user u WHERE .u.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Errorf("Table with escape character, got %v", sql)
	}

	if DB.First(&User{}, "\"name\" = ?", user.Name).Error == nil {
		t.Errorf("expected soft deleted user to be excluded from default queries, but it was found")
	}

	count = 0
	if DB.Model(&User{}).Where("\"name\" = ?", user.Name).Count(&count).Error != nil || count != 0 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 0, count)
	}

	age = 0
	if DB.Model(&User{}).Select("age").Where("\"name\" = ?", user.Name).Scan(&age).Error != nil || age != 0 {
		t.Errorf("Age soft deleted record, expects: %v, got: %v", 0, age)
	}

	if err := DB.Unscoped().First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
		t.Errorf("Should find soft deleted record with Unscoped, but got err %s", err)
	}

	count = 0
	if DB.Unscoped().Model(&User{}).Where("\"name\" = ?", user.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, count: %v", 1, count)
	}

	age = 0
	if DB.Unscoped().Model(&User{}).Select("age").Where("\"name\" = ?", user.Name).Scan(&age).Error != nil || age != user.Age {
		t.Errorf("Age soft deleted record, expects: %v, got: %v", 0, age)
	}

	DB.Unscoped().Delete(&user)
	if err := DB.Unscoped().First(&User{}, "\"name\" = ?", user.Name).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("Can't find permanently deleted record")
	}
}

func TestDeletedAtUnMarshal(t *testing.T) {
	expected := &gorm.Model{}
	b, _ := json.Marshal(expected)

	result := &gorm.Model{}
	_ = json.Unmarshal(b, result)
	if result.DeletedAt != expected.DeletedAt {
		t.Errorf("Failed, result.DeletedAt: %v is not same as expected.DeletedAt: %v", result.DeletedAt, expected.DeletedAt)
	}
}

func TestDeletedAtOneOr(t *testing.T) {
	actualSQL := DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Or("id = ?", 1).Find(&User{})
	})

	if !regexp.MustCompile(` WHERE id = 1 AND .users.\..deleted_at. IS NULL`).MatchString(actualSQL) {
		t.Fatalf("invalid sql generated, got %v", actualSQL)
	}
}

func TestSoftDeleteZeroValue(t *testing.T) {
	type SoftDeleteBook struct {
		ID        uint
		Name      string
		Pages     uint
		DeletedAt gorm.DeletedAt `gorm:"zeroValue:'1970-01-01 00:00:01'"`
	}
	DB.Migrator().DropTable(&SoftDeleteBook{})
	if err := DB.AutoMigrate(&SoftDeleteBook{}); err != nil {
		t.Fatalf("failed to auto migrate soft delete table")
	}

	book := SoftDeleteBook{Name: "jinzhu", Pages: 10}
	DB.Save(&book)

	var count int64
	if DB.Model(&SoftDeleteBook{}).Where("\"name\" = ?", book.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 1, count)
	}

	var pages uint
	if DB.Model(&SoftDeleteBook{}).Select("pages").Where("\"name\" = ?", book.Name).Scan(&pages).Error != nil || pages != book.Pages {
		t.Errorf("Pages soft deleted record, expects: %v, got: %v", 0, pages)
	}

	if err := DB.Delete(&book).Error; err != nil {
		t.Fatalf("No error should happen when soft delete user, but got %v", err)
	}

	zeroTime, _ := Parse("1970-01-01 00:00:01")

	if book.DeletedAt.Time.Equal(zeroTime) {
		t.Errorf("book's deleted at should not be zero, DeletedAt: %v", book.DeletedAt)
	}

	if DB.First(&SoftDeleteBook{}, "\"name\" = ?", book.Name).Error == nil {
		t.Errorf("expected soft deleted record to be excluded, but it was found")
	}

	count = 0
	if DB.Model(&SoftDeleteBook{}).Where("\"name\" = ?", book.Name).Count(&count).Error != nil || count != 0 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 0, count)
	}

	pages = 0
	if err := DB.Model(&SoftDeleteBook{}).Select("pages").Where("\"name\" = ?", book.Name).Scan(&pages).Error; err != nil || pages != 0 {
		t.Fatalf("Age soft deleted record, expects: %v, got: %v, err %v", 0, pages, err)
	}

	if err := DB.Unscoped().First(&SoftDeleteBook{}, "\"name\" = ?", book.Name).Error; err != nil {
		t.Errorf("Should find soft deleted record with Unscoped, but got err %s", err)
	}

	count = 0
	if DB.Unscoped().Model(&SoftDeleteBook{}).Where("\"name\" = ?", book.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, count: %v", 1, count)
	}

	pages = 0
	if DB.Unscoped().Model(&SoftDeleteBook{}).Select("pages").Where("\"name\" = ?", book.Name).Scan(&pages).Error != nil || pages != book.Pages {
		t.Errorf("Age soft deleted record, expects: %v, got: %v", 0, pages)
	}

	DB.Unscoped().Delete(&book)
	if err := DB.Unscoped().First(&SoftDeleteBook{}, "\"name\" = ?", book.Name).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("Can't find permanently deleted record")
	}
}
func TestSoftDeleteWithPreload(t *testing.T) {
	user := GetUser("soft_delete_preload", Config{Account: true})
	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user with preload: %v", err)
	}

	if err := DB.Delete(user).Error; err != nil {
		t.Fatalf("failed to soft delete user: %v", err)
	}

	var result User
	err := DB.Preload("Account").First(&result, user.ID).Error
	if err == nil {
		t.Errorf("expected soft deleted user to be excluded with preload, but it was found")
	}

	err = DB.Unscoped().Preload("Pets").First(&result, "\"id\" = ?", user.ID).Error
	if err != nil {
		t.Errorf("should preload soft deleted user with unscoped, got %v", err)
	}
}
func TestSoftDeleteWithWhereClause(t *testing.T) {
	user := *GetUser("inline_soft_delete", Config{})
	DB.Save(&user)

	if err := DB.Where("\"name\" = ?", user.Name).Delete(&User{}).Error; err != nil {
		t.Errorf("inline condition delete should not error, got: %v", err)
	}

	var found User
	if err := DB.First(&found, user.ID).Error; err == nil {
		t.Errorf("expected soft deleted record to be excluded")
	}
}
func TestSoftDeleteTimeFilter(t *testing.T) {
	user := *GetUser("time_filter", Config{})
	DB.Create(&user)
	DB.Delete(&user)

	var result User

	year, month, day := time.Now().Date()
	endOfDay := time.Date(year, month, day, 23, 59, 59, int(time.Second-time.Nanosecond), time.Now().Location())
	err := DB.Unscoped().Where("\"deleted_at\" <= ?", endOfDay).First(&result).Error
	if err != nil {
		t.Errorf("expected to find soft deleted record by deleted_at, but got: %v", err)
	}
}
func TestSoftDeleteIdempotent(t *testing.T) {
	user := *GetUser("idempotent_soft_delete", Config{})
	DB.Create(&user)
	DB.Delete(&user)

	if err := DB.Delete(&user).Error; err != nil {
		t.Errorf("re-deleting soft deleted user should not error, got: %v", err)
	}
}
func TestSoftDeleteWithClause(t *testing.T) {
	user := *GetUser("soft_clause", Config{})
	DB.Create(&user)

	if err := DB.Clauses(clause.Where{Exprs: []clause.Expression{
		clause.Eq{Column: "name", Value: user.Name},
	}}).Delete(&User{}).Error; err != nil {
		t.Errorf("soft delete with clause failed: %v", err)
	}

	var count int64
	DB.Model(&User{}).Where("name = ?", user.Name).Count(&count)
	if count != 0 {
		t.Errorf("soft deleted record should be excluded, got count: %v", count)
	}
}
func TestSoftDeleteWithCompositeKey(t *testing.T) {
	type CompositeModel struct {
		ID1       uint `gorm:"primaryKey"`
		ID2       uint `gorm:"primaryKey"`
		Name      string
		DeletedAt gorm.DeletedAt
	}

	DB.Migrator().DropTable(&CompositeModel{})
	DB.AutoMigrate(&CompositeModel{})

	record := CompositeModel{ID1: 1, ID2: 2, Name: "record1"}
	DB.Create(&record)

	DB.Delete(&record)

	var result CompositeModel
	if err := DB.First(&result, "\"id1\" = ? AND \"id2\" = ?", 1, 2).Error; err == nil {
		t.Errorf("record should be soft deleted, but found")
	}

	if err := DB.Unscoped().First(&result, "\"id1\" = ? AND \"id2\" = ?", 1, 2).Error; err != nil {
		t.Errorf("record should be found with unscoped, got %v", err)
	}
}
func TestSoftDeletedRecordReinsert(t *testing.T) {
	user := *GetUser("soft_reinsert", Config{})
	DB.Create(&user)
	DB.Delete(&user)

	if err := DB.Create(&user).Error; err == nil {
		t.Errorf("shouldn't allow inserting the same record after soft delete")
	}

	user2 := *GetUser("soft_reinsert", Config{})
	if err := DB.Create(&user2).Error; err != nil {
		t.Errorf("should allow inserting new record after soft delete, got err %v", err)
	}
}
