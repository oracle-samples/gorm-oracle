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
	"fmt"
	"regexp"
	"strings"
	"testing"

	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestRow(t *testing.T) {
	user1 := User{Name: "RowUser1", Age: 1}
	user2 := User{Name: "RowUser2", Age: 10}
	user3 := User{Name: "RowUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	row := DB.Table("users").Where("\"name\" = ?", user2.Name).Select("\"age\"").Row()

	var age int64
	if err := row.Scan(&age); err != nil {
		t.Fatalf("Failed to scan age, got %v", err)
	}

	if age != 10 {
		t.Errorf("Scan with Row, age expects: %v, got %v", user2.Age, age)
	}

	table := "\"users\""

	DB.Table(table).Where(map[string]interface{}{"name": user2.Name}).Update("age", 20)

	row2 := DB.Table(table+" u").Where("u.\"name\" = ?", user2.Name).Select("\"age\"").Row()
	if err := row2.Scan(&age); err != nil {
		t.Fatalf("Failed to scan age, got %v", err)
	}

	if age != 20 {
		t.Errorf("Scan with Row, age expects: %v, got %v", user2.Age, age)
	}

	row3 := DB.Table(table+" u").Where("u.\"name\" = @username", map[string]interface{}{"username": user1.Name}).Select("\"age\"").Row()
	if err := row3.Scan(&age); err != nil {
		t.Fatalf("Failed to scan age, got %v", err)
	}

	if age != 1 {
		t.Fatalf("Scan with Row, age expects: %v, got %v", user2.Age, age)
	}

	row4 := DB.Table(table+" \"u\"").Where("\"u\".\"name\" = ?", user3.Name).Select("\"age\"").Row()
	if err := row4.Scan(&age); err != nil {
		t.Fatalf("Failed to scan age, got %v", err)
	}

	if age != 20 {
		t.Fatalf("Scan with Row, age expects: %v, got %v", user1.Age, age)
	}

	row5 := DB.Table(table+" \"u\"").Where("\"u\".\"name\" = @p1", map[string]interface{}{"p1": user1.Name}).Select("\"age\"").Row()
	if err := row5.Scan(&age); err != nil {
		t.Fatalf("Failed to scan age, got %v", err)
	}

	if age != 1 {
		t.Fatalf("Scan with Row, age expects: %v, got %v", user1.Age, age)
	}

	row6 := DB.Table(table+" AS \"u\"").Where("\"u\".\"name\" = ?", user2.Name).Select("\"u\".\"age\"").Row()
	if err := row6.Scan(&age); err == nil {
		t.Fatalf("Should report error because AS is not supported for table name but got null")
	}

	row7 := DB.Table(table+" \"u\"").Where("u.\"name\" = ?", user2.Name).Select("\"age\"").Row()
	if err := row7.Scan(&age); err == nil {
		t.Fatalf("Should report error but got null error")
	}
}

func TestRows(t *testing.T) {
	user1 := User{Name: "RowsUser1", Age: 1}
	user2 := User{Name: "RowsUser2", Age: 10}
	user3 := User{Name: "RowsUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	rows, err := DB.Table("users").Where("\"name\" = ? or \"name\" = ?", user2.Name, user3.Name).Select("\"name\", \"age\"").Rows()
	if err != nil {
		t.Errorf("No error should happen, got %v", err)
	}
	if rows != nil {
		defer rows.Close()
	}

	count := 0
	for rows.Next() {
		var name string
		var age int64
		rows.Scan(&name, &age)
		count++
	}

	if count != 2 {
		t.Errorf("Should found two records, but got %d", count)
	}

	rows, err = DB.Table("\"users\" \"u\"").Where("\"u\".\"name\" like ?", user3.Name+"%").Select("\"age\"").Rows()
	if err != nil {
		t.Errorf("No error should happen, got %v", err)
	}
	if rows != nil {
		defer rows.Close()
	}

	count = 0
	var age uint
	for rows.Next() {
		if err := rows.Scan(&age); err != nil && age != user3.Age {
			t.Errorf("Scan with Rows, age expects: %v, got %v", user3.Age, age)
		}
		count++
	}
	if count != 1 {
		t.Errorf("Should found one records, but got %d", count)
	}

	rows, err = DB.Table("\"users\" u").Where("upper(u.\"name\") like upper('RowsUser%')").Select("\"age\"").Rows()
	if err != nil {
		t.Errorf("No error should happen, got %v", err)
	}
	if rows != nil {
		defer rows.Close()
	}

	count = 0
	for rows.Next() {
		count++
	}
	if count != 3 {
		t.Errorf("Should found three records, but got %d", count)
	}

	rows, err = DB.Table("\"users\" u").Where("upper(\"u\".\"name\") like upper('RowsUser%')").Select("\"age\"").Rows()
	if err == nil {
		t.Errorf("Should report error but got null error")
	}
	if rows != nil {
		t.Errorf("Rows should be nil, got %v", rows)
	}

}

func TestRaw(t *testing.T) {
	user1 := User{Name: "ExecRawSqlUser1", Age: 1}
	user2 := User{Name: "ExecRawSqlUser2", Age: 10}
	user3 := User{Name: "ExecRawSqlUser3", Age: 20}
	DB.Save(&user1).Save(&user2).Save(&user3)

	type result struct {
		Name  string
		Email string
		Age   int64
	}

	var results []result
	DB.Raw("SELECT \"name\", \"age\" FROM \"users\" WHERE \"name\" = ? or \"name\" = ?", user2.Name, user3.Name).Scan(&results)
	if len(results) != 2 || results[0].Name != user2.Name || results[1].Name != user3.Name {
		t.Errorf("Raw with scan")
	}

	rows, _ := DB.Raw("select \"name\", \"age\" from \"users\" where \"name\" = ?", user3.Name).Rows()
	if rows != nil {
		defer rows.Close()
	}

	count := 0
	for rows.Next() {
		count++
	}
	if count != 1 {
		t.Errorf("Raw with Rows should find one record with name 3")
	}

	DB.Exec("update \"users\" set \"name\"=? where \"name\" in (?)", "jinzhu-raw", []string{user1.Name, user2.Name, user3.Name})
	if DB.Where("\"name\" in (?)", []string{user1.Name, user2.Name, user3.Name}).First(&User{}).Error != gorm.ErrRecordNotFound {
		t.Error("Raw sql to update records")
	}

	DB.Exec("update \"users\" set \"age\"=? where \"name\" = ?", gorm.Expr("\"age\" * ? + ?", 2, 10), "jinzhu-raw")

	var age int
	DB.Raw("select sum(\"age\") from \"users\" where \"name\" = ?", "jinzhu-raw").Scan(&age)

	if age != ((1+10+20)*2 + 30) {
		t.Errorf("Invalid age, got %v", age)
	}

	DB.Exec("update \"users\" u set u.\"age\"=? where \"name\" = ? and ROWNUM <= 1", 100, "jinzhu-raw")

	DB.Raw("SELECT \"name\", \"age\" FROM \"users\" WHERE \"name\" = ? and ROWNUM <= 1", "jinzhu-raw").Scan(&results)
	if len(results) != 1 || results[0].Name != "jinzhu-raw" || results[0].Age != 100 {
		t.Errorf("Raw with scan")
	}

}

func TestRowsWithGroup(t *testing.T) {
	users := []User{
		{Name: "having_user_1", Age: 1},
		{Name: "having_user_2", Age: 10},
		{Name: "having_user_1", Age: 20},
		{Name: "having_user_1", Age: 30},
	}

	DB.Create(&users)

	rows, err := DB.Select("\"name\", count(*) as \"total\"").Table("users").Group("name").Having("\"name\" IN ?", []string{users[0].Name, users[1].Name}).Rows()
	if err != nil {
		t.Fatalf("got error %v", err)
	}
	if rows != nil {
		defer rows.Close()
	}

	for rows.Next() {
		var name string
		var total int64
		rows.Scan(&name, &total)

		if name == users[0].Name && total != 3 {
			t.Errorf("Should have three user having name %v", users[0].Name)
		} else if name == users[1].Name && total != 1 {
			t.Errorf("Should have one users having name %v", users[1].Name)
		}
	}

	rows2, err2 := DB.Table("users").Select("\"name\", count(*) as total").Group("name").Having("\"name\" like ?", "having_user_%").Rows()
	if err2 != nil {
		t.Fatalf("got error in group by name: %v", err2)
	}
	if rows2 != nil {
		defer rows2.Close()
	}

	var groupCounts2 = map[string]int64{}
	for rows2.Next() {
		var name string
		var total int64
		rows2.Scan(&name, &total)
		groupCounts2[name] = total
	}
	expected := map[string]int64{
		"having_user_1": 3,
		"having_user_2": 1,
	}
	for k, v := range expected {
		if groupCounts2[k] != v {
			t.Errorf("Expected total=%d for name=%s, got %d", v, k, groupCounts2[k])
		}
	}

	rows3, err3 := DB.Table("\"users\" u").Select("u.\"name\", count(*) as total").Having("u.\"name\" = ? or u.\"name\" = ?", "having_user_1", "having_user_2").Group("u.\"name\"").Rows()
	if err3 != nil {
		t.Fatalf("got error in group by name: %v", err3)
	}
	if rows3 != nil {
		defer rows3.Close()
	}

	var groupCounts3 = map[string]int64{}
	for rows3.Next() {
		var name string
		var total int64
		rows3.Scan(&name, &total)
		groupCounts3[name] = total
	}
	for k, v := range expected {
		if groupCounts3[k] != v {
			t.Errorf("Expected total=%d for name=%s, got %d", v, k, groupCounts3[k])
		}
	}

	rows4, err4 := DB.Table("\"users\" \"u\"").Select("\"u\".\"age\", count(*) as total").Where("\"name\" like ?", "having_user_%").Group("\"u\".\"age\"").Having("\"u\".\"age\" >= 1").Rows()
	if err4 != nil {
		t.Fatalf("got error in group by age: %v", err4)
	}
	if rows4 != nil {
		defer rows4.Close()
	}

	var groupCounts4 = map[int]int64{}
	for rows4.Next() {
		var age int
		var total int64
		rows4.Scan(&age, &total)
		groupCounts4[age] = total
	}
	expected2 := map[int]int64{
		1:  1,
		10: 1,
		20: 1,
		30: 1,
	}
	for k, v := range expected2 {
		if groupCounts4[k] != v {
			t.Errorf("Expected total=%d for age=%d, got %d", v, k, groupCounts4[k])
		}
	}

	rows5, err5 := DB.Select("\"name\", count(*) as \"total\"").Table("users").Group("name").Having("\"name\" like ?", "having_user_%").Having("count(*) > ?", 1).Rows()
	if err5 != nil {
		t.Fatalf("got error in group by name with having count > 1: %v", err5)
	}
	if rows5 != nil {
		defer rows5.Close()
	}

	for rows5.Next() {
		var name string
		var total int64
		rows5.Scan(&name, &total)
		if name != "having_user_1" {
			t.Errorf("Should only return having_user_1, got %v", name)
		}
		if total != 3 {
			t.Errorf("Expected 3 users for name having_user_1, got %d", total)
		}
	}
}

func TestQueryRaw(t *testing.T) {
	users := []*User{
		GetUser("row_query_user", Config{}),
		GetUser("row_query_user", Config{}),
		GetUser("row_query_user", Config{}),
	}
	DB.Create(&users)

	var user User
	DB.Raw("select * from \"users\" WHERE \"id\" = ?", users[1].ID).First(&user)
	CheckUser(t, user, *users[1])

	DB.Raw("select * from \"users\" u WHERE u.\"id\" = ?", users[2].ID).First(&user)
	CheckUser(t, user, *users[2])

	DB.Raw("select * from \"users\" \"u\" WHERE \"u\".\"id\" = ?", users[0].ID).First(&user)
	CheckUser(t, user, *users[0])
}

func TestDryRun(t *testing.T) {
	user := *GetUser("dry-run", Config{})

	dryRunDB := DB.Session(&gorm.Session{DryRun: true})

	stmt := dryRunDB.Create(&user).Statement
	if stmt.SQL.String() == "" || len(stmt.Vars) != 9 {
		t.Errorf("Failed to generate sql, got %v", stmt.SQL.String())
	}

	stmt2 := dryRunDB.Find(&user, "id = ?", user.ID).Statement
	if stmt2.SQL.String() == "" || len(stmt2.Vars) != 1 {
		t.Errorf("Failed to generate sql, got %v", stmt2.SQL.String())
	}
}

type ageInt int8

func (ageInt) String() string {
	return "age"
}

type ageBool bool

func (ageBool) String() string {
	return "age"
}

type ageUint64 uint64

func (ageUint64) String() string {
	return "age"
}

type ageFloat float64

func (ageFloat) String() string {
	return "age"
}

func TestExplainSQL(t *testing.T) {
	user := *GetUser("explain-sql", Config{})
	dryRunDB := DB.Session(&gorm.Session{DryRun: true})

	stmt := dryRunDB.Model(&user).Where("id = ?", 1).Updates(map[string]interface{}{"age": ageInt(8)}).Statement
	sql := DB.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)
	if !regexp.MustCompile(`.*age.*=8,`).MatchString(sql) {
		t.Errorf("Failed to generate sql, got %v", sql)
	}

	stmt = dryRunDB.Model(&user).Where("id = ?", 1).Updates(map[string]interface{}{"age": ageUint64(10241024)}).Statement
	sql = DB.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)
	if !regexp.MustCompile(`.*age.*=10241024,`).MatchString(sql) {
		t.Errorf("Failed to generate sql, got %v", sql)
	}

	stmt = dryRunDB.Model(&user).Where("id = ?", 1).Updates(map[string]interface{}{"age": ageBool(false)}).Statement
	sql = DB.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)
	if !regexp.MustCompile(`.*age.*=false,`).MatchString(sql) {
		t.Errorf("Failed to generate sql, got %v", sql)
	}

	stmt = dryRunDB.Model(&user).Where("id = ?", 1).Updates(map[string]interface{}{"age": ageFloat(0.12345678)}).Statement
	sql = DB.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)
	if !regexp.MustCompile(`.*age.*=0.123457,`).MatchString(sql) {
		t.Errorf("Failed to generate sql, got %v", sql)
	}
}

func TestGroupConditions(t *testing.T) {
	type Pizza struct {
		ID   uint
		Name string
		Size string
	}
	dryRunDB := DB.Session(&gorm.Session{DryRun: true})

	stmt := dryRunDB.Where(
		DB.Where("\"pizza\" = ?", "pepperoni").Where(DB.Where("\"size\" = ?", "small").Or("\"size\" = ?", "medium")),
	).Or(
		DB.Where("\"pizza\" = ?", "hawaiian").Where("\"size\" = ?", "xlarge"),
	).Find(&Pizza{}).Statement

	execStmt := dryRunDB.Exec("WHERE (\"pizza\" = ? AND (\"size\" = ? OR \"size\" = ?)) OR (\"pizza\" = ? AND \"size\" = ?)", "pepperoni", "small", "medium", "hawaiian", "xlarge").Statement

	result := DB.Dialector.Explain(stmt.SQL.String(), stmt.Vars...)
	expects := DB.Dialector.Explain(execStmt.SQL.String(), execStmt.Vars...)

	if !strings.HasSuffix(result, expects) {
		t.Errorf("expects: %v, got %v", expects, result)
	}

	stmt2 := dryRunDB.Where(
		DB.Scopes(NameIn1And2),
	).Or(
		DB.Where("\"pizza\" = ?", "hawaiian").Where("\"size\" = ?", "xlarge"),
	).Find(&Pizza{}).Statement

	execStmt2 := dryRunDB.Exec(`WHERE "name" in ? OR ("pizza" = ? AND "size" = ?)`, []string{"ScopeUser1", "ScopeUser2"}, "hawaiian", "xlarge").Statement

	result2 := DB.Dialector.Explain(stmt2.SQL.String(), stmt2.Vars...)
	expects2 := DB.Dialector.Explain(execStmt2.SQL.String(), execStmt2.Vars...)

	if !strings.HasSuffix(result2, expects2) {
		t.Errorf("expects: %v, got %v", expects2, result2)
	}

	stmt3 := dryRunDB.Model(&Pizza{}).Select("\"name\", count(*) as \"total\"").Group("name").Having("count(*) > ?", 1).Find(&[]Pizza{}).Statement

	execStmt3 := dryRunDB.Exec("SELECT \"name\", count(*) as \"total\" FROM \"pizzas\" GROUP BY \"name\" HAVING count(*) > ?", 1).Statement

	result3 := DB.Dialector.Explain(stmt3.SQL.String(), stmt3.Vars...)
	expects3 := DB.Dialector.Explain(execStmt3.SQL.String(), execStmt3.Vars...)

	if !strings.HasSuffix(result3, expects3) {
		t.Errorf("expects: %v, got %v", expects3, result3)
	}

	stmt4 := dryRunDB.Model(&Pizza{}).Table("\"pizzas\" as p").Select("p.\"name\", count(*) as \"total\"").Group("p.\"name\"").Having("count(*) > ?", 1).Find(&[]Pizza{}).Statement

	execStmt4 := dryRunDB.Exec("SELECT p.\"name\", count(*) as \"total\" FROM \"pizzas\" as p GROUP BY p.\"name\" HAVING count(*) > ?", 1).Statement

	result4 := DB.Dialector.Explain(stmt4.SQL.String(), stmt4.Vars...)
	expects4 := DB.Dialector.Explain(execStmt4.SQL.String(), execStmt4.Vars...)

	if !strings.HasSuffix(result4, expects4) {
		t.Errorf("expects: %v, got %v", expects4, result4)
	}

	stmt5 := dryRunDB.Model(&Pizza{}).Where("size = ?", "large").Or("name = ?", "Margherita").Not("size = ?", "small").Find(&[]Pizza{}).Statement
	execStmt5 := dryRunDB.Exec("SELECT * FROM \"pizzas\" WHERE size = 'large' OR name = 'Margherita' AND NOT size = 'small'", 1).Statement

	result5 := DB.Dialector.Explain(stmt5.SQL.String(), stmt5.Vars...)
	expects5 := DB.Dialector.Explain(execStmt5.SQL.String(), execStmt5.Vars...)

	if !strings.HasSuffix(result5, expects5) {
		t.Errorf("expects: %v, got %v", expects5, result5)
	}
}

func TestCombineStringConditions(t *testing.T) {
	dryRunDB := DB.Session(&gorm.Session{DryRun: true})
	sql := dryRunDB.Where("a = ? or b = ?", "a", "b").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(a = .+ or b = .+\) AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("a = ? or b = ?", "a", "b").Or("c = ? and d = ?", "c", "d").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(\(a = .+ or b = .+\) OR \(c = .+ and d = .+\)\) AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("a = ? or b = ?", "a", "b").Or("c = ?", "c").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(\(a = .+ or b = .+\) OR c = .+\) AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("a = ? or b = ?", "a", "b").Or("c = ? and d = ?", "c", "d").Or("e = ? and f = ?", "e", "f").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(\(a = .+ or b = .+\) OR \(c = .+ and d = .+\) OR \(e = .+ and f = .+\)\) AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("a = ? or b = ?", "a", "b").Where("c = ? and d = ?", "c", "d").Not("e = ? and f = ?", "e", "f").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(a = .+ or b = .+\) AND \(c = .+ and d = .+\) AND NOT \(e = .+ and f = .+\) AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("a = ? or b = ?", "a", "b").Where("c = ?", "c").Not("e = ? and f = ?", "e", "f").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(a = .+ or b = .+\) AND c = .+ AND NOT \(e = .+ and f = .+\) AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("a = ? or b = ?", "a", "b").Where("c = ? and d = ?", "c", "d").Not("e = ?", "e").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(a = .+ or b = .+\) AND \(c = .+ and d = .+\) AND NOT e = .+ AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("a = ? or b = ?", "a", "b").Unscoped().Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE a = .+ or b = .+$`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Or("a = ? or b = ?", "a", "b").Unscoped().Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE a = .+ or b = .+$`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	sql = dryRunDB.Not("a = ? or b = ?", "a", "b").Unscoped().Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE NOT \(a = .+ or b = .+\)$`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	subquery := dryRunDB.Model(&User{}).Select("id").Where("active = ?", true)
	sql = dryRunDB.Where("id IN (?)", subquery).Or("name = ?", "special").Not("deleted_at IS NOT NULL").Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(id IN \(SELECT \"id\" FROM \"users\" WHERE active = .+ AND "users"\."deleted_at" IS NULL\) OR name = .+ AND NOT deleted_at IS NOT NULL\) AND "users"\."deleted_at" IS NULL`).MatchString(sql) {
		t.Fatalf("invalid complex (subquery) sql generated, got %v", sql)
	}

	sql = dryRunDB.Where("age > ?", 18).Or("name = ?", "foo").Group("type").Having("COUNT(*) > ?", 2).Find(&User{}).Statement.SQL.String()
	if !regexp.MustCompile(`WHERE \(age > .+ OR name = .+\) AND \"users\"\.\"deleted_at\" IS NULL GROUP BY \"type\" HAVING COUNT\(\*\) > .+`).MatchString(sql) {
		t.Fatalf("invalid sql for AND/OR with GROUP BY/HAVING, got %v", sql)
	}
}

func TestFromWithJoins(t *testing.T) {
	var result User

	newDB := DB.Session(&gorm.Session{NewDB: true, DryRun: true}).Table("users")

	newDB.Clauses(
		clause.From{
			Tables: []clause.Table{{Name: "users"}},
			Joins: []clause.Join{
				{
					Table: clause.Table{Name: "companies", Raw: false},
					ON: clause.Where{
						Exprs: []clause.Expression{
							clause.Eq{
								Column: clause.Column{
									Table: "users",
									Name:  "company_id",
								},
								Value: clause.Column{
									Table: "companies",
									Name:  "id",
								},
							},
						},
					},
				},
			},
		},
	)

	newDB.Joins("inner join rgs on rgs.id = user.id")

	stmt := newDB.First(&result).Statement
	str := stmt.SQL.String()

	if !strings.Contains(str, "rgs.id = user.id") {
		t.Errorf("The second join condition is over written instead of combining")
	}

	if !strings.Contains(str, "`users`.`company_id` = `companies`.`id`") && !strings.Contains(str, "\"users\".\"company_id\" = \"companies\".\"id\"") {
		t.Errorf("The first join condition is over written instead of combining")
	}

	newDB2 := DB.Session(&gorm.Session{NewDB: true, DryRun: true}).Table("users")
	newDB2.Clauses(
		clause.From{
			Tables: []clause.Table{{Name: "users", Alias: "u"}},
			Joins: []clause.Join{
				{
					Type:  "LEFT",
					Table: clause.Table{Name: "profiles", Alias: "p"},
					ON: clause.Where{
						Exprs: []clause.Expression{
							clause.Eq{
								Column: clause.Column{Table: "u", Name: "profile_id"},
								Value:  clause.Column{Table: "p", Name: "id"},
							},
						},
					},
				},
				{
					Type:  "INNER",
					Table: clause.Table{Name: "departments", Alias: "d"},
					ON: clause.Where{
						Exprs: []clause.Expression{
							clause.Eq{
								Column: clause.Column{Table: "d", Name: "id"},
								Value:  clause.Column{Table: "u", Name: "department_id"},
							},
						},
					},
				},
				{
					Type:  "RIGHT",
					Table: clause.Table{Name: "teams", Alias: "t"},
					ON: clause.Where{
						Exprs: []clause.Expression{
							clause.Eq{
								Column: clause.Column{Table: "t", Name: "leader_id"},
								Value:  clause.Column{Table: "u", Name: "id"},
							},
						},
					},
				},
				{
					Type:  "CROSS",
					Table: clause.Table{Name: "projects", Alias: "prj"},
				},
			},
		},
	)
	stmt2 := newDB2.Select("u.name, p.email, d.name, t.name").Where("u.active = ?", true).First(&User{}).Statement
	extStr := stmt2.SQL.String()

	if !strings.Contains(extStr, "LEFT JOIN \"profiles\" \"p\" ON \"u\".\"profile_id\" = \"p\".\"id\"") ||
		!strings.Contains(extStr, "INNER JOIN \"departments\" \"d\" ON \"d\".\"id\" = \"u\".\"department_id\"") ||
		!strings.Contains(extStr, "RIGHT JOIN \"teams\" \"t\" ON \"t\".\"leader_id\" = \"u\".\"id\"") ||
		!strings.Contains(extStr, "CROSS JOIN \"projects\" \"prj\"") {
		t.Errorf("SQL does not contain all expected join types or conditions:\n%s", extStr)
	}

	if !strings.Contains(extStr, "u.name") || !strings.Contains(extStr, "p.email") || !strings.Contains(extStr, "d.name") || !strings.Contains(extStr, "t.name") {
		t.Errorf("SQL does not use aliases in SELECT as expected:\n%s", extStr)
	}

	if !strings.Contains(extStr, "WHERE u.active") {
		t.Errorf("SQL does not use alias in WHERE as expected:\n%s", extStr)
	}
}

func TestToSQL(t *testing.T) {
	// By default DB.DryRun should false
	if DB.DryRun {
		t.Fatal("Failed expect DB.DryRun to be false")
	}

	date, _ := time.ParseInLocation("2006-01-02", "2021-10-18", time.Local)

	// find
	sql := DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("id = ?", 100).Limit(10).Order("age desc").Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE id = 100 AND "users"."deleted_at" IS NULL ORDER BY age desc FETCH NEXT 10 ROWS ONLY`, sql)

	// Join
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Joins("JOIN \"companies\" ON \"users\".\"company_id\" = \"companies\".\"id\"").Where("\"companies\".\"name\" = ?", "Acme").Select("\"id\"").Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT "id" FROM "users" JOIN "companies" ON "users"."company_id" = "companies"."id" WHERE "companies"."name" = "Acme" AND "users"."deleted_at" IS NULL`, sql)

	// Select specific columns
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Select("id, name").Where("age > ?", 18).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT id, name FROM "users" WHERE age > 18 AND "users"."deleted_at" IS NULL`, sql)

	// Distinct select
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Distinct().Select("name").Where("active = ?", true).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT DISTINCT "name" FROM "users" WHERE active = true AND "users"."deleted_at" IS NULL`, sql)

	// Group By, Having
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Select("company_id, COUNT(*) as cnt").Group("company_id").Having("COUNT(*) > ?", 2).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT company_id, COUNT(*) as cnt FROM "users" WHERE "users"."deleted_at" IS NULL GROUP BY "company_id" HAVING COUNT(*) > 2`, sql)

	// Where IN
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("age IN (?)", []int{20, 25, 30}).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE age IN (20,25,30) AND "users"."deleted_at" IS NULL`, sql)

	// Where BETWEEN
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("age BETWEEN ? AND ?", 18, 30).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE (age BETWEEN 18 AND 30) AND "users"."deleted_at" IS NULL`, sql)

	// Nested subquery
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		subq := DB.Model(&User{}).Select("company_id").Where("active = ?", true)
		return tx.Model(&User{}).Where("company_id IN (?)", subq).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE company_id IN (SELECT "company_id" FROM "users" WHERE active = true AND "users"."deleted_at" IS NULL) AND "users"."deleted_at" IS NULL`, sql)

	// Multiple joins with aliases
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).
			Select("\"users\".\"id\", \"users\".\"name\", \"companies\".\"name\", \"profiles\".\"bio\"").
			Joins(`LEFT JOIN "companies" ON "users"."company_id" = "companies"."id"`).
			Joins(`LEFT JOIN "profiles" ON "users"."id" = "profiles"."user_id"`).
			Where(`"companies"."active" = ?`, true).Find(&[]User{})
	})
	fmt.Printf("sql: %v\n", sql)
	assertEqualSQL(t, `SELECT "users"."id", "users"."name", "companies"."name", "profiles"."bio" FROM "users" LEFT JOIN "companies" ON "users"."company_id" = "companies"."id" LEFT JOIN "profiles" ON "users"."id" = "profiles"."user_id" WHERE "companies"."active" = true AND "users"."deleted_at" IS NULL`, sql)

	// EXISTS clause
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).
			Where("EXISTS (SELECT 1 FROM companies WHERE companies.id = users.company_id AND companies.active = ?)", true).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE (EXISTS (SELECT 1 FROM companies WHERE companies.id = users.company_id AND companies.active = true)) AND "users"."deleted_at" IS NULL`, sql)

	// FOR UPDATE lock query
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("age > ?", 30).Clauses(clause.Locking{Strength: "UPDATE"}).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE age > 30 AND "users"."deleted_at" IS NULL FOR UPDATE`, sql)

	// Aggregates with GROUP BY and HAVING
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Select("company_id, AVG(age) as avg_age").Group("company_id").Having("AVG(age) > ?", 20).Find(&[]User{})
	})
	assertEqualSQL(t, `SELECT company_id, AVG(age) as avg_age FROM "users" WHERE "users"."deleted_at" IS NULL GROUP BY "company_id" HAVING AVG(age) > 20`, sql)

	// after model changed
	if DB.Statement.DryRun || DB.DryRun {
		t.Fatal("Failed expect DB.DryRun and DB.Statement.ToSQL to be false")
	}

	if DB.Statement.SQL.String() != "" {
		t.Fatal("Failed expect DB.Statement.SQL to be empty")
	}

	// first
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where(&User{Name: "foo", Age: 20}).Limit(10).Offset(5).Order("name ASC").First(&User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE ("users"."name" = 'foo' AND "users"."age" = 20) AND "users"."deleted_at" IS NULL ORDER BY name ASC,"users"."id" OFFSET 5 ROWS FETCH NEXT 1 ROW ONLY`, sql)

	// last and unscoped
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Unscoped().Where(&User{Name: "bar", Age: 12}).Limit(10).Offset(5).Order("name ASC").Last(&User{})
	})
	assertEqualSQL(t, `SELECT * FROM "users" WHERE "users"."name" = 'bar' AND "users"."age" = 12 ORDER BY name ASC,"users"."id" DESC OFFSET 5 ROWS FETCH NEXT 1 ROW ONLY`, sql)

	// create
	user := &User{Name: "foo", Age: 20}
	user.CreatedAt = date
	user.UpdatedAt = date
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Create(user)
	})
	assertEqualSQL(t, `INSERT INTO "users" ("created_at","updated_at","deleted_at","name","age","birthday","company_id","manager_id","active") VALUES ('2021-10-18 00:00:00','2021-10-18 00:00:00',NULL,'foo',20,NULL,NULL,NULL,false) RETURNING "id" INTO .*`, sql)

	// save
	user = &User{Name: "foo", Age: 20}
	user.CreatedAt = date
	user.UpdatedAt = date
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Save(user)
	})
	assertEqualSQL(t, `INSERT INTO "users" ("created_at","updated_at","deleted_at","name","age","birthday","company_id","manager_id","active") VALUES ('2021-10-18 00:00:00','2021-10-18 00:00:00',NULL,'foo',20,NULL,NULL,NULL,false) RETURNING "id" INTO .*`, sql)

	// updates
	user = &User{Name: "bar", Age: 22}
	user.CreatedAt = date
	user.UpdatedAt = date
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("id = ?", 100).Updates(user)
	})
	assertEqualSQL(t, `UPDATE "users" SET "created_at"='2021-10-18 00:00:00',"updated_at"='2021-10-18 19:50:09.438',"name"='bar',"age"=22 WHERE id = 100 AND "users"."deleted_at" IS NULL`, sql)

	// update
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("id = ?", 100).Update("name", "Foo bar")
	})
	assertEqualSQL(t, `UPDATE "users" SET "name"='Foo bar',"updated_at"='2021-10-18 19:50:09.438' WHERE id = 100 AND "users"."deleted_at" IS NULL`, sql)

	// UpdateColumn
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("id = ?", 100).UpdateColumn("name", "Foo bar")
	})
	assertEqualSQL(t, `UPDATE "users" SET "name"='Foo bar' WHERE id = 100 AND "users"."deleted_at" IS NULL`, sql)

	// UpdateColumns
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(&User{}).Where("id = ?", 100).UpdateColumns(User{Name: "Foo", Age: 100})
	})
	assertEqualSQL(t, `UPDATE "users" SET "name"='Foo',"age"=100 WHERE id = 100 AND "users"."deleted_at" IS NULL`, sql)

	// after model changed
	if DB.Statement.DryRun || DB.DryRun {
		t.Fatal("Failed expect DB.DryRun and DB.Statement.ToSQL to be false")
	}

	// UpdateColumns
	sql = DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Raw("SELECT * FROM users ?", clause.OrderBy{
			Columns: []clause.OrderByColumn{{Column: clause.Column{Name: "id", Raw: true}, Desc: true}},
		})
	})
	assertEqualSQL(t, `SELECT * FROM users ORDER BY id DESC`, sql)
}

// assertEqualSQL for assert that the sql is equal, this method will ignore quote, and dialect specials.
func assertEqualSQL(t *testing.T, expected string, actually string) {
	t.Helper()

	// replace SQL quote
	expected = replaceQuoteInSQL(expected)
	actually = replaceQuoteInSQL(actually)

	// ignore updated_at value, because it's generated in Gorm internal, can't to mock value on update.
	updatedAtRe := regexp.MustCompile(`(?i)"updated_at"=".+?"`)
	actually = updatedAtRe.ReplaceAllString(actually, `"updated_at"=?`)
	expected = updatedAtRe.ReplaceAllString(expected, `"updated_at"=?`)

	// ignore RETURNING "id"
	returningRe := regexp.MustCompile(`RETURNING "id".*`)
	actually = returningRe.ReplaceAllString(actually, ``)
	expected = returningRe.ReplaceAllString(expected, ``)

	actually = strings.TrimSpace(actually)
	expected = strings.TrimSpace(expected)

	if actually != expected {
		t.Fatalf("\nexpected: %s\nactually: %s", expected, actually)
	}
}

func replaceQuoteInSQL(sql string) string {
	// convert single quote into double quote
	return strings.ReplaceAll(sql, `'`, `"`)
}
