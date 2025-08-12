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

	. "github.com/oracle/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestDistinct(t *testing.T) {
	users := []User{
		*GetUser("distinct", Config{}),
		*GetUser("distinct", Config{}),
		*GetUser("distinct", Config{}),
		*GetUser("distinct-2", Config{}),
		*GetUser("distinct-3", Config{}),
	}
	users[0].Age = 20

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create users: %v", err)
	}

	var names []string
	DB.Table("users").Where("\"name\" like ?", "distinct%").Order("\"name\"").Pluck("name", &names)
	tests.AssertEqual(t, names, []string{"distinct", "distinct", "distinct", "distinct-2", "distinct-3"})

	var names1 []string
	DB.Model(&User{}).Where("\"name\" like ?", "distinct%").Distinct().Order("\"name\"").Pluck("Name", &names1)

	tests.AssertEqual(t, names1, []string{"distinct", "distinct-2", "distinct-3"})

	var names2 []string
	DB.Scopes(func(db *gorm.DB) *gorm.DB {
		return db.Table("users")
	}).Where("\"name\" like ?", "distinct%").Order("\"name\"").Pluck("name", &names2)
	tests.AssertEqual(t, names2, []string{"distinct", "distinct", "distinct", "distinct-2", "distinct-3"})

	var results []User
	if err := DB.Distinct("name", "age").Where("\"name\" like ?", "distinct%").Order("\"name\", \"age\" desc").Find(&results).Error; err != nil {
		t.Errorf("failed to query users, got error: %v", err)
	}

	expects := []User{
		{Name: "distinct", Age: 20},
		{Name: "distinct", Age: 18},
		{Name: "distinct-2", Age: 18},
		{Name: "distinct-3", Age: 18},
	}

	if len(results) != 4 {
		t.Fatalf("invalid results length found, expects: %v, got %v", len(expects), len(results))
	}

	for idx, expect := range expects {
		tests.AssertObjEqual(t, results[idx], expect, "Name", "Age")
	}

	var count int64
	if err := DB.Model(&User{}).Where("\"name\" like ?", "distinct%").Count(&count).Error; err != nil || count != 5 {
		t.Errorf("failed to query users count, got error: %v, count: %v", err, count)
	}

	if err := DB.Model(&User{}).Distinct("name").Where("\"name\" like ?", "distinct%").Count(&count).Error; err != nil || count != 3 {
		t.Errorf("failed to query users count, got error: %v, count %v", err, count)
	}

	dryDB := DB.Session(&gorm.Session{DryRun: true})
	r := dryDB.Distinct("u.id, u.*").Table("user_speaks s").Joins("inner join users u on u.id = s.user_id").Where("s.language_code ='US' or s.language_code ='ES'").Find(&User{})
	if !regexp.MustCompile(`SELECT DISTINCT u\.id, u\.\* FROM user_speaks s inner join users u`).MatchString(r.Statement.SQL.String()) {
		t.Fatalf("Build Distinct with u.*, but got %v", r.Statement.SQL.String())
	}
}

func TestDistinctWithVaryingCase(t *testing.T) {
	RunMigrations()
	users := []User{
		{Name: "Alpha"},
		{Name: "alpha"},
		{Name: "BETA"},
		{Name: "beta"},
		{Name: "Gamma"},
	}
	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	var results []string
	if err := DB.
		Table("users").
		Select("DISTINCT UPPER(\"name\") as name").
		Pluck("name", &results).Error; err != nil {
		t.Fatalf("failed to query distinct upper names: %v", err)
	}

	tests.AssertEqual(t, results, []string{"ALPHA", "BETA", "GAMMA"})
}

func TestDistinctComputedColumn(t *testing.T) {
	t.Skip()
	type UserWithComputationColumn struct {
		ID   int64 `gorm:"primary_key"`
		Name string
		Col  int64
	}

	if err := DB.Migrator().DropTable(&UserWithComputationColumn{}); err != nil {
		t.Fatalf("failed to drop table: %v", err)
	}
	if err := DB.AutoMigrate(&UserWithComputationColumn{}); err != nil {
		t.Fatalf("failed to migrate table: %v", err)
	}

	records := []UserWithComputationColumn{
		{Name: "U1", Col: 5000},
		{Name: "U2", Col: 5000},
		{Name: "U3", Col: 6000},
	}
	if err := DB.Create(&records).Error; err != nil {
		t.Fatalf("failed to create users: %v", err)
	}

	var computedRecords []int
	if err := DB.
		Table("USER_WITH_COMPUTATION_COLUMNS").
		Select("DISTINCT col * 12 as Computed_Column").
		Order("Computed_Column").
		Pluck("Computed_Column", &computedRecords).Error; err != nil {
		t.Fatalf("failed to query distinct Computed Columns: %v", err)
	}

	tests.AssertEqual(t, computedRecords, []int{60000, 72000})
}

func TestDistinctWithAggregation(t *testing.T) {
	t.Skip()
	type UserWithComputationColumn struct {
		ID   int64 `gorm:"primaryKey"`
		Name string
		Col  int64
	}

	if err := DB.Migrator().DropTable(&UserWithComputationColumn{}); err != nil {
		t.Fatalf("failed to drop table: %v", err)
	}
	if err := DB.AutoMigrate(&UserWithComputationColumn{}); err != nil {
		t.Fatalf("failed to migrate table: %v", err)
	}

	records := []UserWithComputationColumn{
		{Name: "U1", Col: 5000},
		{Name: "U2", Col: 5000},
		{Name: "U3", Col: 6000},
		{Name: "U4", Col: 7000},
		{Name: "U5", Col: 7000},
	}

	if err := DB.Create(&records).Error; err != nil {
		t.Fatalf("failed to insert test users: %v", err)
	}

	var result struct {
		Sum   int64
		Avg   float64
		Count int64
	}

	err := DB.
		Table("USER_WITH_COMPUTATION_COLUMNS").
		Select(`
			SUM(DISTINCT col) AS Sum,
			AVG(DISTINCT col) AS Avg,
			COUNT(DISTINCT col) AS Count
		`).Scan(&result).Error

	if err != nil {
		t.Fatalf("failed to execute aggregate DISTINCT query: %v", err)
	}

	tests.AssertEqual(t, result.Sum, int64(18000))
	tests.AssertEqual(t, result.Avg, float64(6000))
	tests.AssertEqual(t, result.Count, int64(3))
}
