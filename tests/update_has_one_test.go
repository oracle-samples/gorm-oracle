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

	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	t.Run("ReplaceAssociation", func(t *testing.T) {
		user := *GetUser("replace-has-one", Config{})

		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create user: %v", err)
		}

		acc1 := Account{AccountNumber: "first-account"}
		user.Account = acc1

		if err := DB.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when saving user with first account: %v", err)
		}

		acc2 := Account{AccountNumber: "second-account"}
		user.Account = acc2
		if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
			t.Fatalf("errors happened when replacing association: %v", err)
		}

		var result User
		DB.Preload("Account").First(&result, user.ID)
		if result.Account.AccountNumber != "second-account" {
			t.Fatalf("expected replaced account to have AccountNumber 'second-account', got %v", result.Account.AccountNumber)
		}
	})

	t.Run("ClearHasOneAssociation", func(t *testing.T) {
		user := *GetUser("nullify-has-one", Config{})

		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create user: %v", err)
		}

		user.Account = Account{AccountNumber: "to-be-nullified"}
		if err := DB.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when saving user: %v", err)
		}

		DB.Model(&user).Association("Account").Clear()

		var result User
		DB.Preload("Account").First(&result, user.ID)
		if result.Account.AccountNumber != "" {
			t.Fatalf("expected account to be nullified/empty, got %v", result.Account.AccountNumber)
		}
	})

	t.Run("ClearPolymorphicAssociation", func(t *testing.T) {
		pet := Pet{Name: "clear-poly"}
		pet.Toy = Toy{Name: "polytoy"}
		DB.Create(&pet)

		DB.Model(&pet).Association("Toy").Clear()

		var pet2 Pet
		DB.Preload("Toy").First(&pet2, pet.ID)
		if pet2.Toy.Name != "" {
			t.Fatalf("expected Toy cleared, got %v", pet2.Toy.Name)
		}
	})

	t.Run("UpdateWithoutAssociation", func(t *testing.T) {
		user := *GetUser("no-assoc-update", Config{})
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create user: %v", err)
		}
		newName := user.Name + "-updated"
		if err := DB.Model(&user).Update("name", newName).Error; err != nil {
			t.Fatalf("errors happened when updating only parent: %v", err)
		}
		var result User
		DB.Preload("Account").First(&result, user.ID)
		if result.Name != newName {
			t.Fatalf("user name not updated as expected")
		}
		if result.Account.ID != 0 {
			t.Fatalf("expected no Account associated, got ID %v", result.Account.ID)
		}
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

	t.Run("AssociationWithoutPreload", func(t *testing.T) {
		user := *GetUser("no-preload", Config{})
		user.Account = Account{AccountNumber: "np-account"}
		DB.Create(&user)

		var result User
		DB.First(&result, user.ID) // no preload
		if result.Account.AccountNumber != "" {
			t.Fatalf("expected Account field empty without preload, got %v", result.Account.AccountNumber)
		}

		var acc Account
		DB.First(&acc, "\"user_id\" = ?", user.ID)
		if acc.AccountNumber != "np-account" {
			t.Fatalf("account not found as expected")
		}
	})

	t.Run("SkipFullSaveAssociations", func(t *testing.T) {
		user := *GetUser("skip-fsa", Config{})
		user.Account = Account{AccountNumber: "skipfsa"}
		DB.Create(&user)

		user.Account.AccountNumber = "should-not-update"
		if err := DB.Session(&gorm.Session{FullSaveAssociations: false}).Save(&user).Error; err != nil {
			t.Fatalf("error saving with FSA false: %v", err)
		}

		var acc Account
		DB.First(&acc, "\"user_id\" = ?", user.ID)
		if acc.AccountNumber != "skipfsa" {
			t.Fatalf("account should not have updated, got %v", acc.AccountNumber)
		}
	})

	t.Run("HasOneZeroForeignKey", func(t *testing.T) {
		now := time.Now()
		user := User{Name: "zero-value-clear", Age: 18, Birthday: &now}
		DB.Create(&user)

		account := Account{AccountNumber: "to-clear", UserID: sql.NullInt64{Int64: int64(user.ID), Valid: true}}
		DB.Create(&account)

		account.UserID = sql.NullInt64{Int64: 0, Valid: false}
		DB.Model(&account).Select("UserID").Updates(account)

		var result User
		DB.Preload("Account").First(&result, user.ID)
		if result.Account.AccountNumber != "" {
			t.Fatalf("expected account cleared, got %v", result.Account.AccountNumber)
		}
	})

	t.Run("PolymorphicZeroForeignKey", func(t *testing.T) {
		pet := Pet{Name: "poly-zero"}
		pet.Toy = Toy{Name: "polytoy-zero"}
		DB.Create(&pet)

		pet.Toy.OwnerID = ""
		DB.Model(&pet.Toy).Select("OwnerID").Updates(&pet.Toy)

		var pet2 Pet
		DB.Preload("Toy").First(&pet2, pet.ID)
		if pet2.Toy.Name != "" {
			t.Fatalf("expected polymorphic association cleared, got %v", pet2.Toy.Name)
		}
	})

	t.Run("InvalidForeignKey", func(t *testing.T) {
		acc := Account{AccountNumber: "badfk", UserID: sql.NullInt64{Int64: 99999999, Valid: true}}
		err := DB.Create(&acc).Error
		if err == nil {
			t.Fatalf("expected foreign key constraint error, got nil")
		}
	})

	t.Run("UpdateWithSelectOmit", func(t *testing.T) {
		user := *GetUser("select-omit", Config{})
		user.Account = Account{AccountNumber: "selomit"}
		DB.Create(&user)

		user.Name = "selomit-updated"
		user.Account.AccountNumber = "selomit-updated"
		if err := DB.Select("Name").Omit("Account").Save(&user).Error; err != nil {
			t.Fatalf("error on select/omit: %v", err)
		}

		var acc Account
		DB.First(&acc, "\"user_id\" = ?", user.ID)
		if acc.AccountNumber != "selomit" {
			t.Fatalf("account should not update with Omit(Account), got %v", acc.AccountNumber)
		}
	})

	t.Run("NestedUpdate", func(t *testing.T) {
		user := *GetUser("nested-update", Config{})
		user.Account = Account{AccountNumber: "nested"}
		DB.Create(&user)

		user.Name = "nested-updated"
		user.Account.AccountNumber = "nested-updated"
		if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Updates(&user).Error; err != nil {
			t.Fatalf("nested update failed: %v", err)
		}

		var result User
		DB.Preload("Account").First(&result, user.ID)
		if result.Name != "nested-updated" || result.Account.AccountNumber != "nested-updated" {
			t.Fatalf("nested update didn't apply: %v / %v", result.Name, result.Account.AccountNumber)
		}
	})

	t.Run("EmptyStructNoFullSave", func(t *testing.T) {
		user := *GetUser("empty-nofsa", Config{})
		user.Account = Account{AccountNumber: "keep"}
		DB.Create(&user)

		user.Account = Account{}
		if err := DB.Save(&user).Error; err != nil {
			t.Fatalf("save failed: %v", err)
		}

		var result User
		DB.Preload("Account").First(&result, user.ID)
		if result.Account.AccountNumber != "keep" {
			t.Fatalf("account should not be cleared without FullSaveAssociations")
		}
	})

	t.Run("DeleteParentCascade", func(t *testing.T) {
		type AccountCascadeDelete struct {
			gorm.Model
			AccountNumber string
			UserID        uint
		}

		type UserCascadeDelete struct {
			gorm.Model
			Name    string
			Account AccountCascadeDelete `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
		}

		DB.Migrator().DropTable(&AccountCascadeDelete{}, &UserCascadeDelete{})
		if err := DB.AutoMigrate(&UserCascadeDelete{}, &AccountCascadeDelete{}); err != nil {
			t.Fatalf("failed to migrate: %v", err)
		}

		user := UserCascadeDelete{
			Name: "delete-parent",
			Account: AccountCascadeDelete{
				AccountNumber: "cascade",
			},
		}

		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		if err := DB.Unscoped().Delete(&user).Error; err != nil {
			t.Fatalf("delete parent failed: %v", err)
		}

		var acc AccountCascadeDelete
		err := DB.First(&acc, "\"user_id\" = ?", user.ID).Error
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("expected account deleted, got %v", acc)
		}
	})

	t.Run("OmitAllAssociations", func(t *testing.T) {
		user := *GetUser("omit-assoc", Config{})
		user.Account = Account{AccountNumber: "original-child"}
		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("failed to create user: %v", err)
		}

		newName := "parent-updated"
		user.Name = newName
		user.Account.AccountNumber = "child-updated"

		if err := DB.Model(&user).Omit(clause.Associations).Updates(user).Error; err != nil {
			t.Fatalf("update with omit associations failed: %v", err)
		}

		var result User
		DB.Preload("Account").First(&result, user.ID)

		if result.Name != newName {
			t.Fatalf("expected parent name updated to %v, got %v", newName, result.Name)
		}

		if result.Account.AccountNumber != "original-child" {
			t.Fatalf("expected child to remain unchanged, got %v", result.Account.AccountNumber)
		}
	})

}
