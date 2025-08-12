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

	r = dryDB.Table("USER U").Select("NAME").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .NAME. FROM USER U WHERE .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("`people`").Table("`user`").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM `user`").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("PEOPLE P").Table("USER U").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM USER U WHERE .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("PEOPLE P").Table("user").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM .user. WHERE .user.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("gorm.people").Table("user").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM .user. WHERE .user.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("gorm.user").Select("NAME").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .NAME. FROM .gorm.\\..user. WHERE .user.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Select("NAME").Find(&UserWithTable{}).Statement
	if !regexp.MustCompile("SELECT .NAME. FROM .gorm.\\..user. WHERE .user.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Create(&UserWithTable{}).Statement
	if !regexp.MustCompile(`INSERT INTO .user. (.*NAME.*) VALUES (.*)`).MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) AS U", DB.Model(&User{}).Select("NAME")).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .NAME. FROM .USERS. WHERE .USERS.\\..DELETED_AT. IS NULL\\) AS U WHERE .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("NAME"), DB.Model(&Pet{}).Select("NAME")).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .NAME. FROM .USERS. WHERE .USERS.\\..DELETED_AT. IS NULL\\) AS U, \\(SELECT .NAME. FROM .PETS. WHERE .PETS.\\..DELETED_AT. IS NULL\\) AS P WHERE .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Where("name = ?", 1).Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("NAME").Where("name = ?", 2), DB.Model(&Pet{}).Where("name = ?", 4).Select("NAME")).Where("name = ?", 3).Find(&User{}).Statement
	if !regexp.MustCompile("SELECT \\* FROM \\(SELECT .NAME. FROM .USERS. WHERE name = .+ AND .USERS.\\..DELETED_AT. IS NULL\\) AS U, \\(SELECT .NAME. FROM .PETS. WHERE name = .+ AND .PETS.\\..DELETED_AT. IS NULL\\) AS P WHERE name = .+ AND name = .+ AND .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	tests.AssertEqual(t, r.Statement.Vars, []interface{}{2, 4, 1, 3})
}

func TestTableWithAllFields(t *testing.T) {
	dryDB := DB.Session(&gorm.Session{DryRun: true, QueryFields: true})
	userQuery := "SELECT .*USER.*ID.*USER.*CREATED_AT.*USER.*UPDATED_AT.*USER.*DELETED_AT.*USER.*NAME.*USER.*AGE" +
		".*USER.*BIRTHDAY.*USER.*COMPANY_ID.*USER.*MANAGER_ID.*USER.*ACTIVE.* "

	r := dryDB.Table("`user`").Find(&User{}).Statement
	if !regexp.MustCompile("" + userQuery + "FROM `user`").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("user U").Select("NAME").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .NAME. FROM user U WHERE .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("gorm.user").Select("NAME").Find(&User{}).Statement
	if !regexp.MustCompile("SELECT .NAME. FROM .gorm.\\..user. WHERE .user.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Select("NAME").Find(&UserWithTable{}).Statement
	if !regexp.MustCompile("SELECT .NAME. FROM .gorm.\\..user. WHERE .user.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {

		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Create(&UserWithTable{}).Statement
	if !regexp.MustCompile(`INSERT INTO .user. (.*NAME.*) VALUES (.*)`).MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	userQueryCharacter := "SELECT .*U.*ID.*U.*CREATED_AT.*U.*UPDATED_AT.*U.*DELETED_AT.*U.*NAME.*U.*AGE.*U.*BIRTHDAY" +
		".*U.*COMPANY_ID.*U.*MANAGER_ID.*U.*ACTIVE.* "

	r = dryDB.Table("(?) AS U", DB.Model(&User{}).Select("NAME")).Find(&User{}).Statement
	if !regexp.MustCompile("" + userQueryCharacter + "FROM \\(SELECT .NAME. FROM .USERS. WHERE .USERS.\\..DELETED_AT. IS NULL\\) AS U WHERE .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("NAME"), DB.Model(&Pet{}).Select("NAME")).Find(&User{}).Statement
	if !regexp.MustCompile("" + userQueryCharacter + "FROM \\(SELECT .NAME. FROM .USERS. WHERE .USERS.\\..DELETED_AT. IS NULL\\) AS U, \\(SELECT .NAME. FROM .PETS. WHERE .PETS.\\..DELETED_AT. IS NULL\\) AS P WHERE .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
		t.Errorf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	r = dryDB.Where("name = ?", 1).Table("(?) AS U, (?) AS P", DB.Model(&User{}).Select("NAME").Where("name = ?", 2), DB.Model(&Pet{}).Where("name = ?", 4).Select("NAME")).Where("name = ?", 3).Find(&User{}).Statement
	if !regexp.MustCompile("" + userQueryCharacter + "FROM \\(SELECT .NAME. FROM .USERS. WHERE name = .+ AND .USERS.\\..DELETED_AT. IS NULL\\) AS U, \\(SELECT .NAME. FROM .PETS. WHERE name = .+ AND .PETS.\\..DELETED_AT. IS NULL\\) AS P WHERE name = .+ AND name = .+ AND .U.\\..DELETED_AT. IS NULL").MatchString(r.Statement.SQL.String()) {
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
