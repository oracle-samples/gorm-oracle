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
	"errors"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestNamedArg(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 string
		Name2 string
		Name3 string
	}

	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	namedUser := NamedUser{Name1: "jinzhu1", Name2: "jinzhu2", Name3: "jinzhu3"}
	DB.Create(&namedUser)

	var result NamedUser
	DB.First(&result, "\"name1\" = @name OR \"name2\" = @name OR \"name3\" = @name", sql.Named("name", "jinzhu2"))

	tests.AssertEqual(t, result, namedUser)

	var result2 NamedUser
	DB.Where("\"name1\" = @name OR \"name2\" = @name OR \"name3\" = @name", sql.Named("name", "jinzhu2")).First(&result2)

	tests.AssertEqual(t, result2, namedUser)

	var result3 NamedUser
	DB.Where("\"name1\" = @name OR \"name2\" = @name OR \"name3\" = @name", map[string]interface{}{"name": "jinzhu2"}).First(&result3)

	tests.AssertEqual(t, result3, namedUser)

	var result4 NamedUser
	if err := DB.Raw("SELECT * FROM \"named_users\" WHERE \"name1\" = @name OR \"name2\" = @name2 OR \"name3\" = @name", sql.Named("name", "jinzhu-none"), sql.Named("name2", "jinzhu2")).Find(&result4).Error; err != nil {
		t.Errorf("failed to update with named arg")
	}

	tests.AssertEqual(t, result4, namedUser)

	if err := DB.Exec("UPDATE \"named_users\" SET \"name1\" = @name, \"name2\" = @name2, \"name3\" = @name", sql.Named("name", "jinzhu-new"), sql.Named("name2", "jinzhu-new2")).Error; err != nil {
		t.Errorf("failed to update with named arg")
	}

	namedUser.Name1 = "jinzhu-new"
	namedUser.Name2 = "jinzhu-new2"
	namedUser.Name3 = "jinzhu-new"

	var result5 NamedUser
	if err := DB.Raw("SELECT * FROM \"named_users\" WHERE (\"name1\" = @name AND \"name3\" = @name) AND \"name2\" = @name2", map[string]interface{}{"name": "jinzhu-new", "name2": "jinzhu-new2"}).Find(&result5).Error; err != nil {
		t.Errorf("failed to update with named arg")
	}

	tests.AssertEqual(t, result5, namedUser)

	var result6 NamedUser
	if err := DB.Raw(`SELECT * FROM "named_users" WHERE ("name1" = @name
	AND "name3" = @name) AND "name2" = @name2`, map[string]interface{}{"name": "jinzhu-new", "name2": "jinzhu-new2"}).Find(&result6).Error; err != nil {
		t.Errorf("failed to update with named arg")
	}

	tests.AssertEqual(t, result6, namedUser)

	var result7 NamedUser
	if err := DB.Where("\"name1\" = @name OR \"name2\" = @name", sql.Named("name", "jinzhu-new")).Where("\"name3\" = 'jinzhu-new3'").First(&result7).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should return record not found error, but got %v", err)
	}

	DB.Delete(&namedUser)

	var result8 NamedUser
	if err := DB.Where("\"name1\" = @name OR \"name2\" = @name", map[string]interface{}{"name": "jinzhu-new"}).First(&result8).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should return record not found error, but got %v", err)
	}
}

func TestNamedArgMultipleSameParamRefs(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 string
	}
	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	user := NamedUser{Name1: "multi-ref"}
	DB.Create(&user)

	var result NamedUser
	if err := DB.Raw(`SELECT * FROM "named_users" WHERE @name = @name AND "name1" = @name`,
		sql.Named("name", "multi-ref")).Scan(&result).Error; err != nil {
		t.Errorf("failed with multiple same param refs: %v", err)
	}
	tests.AssertEqual(t, result, user)
}

func TestNamedArgNullValues(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 *string
	}
	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	DB.Create(&NamedUser{Name1: nil})

	var count int64
	if err := DB.Raw(`SELECT count(*) FROM "named_users" WHERE (:name IS NULL AND "name1" IS NULL)`,
		sql.Named("name", nil)).Scan(&count).Error; err != nil {
		t.Errorf("failed null param test: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 null record, got %d", count)
	}
}

func TestNamedArgMixedNamedAndMapParams(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 string
		Name2 string
	}
	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	user := NamedUser{Name1: "n1", Name2: "n2"}
	DB.Create(&user)

	var result NamedUser
	if err := DB.Raw(`SELECT * FROM "named_users" WHERE "name1" = @name1 AND "name2" = @name2`,
		sql.Named("name1", "n1"), map[string]interface{}{"name2": "n2"}).Scan(&result).Error; err != nil {
		t.Errorf("failed mixed param test: %v", err)
	}
	tests.AssertEqual(t, result, user)
}

func TestNamedArgUnusedParameter(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 string
	}
	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	user := NamedUser{Name1: "unused"}
	DB.Create(&user)

	var result NamedUser
	if err := DB.Raw(`SELECT * FROM "named_users" WHERE "name1" = @name1`,
		sql.Named("name1", "unused"), sql.Named("extra", "notused")).Scan(&result).Error; err != nil {
		t.Errorf("failed unused param test: %v", err)
	}
	tests.AssertEqual(t, result, user)
}

func TestNamedArgCaseSensitivity(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 string
	}
	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	user := NamedUser{Name1: "CaseTest"}
	DB.Create(&user)

	var result NamedUser
	if err := DB.Raw(`SELECT * FROM "named_users" WHERE "name1" = @Name`,
		sql.Named("Name", "CaseTest")).Scan(&result).Error; err != nil {
		t.Errorf("failed case sensitivity test: %v", err)
	}
	tests.AssertEqual(t, result, user)
}

func TestNamedArgInClause(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 string
	}
	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	user1 := NamedUser{Name1: "in1"}
	user2 := NamedUser{Name1: "in2"}
	DB.Create(&user1)
	DB.Create(&user2)

	var results []NamedUser
	if err := DB.Raw(`SELECT * FROM "named_users" WHERE "name1" IN (@n1, @n2)`,
		sql.Named("n1", "in1"), sql.Named("n2", "in2")).Scan(&results).Error; err != nil {
		t.Errorf("failed IN clause test: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestNamedArgReservedWordParam(t *testing.T) {
	type NamedUser struct {
		gorm.Model
		Name1 string
	}
	DB.Migrator().DropTable(&NamedUser{})
	DB.AutoMigrate(&NamedUser{})

	user := NamedUser{Name1: "reserved"}
	DB.Create(&user)

	var result NamedUser
	if err := DB.Raw(`SELECT * FROM "named_users" WHERE "name1" = @order`,
		sql.Named("order", "reserved")).Scan(&result).Error; err != nil {
		t.Errorf("failed reserved word param test: %v", err)
	}
	tests.AssertEqual(t, result, user)
}
