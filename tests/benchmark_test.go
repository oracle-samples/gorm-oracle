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

	"gorm.io/gorm"
	. "github.com/oracle-samples/gorm-oracle/tests/utils"
)

func BenchmarkCreate(b *testing.B) {
	user := *GetUser("bench", Config{})

	for x := 0; x < b.N; x++ {
		user.ID = 0
		DB.Create(&user)
	}
}

func BenchmarkFind(b *testing.B) {
	user := *GetUser("find", Config{})
	DB.Create(&user)

	for x := 0; x < b.N; x++ {
		DB.Find(&User{}, "id = ?", user.ID)
	}
}

func BenchmarkScan(b *testing.B) {
	user := *GetUser("scan", Config{})
	DB.Create(&user)

	var u User
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Raw("select * from users where id = ?", user.ID).Scan(&u)
	}
}

func BenchmarkScanSlice(b *testing.B) {
	DB.Exec("delete from users")
	for i := 0; i < 10_000; i++ {
		user := *GetUser(fmt.Sprintf("scan-%d", i), Config{})
		DB.Create(&user)
	}

	var u []User
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Raw("select * from users").Scan(&u)
	}
}

func BenchmarkScanSlicePointer(b *testing.B) {
	DB.Exec("delete from users")
	for i := 0; i < 10_000; i++ {
		user := *GetUser(fmt.Sprintf("scan-%d", i), Config{})
		DB.Create(&user)
	}

	var u []*User
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Raw("select * from users").Scan(&u)
	}
}

func BenchmarkUpdate(b *testing.B) {
	user := *GetUser("find", Config{})
	DB.Create(&user)

	for x := 0; x < b.N; x++ {
		DB.Model(&user).Updates(map[string]interface{}{"Age": x})
	}
}

func BenchmarkDelete(b *testing.B) {
	user := *GetUser("find", Config{})

	for x := 0; x < b.N; x++ {
		user.ID = 0
		DB.Create(&user)
		DB.Delete(&user)
	}
}

func BenchmarkCreateInBatches(b *testing.B) {
	users := make([]User, 100)
	for i := 0; i < len(users); i++ {
		users[i] = *GetUser(fmt.Sprintf("batch-%d", i), Config{})
	}

	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.CreateInBatches(users, 20)
	}
}

func BenchmarkFirst(b *testing.B) {
	user := *GetUser("first", Config{})
	DB.Create(&user)

	var u User
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.First(&u, user.ID)
	}
}

func BenchmarkWhere(b *testing.B) {
	user := *GetUser("where", Config{})
	DB.Create(&user)

	var u User
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Where("name = ?", user.Name).First(&u)
	}
}

func BenchmarkCount(b *testing.B) {
	for i := 0; i < 1000; i++ {
		user := *GetUser(fmt.Sprintf("count-%d", i), Config{})
		DB.Create(&user)
	}

	var count int64
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Model(&User{}).Count(&count)
	}
}

func BenchmarkTransaction(b *testing.B) {
	for x := 0; x < b.N; x++ {
		DB.Transaction(func(tx *gorm.DB) error {
			user := *GetUser(fmt.Sprintf("tx-%d", x), Config{})
			return tx.Create(&user).Error
		})
	}
}

func BenchmarkJoin(b *testing.B) {
	user := *GetUser("join-user", Config{})
	DB.Create(&user)

	type Profile struct {
		ID     uint
		UserID uint
		Bio    string
	}
	DB.AutoMigrate(&Profile{})
	DB.Create(&Profile{UserID: user.ID, Bio: "benchmark profile"})

	var result struct {
		User
		Profile
	}

	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Table("users").
			Select("users.id, users.name, profiles.bio").
			Joins("left join profiles on profiles.user_id = users.id").
			Where("users.id = ?", user.ID).
			Scan(&result)
	}
}

func BenchmarkPagination(b *testing.B) {
	DB.Exec("delete from users")
	for i := 0; i < 10000; i++ {
		user := *GetUser(fmt.Sprintf("page-%d", i), Config{})
		DB.Create(&user)
	}

	var users []User
	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Limit(50).Offset(200).Find(&users)
	}
}

func BenchmarkBulkDelete(b *testing.B) {
	DB.Exec("delete from users")
	for i := 0; i < 5000; i++ {
		user := *GetUser(fmt.Sprintf("bulk-%d", i), Config{})
		DB.Create(&user)
	}

	b.ResetTimer()
	for x := 0; x < b.N; x++ {
		DB.Where("id < ?", 2500).Delete(&User{})
	}
}
