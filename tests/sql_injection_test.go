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
	"testing"
)

type TestUser struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"uniqueIndex"`
	Email    string
	Password string
	Role     string
}

var sqlInjectionTestCases = []string{
	"admin'; DROP TABLE \"test_users\"; --",
	"admin' UNION SELECT 1,\"username\",\"password\",1,1 FROM \"test_users\" --",
	"admin' AND 1=1 --",
	"admin' OR '1'='1' --",
	"admin' AND 1=2 --",
	"admin' OR '1'='2' --",
	"admin'; WAITFOR DELAY '00:00:05' --",
	"admin' AND (SELECT COUNT(*) FROM user_tables) > 0 --",
	"admin'; INSERT INTO \"test_users\" (\"username\",\"email\",\"role\") VALUES ('hacker','hacker@evil.com','admin'); --",
	"admin' --",
	"admin\x00",                  // Null byte injection
	"0x61646D696E",               // Hexadecimal representation of "admin"
	"admin%27%20OR%201%3D1%20--", // URL-encoded injection "admin' OR 1=1 --"
}

func TestRawQueryInjection(t *testing.T) {
	if err := DB.AutoMigrate(&TestUser{}); err != nil {
		t.Errorf("failed to create test table: %v", err)
	}

	testUsers := []TestUser{
		{Username: "admin", Email: "admin@example.com", Password: "admin123", Role: "admin"},
		{Username: "user1", Email: "user1@example.com", Password: "user123", Role: "user"},
	}
	for _, user := range testUsers {
		DB.FirstOrCreate(&user, TestUser{Username: user.Username})
	}
	for _, test := range sqlInjectionTestCases {
		var testUser []TestUser
		result := DB.Raw("SELECT * FROM \"test_users\" WHERE \"username\" = ?", test).Scan(&testUser)
		if result.Error == nil && len(testUser) > 0 {
			t.Errorf("Query should fail or returned no results: %v", result.Error)
		}

		if !DB.Migrator().HasTable(&TestUser{}) {
			t.Errorf("Test table 'test_users' does not exist after migration")
		}

		var count int64
		DB.Model(&TestUser{}).Where("\"username\" = ?", "hacker").Count(&count)
		if count > 0 {
			t.Errorf("Unexpected user 'hacker' was found!")
		}

		var rowCount int64
		DB.Model(&TestUser{}).Count(&rowCount)
		if rowCount != 2 {
			t.Errorf("Expected user table to have 2 rows, but found %d", rowCount)
		}
	}
	DB.Migrator().DropTable(&TestUser{})
}

func TestWhereClauseInjection(t *testing.T) {
	if err := DB.AutoMigrate(&TestUser{}); err != nil {
		t.Errorf("failed to create test table: %v", err)
	}

	testUsers := []TestUser{
		{Username: "admin", Email: "admin@example.com", Password: "admin123", Role: "admin"},
		{Username: "user1", Email: "user1@example.com", Password: "user123", Role: "user"},
	}
	for _, user := range testUsers {
		DB.FirstOrCreate(&user, TestUser{Username: user.Username})
	}
	for _, test := range sqlInjectionTestCases {
		var testUser []TestUser
		result := DB.Where("\"username\" = ?", test).Find(&testUsers)
		if result.Error == nil && len(testUser) > 0 {
			t.Errorf("Query should fail or returned no results: %v", result.Error)
		}

		if !DB.Migrator().HasTable(&TestUser{}) {
			t.Errorf("Test table 'test_users' does not exist after migration")
		}

		var count int64
		DB.Model(&TestUser{}).Where("\"username\" = ?", "hacker").Count(&count)
		if count > 0 {
			t.Errorf("Unexpected user 'hacker' was found!")
		}

		var rowCount int64
		DB.Model(&TestUser{}).Count(&rowCount)
		if rowCount != 2 {
			t.Errorf("Expected user table to have 2 rows, but found %d", rowCount)
		}
	}
	DB.Migrator().DropTable(&TestUser{})
}

func TestUpdateInjection(t *testing.T) {
	if err := DB.AutoMigrate(&TestUser{}); err != nil {
		t.Errorf("failed to create test table: %v", err)
	}

	testUser := TestUser{Username: "updatetest", Email: "update@test.com", Role: "user"}
	DB.Create(&testUser)

	for _, test := range sqlInjectionTestCases {
		DB.Model(&testUser).Where("\"id\" = ?", testUser.ID).Update("email", test)
		var user TestUser
		result := DB.Where("username = ?", "updatetest").First(&user)
		if result.Error == nil && user.Email != test {
			t.Errorf("Expected email to be '%s', but got '%s'", test, user.Email)
		}

		if !DB.Migrator().HasTable(&TestUser{}) {
			t.Errorf("Test table 'test_users' does not exist after migration")
		}

		var count int64
		DB.Model(&TestUser{}).Where("\"username\" = ?", "hacker").Count(&count)
		if count > 0 {
			t.Errorf("Unexpected user 'hacker' was found!")
		}

		var rowCount int64
		DB.Model(&TestUser{}).Count(&rowCount)
		if rowCount != 1 {
			t.Errorf("Expected user table to have 1 rows, but found %d", rowCount)
		}
	}
	DB.Migrator().DropTable(&TestUser{})
}

func TestFirstOrCreateInjection(t *testing.T) {
	if err := DB.AutoMigrate(&TestUser{}); err != nil {
		t.Errorf("failed to create test table: %v", err)
	}

	for i, test := range sqlInjectionTestCases {
		user := TestUser{
			Username: test,
			Email:    fmt.Sprintf("test%d@example.com", i),
			Role:     "user",
		}
		DB.FirstOrCreate(&user, TestUser{Username: test})

		var count int64
		DB.Model(&TestUser{}).Where("\"username\" = ?", test).Count(&count)
		if count != 1 {
			t.Errorf("expected 1 record for input '%s', but found %d", test, count)
		}
	}

	DB.Migrator().DropTable(&TestUser{})
}
