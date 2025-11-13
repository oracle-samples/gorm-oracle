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

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

type UserWithTable struct {
	gorm.Model
	Name string
}

func (UserWithTable) TableName() string {
	return "gorm.user"
}

func TestTable(t *testing.T) {
	dryDB := DB.Session(&gorm.Session{DryRun: true})

	r := dryDB.Table("`user`").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM `user`").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("USER U").
		Clauses(clause.Select{Columns: []clause.Column{{Table: "U", Name: "name"}}}).
		Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .U.\\..name. FROM USER U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("USER U").
		Clauses(clause.Select{Columns: []clause.Column{{Table: "U", Name: "NAME"}}}).
		Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .U.\\..NAME. FROM USER U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("USER U").Select("name").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM USER U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("USER U").Select("Name").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM USER U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("USER U").Select("NAME").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM USER U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("`people`").Table("`user`").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM `user`").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("PEOPLE P").Table("USER U").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM USER U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("PEOPLE P").Table("user").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM .user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("gorm.people").Table("user").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM .user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("gorm.user").Select("name").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM .gorm.\\..user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Select("name").Find(&UserWithTable{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM .gorm.\\..user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Create(&UserWithTable{}).Statement
	if !regexp.MustCompile(`INSERT INTO .user. (.*name.*) VALUES (.*)`).MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) AS U", DB.Model(&User{}).Select("name")).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .name. FROM .users. WHERE .users.\\..deleted_at. IS NULL\\) AS U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("name"), DB.Model(&Pet{}).Select("name")).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .name. FROM .users. WHERE .users.\\..deleted_at. IS NULL\\) AS U, \\(SELECT .name. FROM .pets. WHERE .pets.\\..deleted_at. IS NULL\\) AS P WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Where("name = ?", 1).Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("name").Where("name = ?", 2), DB.Model(&Pet{}).Where("name = ?", 4).Select("name")).Where("name = ?", 3).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .name. FROM .users. WHERE name = .+ AND .users.\\..deleted_at. IS NULL\\) AS U, \\(SELECT .name. FROM .pets. WHERE name = .+ AND .pets.\\..deleted_at. IS NULL\\) AS P WHERE name = .+ AND name = .+ AND .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	tests.AssertEqual(t, r.Statement.Vars, []interface{}{2, 4, 1, 3})
}

func TestTableWithAllFields(t *testing.T) {
	dryDB := DB.Session(&gorm.Session{DryRun: true, QueryFields: true})
	userQuery := "SELECT .*user.*id.*user.*created_at.*user.*updated_at.*user.*deleted_at.*user.*name.*user.*age" +
		".*user.*birthday.*user.*company_id.*user.*manager_id.*user.*active.* "

	r := dryDB.Table("`user`").Find(&User{}).Statement
	if !regexp.MustCompile("" + userQuery + "FROM `user`").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("user U").Select("name").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM user U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("gorm.user").Select("name").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM .gorm.\\..user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Select("name").Find(&UserWithTable{}).Statement
	if !regexp.MustCompile("SELECT .name. FROM .gorm.\\..user. WHERE .user.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {

		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Create(&UserWithTable{}).Statement
	if !regexp.MustCompile(`INSERT INTO .user. (.*name.*) VALUES (.*)`).MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	userQueryCharacter := "SELECT .*U.*id.*U.*created_at.*U.*updated_at.*U.*deleted_at.*U.*name.*U.*age.*U.*birthday" +
		".*U.*company_id.*U.*manager_id.*U.*active.* "

	r = dryDB.Table("(?) AS U", DB.Model(&User{}).Select("name")).Find(&User{}).Statement
	if !regexp.MustCompile("" + userQueryCharacter + "FROM \\(SELECT .name. FROM .users. WHERE .users.\\..deleted_at. IS NULL\\) AS U WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("name"), DB.Model(&Pet{}).Select("name")).Find(&User{}).Statement
	if !regexp.MustCompile("" + userQueryCharacter + "FROM \\(SELECT .name. FROM .users. WHERE .users.\\..deleted_at. IS NULL\\) AS U, \\(SELECT .name. FROM .pets. WHERE .pets.\\..deleted_at. IS NULL\\) AS P WHERE .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Where("name = ?", 1).Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("name").Where("name = ?", 2), DB.Model(&Pet{}).Where("name = ?", 4).Select("name")).Where("name = ?", 3).Find(&User{}).Statement
	if !regexp.MustCompile("" + userQueryCharacter + "FROM \\(SELECT .name. FROM .users. WHERE name = .+ AND .users.\\..deleted_at. IS NULL\\) AS U, \\(SELECT .name. FROM .pets. WHERE name = .+ AND .pets.\\..deleted_at. IS NULL\\) AS P WHERE name = .+ AND name = .+ AND .U.\\..deleted_at. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	tests.AssertEqual(t, r.Statement.Vars, []interface{}{2, 4, 1, 3})
}

type UserWithTableNamer struct {
	gorm.Model
	Name string
}

func (UserWithTableNamer) TableName(namer schema.Namer) string {
	return namer.TableName("user")
}

func TestTableWithNamer(t *testing.T) {
	db, _ := gorm.Open(tests.DummyDialector{}, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "t_",
		},
	})

	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&UserWithTableNamer{}).Find(&UserWithTableNamer{})
	})

	if !regexp.MustCompile("SELECT \\* FROM `t_users`").MatchString(sql) {
		t.Errorf("Table with namer, got %v", sql)
	}
}

type mockUniqueNamingStrategy struct {
	UName string
	schema.NamingStrategy
}

func (a mockUniqueNamingStrategy) UniqueName(table, column string) string {
	return a.UName
}
