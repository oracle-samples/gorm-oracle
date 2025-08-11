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
	"testing"

	"time"

	. "github.com/oracle/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestUpdateHasOne(t *testing.T) {
	user := *GetUser("update-has-one", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Account = Account{AccountNumber: "account-has-one-association"}

	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Account").Find(&user2, "\"id\" = ?", user.ID)
	CheckUserSkipUpdatedAt(t, user2, user)

	user.Account.AccountNumber += "new"
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user3 User
	DB.Preload("Account").Find(&user3, "\"id\" = ?", user.ID)

	CheckUserSkipUpdatedAt(t, user2, user3)
	lastUpdatedAt := user2.Account.UpdatedAt
	time.Sleep(time.Second)

	if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user4 User
	DB.Preload("Account").Find(&user4, "\"id\" = ?", user.ID)

	if lastUpdatedAt.Format(time.RFC3339) == user4.Account.UpdatedAt.Format(time.RFC3339) {
		t.Fatalf("updated at should be updated, but not, old: %v, new %v", lastUpdatedAt.Format(time.RFC3339), user3.Account.UpdatedAt.Format(time.RFC3339))
	} else {
		user.Account.UpdatedAt = user4.Account.UpdatedAt
		CheckUserSkipUpdatedAt(t, user4, user)
	}

	t.Run("Polymorphic", func(t *testing.T) {
		pet := Pet{Name: "create"}

		if err := DB.Create(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		pet.Toy = Toy{Name: "Update-HasOneAssociation-Polymorphic"}

		if err := DB.Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var pet2 Pet
		DB.Preload("Toy").Find(&pet2, "\"id\" = ?", pet.ID)
		CheckPetSkipUpdatedAt(t, pet2, pet)

		pet.Toy.Name += "new"
		if err := DB.Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var pet3 Pet
		DB.Preload("Toy").Find(&pet3, "\"id\" = ?", pet.ID)
		CheckPetSkipUpdatedAt(t, pet2, pet3)

		if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var pet4 Pet
		DB.Preload("Toy").Find(&pet4, "\"id\" = ?", pet.ID)
		CheckPetSkipUpdatedAt(t, pet4, pet)
	})

	t.Run("Restriction", func(t *testing.T) {
		type CustomizeAccount struct {
			gorm.Model
			UserID  sql.NullInt64
			Number  string `gorm:"<-:create"`
			Number2 string
		}

		type CustomizeUser struct {
			gorm.Model
			Name    string
			Account CustomizeAccount `gorm:"foreignkey:UserID"`
		}

		DB.Migrator().DropTable(&CustomizeUser{})
		DB.Migrator().DropTable(&CustomizeAccount{})

		if err := DB.AutoMigrate(&CustomizeUser{}); err != nil {
			t.Fatalf("failed to migrate, got error: %v", err)
		}
		if err := DB.AutoMigrate(&CustomizeAccount{}); err != nil {
			t.Fatalf("failed to migrate, got error: %v", err)
		}

		number := "number-has-one-associations"
		cusUser := CustomizeUser{
			Name: "update-has-one-associations",
			Account: CustomizeAccount{
				Number:  number,
				Number2: number,
			},
		}

		if err := DB.Create(&cusUser).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}
		cusUser.Account.Number += "-update"
		cusUser.Account.Number2 += "-update"
		if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&cusUser).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var account2 CustomizeAccount
		DB.Find(&account2, "\"user_id\" = ?", cusUser.ID)
		tests.AssertEqual(t, account2.Number, number)
		tests.AssertEqual(t, account2.Number2, cusUser.Account.Number2)
	})
}
