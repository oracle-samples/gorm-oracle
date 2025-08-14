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
	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"
)

func TestExceptionsWithInvalidSql(t *testing.T) {
	var columns []string

	if DB.Where("sdsd.zaaa = ?", "sd;;;aa").Pluck("aaa", &columns).Error == nil {
		t.Errorf("Should got error with invalid SQL")
	}

	tx := DB.Model(&User{}).Where("name = ?", "sd;;;aa").Pluck("name", &columns)
	fmt.Println(tx.Error)

	if DB.Model(&User{}).Where("sdsd.zaaa = ?", "sd;;;aa").Pluck("aaa", &columns).Error == nil {
		t.Errorf("Should got error with invalid SQL")
	}

	if DB.Where("sdsd.zaaa = ?", "sd;;;aa").Find(&User{}).Error == nil {
		t.Errorf("Should got error with invalid SQL")
	}

	var count1, count2 int64
	DB.Model(&User{}).Count(&count1)
	if count1 <= 0 {
		t.Errorf("Should find some users")
	}

	if DB.Where("name = ?", "jinzhu; delete * from users").First(&User{}).Error == nil {
		t.Errorf("Should got error with invalid SQL")
	}

	DB.Model(&User{}).Count(&count2)
	if count1 != count2 {
		t.Errorf("No user should not be deleted by invalid SQL")
	}
}

func TestSetAndGet(t *testing.T) {
	if value, ok := DB.Set("hello", "world").Get("hello"); !ok {
		t.Errorf("Should be able to get setting after set")
	} else if value.(string) != "world" {
		t.Errorf("Set value should not be changed")
	}

	if _, ok := DB.Get("non_existing"); ok {
		t.Errorf("Get non existing key should return error")
	}
}

func TesUserInsertScenarios(t *testing.T) {
	type UserWithAge struct {
		ID   uint   `gorm:"column:ID;primaryKey"`
		Name string `gorm:"column:NAME"`
		Age  int    `gorm:"column:AGE"`
	}

	if err := DB.AutoMigrate(&UserWithAge{}); err != nil {
		t.Fatalf("Failed to migrate table: %v", err)
	}

	user1 := UserWithAge{Name: "Alice", Age: 30}
	if err := DB.Create(&user1).Error; err != nil {
		t.Errorf("Basic insert failed: %v", err)
	}

	user2 := UserWithAge{Name: "Bob"}
	if err := DB.Create(&user2).Error; err != nil {
		t.Errorf("Insert with NULL failed: %v", err)
	}

	user3 := UserWithAge{Name: "O'Reilly", Age: 45}
	if err := DB.Create(&user3).Error; err != nil {
		t.Errorf("Insert with special characters failed: %v", err)
	}

	type UserWithTime struct {
		ID        uint      `gorm:"column:ID;primaryKey"`
		Name      string    `gorm:"column:NAME"`
		CreatedAt time.Time `gorm:"column:CREATED_AT"`
	}

	if err := DB.AutoMigrate(&UserWithTime{}); err != nil {
		t.Fatalf("Failed to migrate UserWithTime table: %v", err)
	}

	user4 := UserWithTime{Name: "Charlie"}
	if err := DB.Create(&user4).Error; err != nil {
		t.Errorf("Insert with default timestamp failed: %v", err)
	}
}
