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
	"sort"
	"strings"
	"testing"

	. "github.com/oracle/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestCountWithGroup(t *testing.T) {
	DB.Create([]Company{
		{Name: "company_count_group_a"},
		{Name: "company_count_group_a"},
		{Name: "company_count_group_a"},
		{Name: "company_count_group_b"},
		{Name: "company_count_group_c"},
	})

	var count1 int64
	if err := DB.Model(&Company{}).Where("\"name\" = ?", "company_count_group_a").Group("name").Count(&count1).Error; err != nil {
		t.Errorf("Count should work, but got err %v", err)
	}
	if count1 != 1 {
		t.Errorf("Count with group should be 1, but got count: %v", count1)
	}

	var count2 int64
	if err := DB.Model(&Company{}).Where("\"name\" in ?", []string{"company_count_group_b", "company_count_group_c"}).Group("name").Count(&count2).Error; err != nil {
		t.Errorf("Count should work, but got err %v", err)
	}
	if count2 != 2 {
		t.Errorf("Count with group should be 2, but got count: %v", count2)
	}
}

func TestCount(t *testing.T) {
	var (
		user1                 = *GetUser("count-1", Config{})
		user2                 = *GetUser("count-2", Config{})
		user3                 = *GetUser("count-3", Config{})
		users                 []User
		count, count1, count2 int64
	)

	DB.Save(&user1).Save(&user2).Save(&user3)

	if err := DB.Where("\"name\" = ?", user1.Name).Or("\"name\" = ?", user3.Name).Find(&users).Count(&count).Error; err != nil {
		t.Errorf("Count should work, but got err %v", err)
	}

	if count != int64(len(users)) {
		t.Errorf("Count() method should get correct value, expect: %v, got %v", count, len(users))
	}

	if err := DB.Model(&User{}).Where("\"name\" = ?", user1.Name).Or("\"name\" = ?", user3.Name).Count(&count).Find(&users).Error; err != nil {
		t.Errorf("Count should work, but got err %v", err)
	}

	if count != int64(len(users)) {
		t.Errorf("Count() method should get correct value, expect: %v, got %v", count, len(users))
	}

	DB.Model(&User{}).Where("\"name\" = ?", user1.Name).Count(&count1).Or("\"name\" in ?", []string{user2.Name, user3.Name}).Count(&count2)
	if count1 != 1 || count2 != 3 {
		t.Errorf("multiple count in chain should works")
	}

	tx := DB.Model(&User{}).Where("\"name\" = ?", user1.Name).Session(&gorm.Session{})
	tx.Count(&count1)
	tx.Or("\"name\" in ?", []string{user2.Name, user3.Name}).Count(&count2)
	if count1 != 1 || count2 != 3 {
		t.Errorf("count after new session should works")
	}

	var count3 int64
	if err := DB.Model(&User{}).Where("\"name\" in ?", []string{user2.Name, user2.Name, user3.Name}).Group("id").Count(&count3).Error; err != nil {
		t.Errorf("Error happened when count with group, but got %v", err)
	}

	if count3 != 2 {
		t.Errorf("Should get correct count for count with group, but got %v", count3)
	}

	dryDB := DB.Session(&gorm.Session{DryRun: true})
	result := dryDB.Table("users").Select("name").Count(&count)
	if !regexp.MustCompile(`SELECT COUNT\(.name.\) FROM .*users.*`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build count with select, but got %v", result.Statement.SQL.String())
	}

	result = dryDB.Table("users").Distinct("name").Count(&count)
	if !regexp.MustCompile(`SELECT COUNT\(DISTINCT\(.name.\)\) FROM .*users.*`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("Build count with select, but got %v", result.Statement.SQL.String())
	}

	var count4 int64
	if err := DB.Table("users").Joins("LEFT JOIN \"companies\" on \"companies\".\"name\" = \"users\".\"name\"").Where("\"users\".\"name\" = ?", user1.Name).Count(&count4).Error; err != nil || count4 != 1 {
		t.Errorf("count with join, got error: %v, count %v", err, count4)
	}

	var count5 int64
	if err := DB.Table("users").Where("\"users\".\"name\" = ?", user1.Name).Order("name").Count(&count5).Error; err != nil || count5 != 1 {
		t.Errorf("count with join, got error: %v, count %v", err, count)
	}

	var count6 int64
	if err := DB.Model(&User{}).Where("\"name\" in ?", []string{user1.Name, user2.Name, user3.Name}).Select(
		"(CASE WHEN \"name\"=? THEN ? ELSE ? END) as \"name\"", "count-1", "main", "other",
	).Count(&count6).Find(&users).Error; err != nil || count6 != 3 {
		t.Fatalf("Count should work, but got err %v", err)
	}

	expects := []User{{Name: "main"}, {Name: "other"}, {Name: "other"}}
	sort.SliceStable(users, func(i, j int) bool {
		return strings.Compare(users[i].Name, users[j].Name) < 0
	})

	tests.AssertEqual(t, users, expects)

	var count7 int64
	if err := DB.Model(&User{}).Where("\"name\" in ?", []string{user1.Name, user2.Name, user3.Name}).Select(
		"(CASE WHEN \"name\"=? THEN ? ELSE ? END) as \"name\", \"age\"", "count-1", "main", "other",
	).Count(&count7).Find(&users).Error; err != nil || count7 != 3 {
		t.Fatalf("Count should work, but got err %v", err)
	}

	expects = []User{{Name: "main", Age: 18}, {Name: "other", Age: 18}, {Name: "other", Age: 18}}
	sort.SliceStable(users, func(i, j int) bool {
		return strings.Compare(users[i].Name, users[j].Name) < 0
	})

	tests.AssertEqual(t, users, expects)

	var count8 int64
	if err := DB.Model(&User{}).Where("\"name\" in ?", []string{user1.Name, user2.Name, user3.Name}).Select(
		"(CASE WHEN \"age\"=18 THEN 1 ELSE 2 END) as \"age\"", "name",
	).Count(&count8).Find(&users).Error; err != nil || count8 != 3 {
		t.Fatalf("Count should work, but got err %v", err)
	}

	expects = []User{{Name: "count-1", Age: 1}, {Name: "count-2", Age: 1}, {Name: "count-3", Age: 1}}
	sort.SliceStable(users, func(i, j int) bool {
		return strings.Compare(users[i].Name, users[j].Name) < 0
	})

	tests.AssertEqual(t, users, expects)

	var count9 int64
	if err := DB.Scopes(func(tx *gorm.DB) *gorm.DB {
		return tx.Table("users")
	}).Where("\"name\" in ?", []string{user1.Name, user2.Name, user3.Name}).Count(&count9).Find(&users).Error; err != nil || count9 != 3 {
		t.Fatalf("Count should work, but got err %v", err)
	}

	var count10 int64
	if err := DB.Model(&User{}).Select("*").Where("\"name\" in ?", []string{user1.Name, user2.Name, user3.Name}).Count(&count10).Error; err != nil || count10 != 3 {
		t.Fatalf("Count should be 3, but got count: %v err %v", count10, err)
	}

	var count11 int64
	sameUsers := make([]*User, 0)
	for i := 0; i < 3; i++ {
		sameUsers = append(sameUsers, GetUser("count-4", Config{}))
	}
	DB.Create(sameUsers)

	if err := DB.Model(&User{}).Where("\"name\" = ?", "count-4").Group("name").Count(&count11).Error; err != nil || count11 != 1 {
		t.Fatalf("Count should be 1, but got count: %v err %v", count11, err)
	}

	var count12 int64
	if err := DB.Table("users").
		Where("\"name\" in ?", []string{user1.Name, user2.Name, user3.Name}).
		Preload("Toys", func(db *gorm.DB) *gorm.DB {
			return db.Table("toys").Select("name")
		}).Count(&count12).Error; err == nil {
		t.Errorf("error should raise when using preload without schema")
	}

	var count13 int64
	if err := DB.Model(User{}).
		Where("\"name\" in ?", []string{user1.Name, user2.Name, user3.Name}).
		Preload("Toys", func(db *gorm.DB) *gorm.DB {
			return db.Table("toys").Select("name")
		}).Count(&count13).Error; err != nil {
		t.Errorf("no error should raise when using count with preload, but got %v", err)
	}
}

func TestCountOnEmptyTable(t *testing.T) {
	DB.Where("1 = 1").Delete(&User{})
	var count int64
	if err := DB.Model(&User{}).Count(&count).Error; err != nil || count != 0 {
		t.Errorf("Expected count 0 on empty table, got %v, err: %v", count, err)
	}
}

func TestCountWithHaving(t *testing.T) {
	var count int64
	err := DB.Model(&User{}).
		Select("name, COUNT(*) as total").
		Group("name").
		Having("COUNT(*) > ?", 1).
		Count(&count).Error
	if err != nil {
		t.Errorf("Error on count with having: %v", err)
	}
}

func TestCountWithSoftDelete(t *testing.T) {
	type SoftUser struct {
		gorm.Model
		Name string
	}
	DB.Migrator().DropTable(&SoftUser{})
	DB.AutoMigrate(&SoftUser{})

	user := SoftUser{Name: "SoftDelete"}
	DB.Create(&user)
	DB.Delete(&user)

	var count int64
	DB.Model(&SoftUser{}).Count(&count)
	if count != 0 {
		t.Errorf("Expected count to exclude soft-deleted records, got %v", count)
	}
}
