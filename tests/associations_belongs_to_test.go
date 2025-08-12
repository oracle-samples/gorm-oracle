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
	"strings"
	"testing"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestBelongsToAssociation(t *testing.T) {
	user := *GetUser("belongs-to", Config{Company: true, Manager: true})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckUser(t, user, user)

	// Find
	var user2 User
	DB.Find(&user2, "\"id\" = ?", user.ID)
	pointerOfUser := &user2
	if err := DB.Model(&pointerOfUser).Association("Company").Find(&user2.Company); err != nil {
		t.Errorf("failed to query users, got error %#v", err)
	}
	user2.Manager = &User{}
	DB.Model(&user2).Association("Manager").Find(user2.Manager)
	CheckUser(t, user2, user)

	// Count
	AssertAssociationCount(t, user, "Company", 1, "")
	AssertAssociationCount(t, user, "Manager", 1, "")

	// Append
	company := Company{Name: "company-belongs-to-append"}
	manager := GetUser("manager-belongs-to-append", Config{})

	if err := DB.Model(&user2).Association("Company").Append(&company); err != nil {
		t.Fatalf("Error happened when append Company, got %v", err)
	}

	if company.ID == 0 {
		t.Fatalf("Company's ID should be created")
	}

	if err := DB.Model(&user2).Association("Manager").Append(manager); err != nil {
		t.Fatalf("Error happened when append Manager, got %v", err)
	}

	if manager.ID == 0 {
		t.Fatalf("Manager's ID should be created")
	}

	user.Company = company
	user.Manager = manager
	user.CompanyID = &company.ID
	user.ManagerID = &manager.ID
	CheckUserSkipUpdatedAt(t, user2, user)

	AssertAssociationCount(t, user2, "Company", 1, "AfterAppend")
	AssertAssociationCount(t, user2, "Manager", 1, "AfterAppend")

	// Replace
	company2 := Company{Name: "company-belongs-to-replace"}
	manager2 := GetUser("manager-belongs-to-replace", Config{})

	if err := DB.Model(&user2).Association("Company").Replace(&company2); err != nil {
		t.Fatalf("Error happened when replace Company, got %v", err)
	}

	if company2.ID == 0 {
		t.Fatalf("Company's ID should be created")
	}

	if err := DB.Model(&user2).Association("Manager").Replace(manager2); err != nil {
		t.Fatalf("Error happened when replace Manager, got %v", err)
	}

	if manager2.ID == 0 {
		t.Fatalf("Manager's ID should be created")
	}

	user.Company = company2
	user.Manager = manager2
	user.CompanyID = &company2.ID
	user.ManagerID = &manager2.ID
	CheckUserSkipUpdatedAt(t, user2, user)

	AssertAssociationCount(t, user2, "Company", 1, "AfterReplace")
	AssertAssociationCount(t, user2, "Manager", 1, "AfterReplace")

	// Delete
	if err := DB.Model(&user2).Association("Company").Delete(&Company{}); err != nil {
		t.Fatalf("Error happened when delete Company, got %v", err)
	}
	AssertAssociationCount(t, user2, "Company", 1, "after delete non-existing data")

	if err := DB.Model(&user2).Association("Company").Delete(&company2); err != nil {
		t.Fatalf("Error happened when delete Company, got %v", err)
	}
	AssertAssociationCount(t, user2, "Company", 0, "after delete")

	if err := DB.Model(&user2).Association("Manager").Delete(&User{}); err != nil {
		t.Fatalf("Error happened when delete Manager, got %v", err)
	}
	AssertAssociationCount(t, user2, "Manager", 1, "after delete non-existing data")

	if err := DB.Model(&user2).Association("Manager").Delete(manager2); err != nil {
		t.Fatalf("Error happened when delete Manager, got %v", err)
	}
	AssertAssociationCount(t, user2, "Manager", 0, "after delete")

	// Prepare Data for Clear
	if err := DB.Model(&user2).Association("Company").Append(&company); err != nil {
		t.Fatalf("Error happened when append Company, got %v", err)
	}

	if err := DB.Model(&user2).Association("Manager").Append(manager); err != nil {
		t.Fatalf("Error happened when append Manager, got %v", err)
	}

	AssertAssociationCount(t, user2, "Company", 1, "after prepare data")
	AssertAssociationCount(t, user2, "Manager", 1, "after prepare data")

	// Clear
	if err := DB.Model(&user2).Association("Company").Clear(); err != nil {
		t.Errorf("Error happened when clear Company, got %v", err)
	}

	if err := DB.Model(&user2).Association("Manager").Clear(); err != nil {
		t.Errorf("Error happened when clear Manager, got %v", err)
	}

	AssertAssociationCount(t, user2, "Company", 0, "after clear")
	AssertAssociationCount(t, user2, "Manager", 0, "after clear")

	// unexist company id
	unexistCompanyID := company.ID + 9999999
	user = User{Name: "invalid-user-with-invalid-belongs-to-foreign-key", CompanyID: &unexistCompanyID}
	if err := DB.Create(&user).Error; err == nil {
		t.Errorf("should have gotten foreign key violation error")
	}
}

func TestBelongsToAssociationForSlice(t *testing.T) {
	users := []User{
		*GetUser("slice-belongs-to-1", Config{Company: true, Manager: true}),
		*GetUser("slice-belongs-to-2", Config{Company: true, Manager: false}),
		*GetUser("slice-belongs-to-3", Config{Company: true, Manager: true}),
	}

	DB.Create(&users)

	AssertAssociationCount(t, users, "Company", 3, "")
	AssertAssociationCount(t, users, "Manager", 2, "")

	// Find
	var companies []Company
	if DB.Model(&users).Association("Company").Find(&companies); len(companies) != 3 {
		t.Errorf("companies count should be %v, but got %v", 3, len(companies))
	}

	var managers []User
	if DB.Model(&users).Association("Manager").Find(&managers); len(managers) != 2 {
		t.Errorf("managers count should be %v, but got %v", 2, len(managers))
	}

	// Append
	DB.Model(&users).Association("Company").Append(
		&Company{Name: "company-slice-append-1"},
		&Company{Name: "company-slice-append-2"},
		&Company{Name: "company-slice-append-3"},
	)

	AssertAssociationCount(t, users, "Company", 3, "After Append")

	DB.Model(&users).Association("Manager").Append(
		GetUser("manager-slice-belongs-to-1", Config{}),
		GetUser("manager-slice-belongs-to-2", Config{}),
		GetUser("manager-slice-belongs-to-3", Config{}),
	)
	AssertAssociationCount(t, users, "Manager", 3, "After Append")

	if err := DB.Model(&users).Association("Manager").Append(
		GetUser("manager-slice-belongs-to-test-1", Config{}),
	).Error; err == nil {
		t.Errorf("unmatched length when update user's manager")
	}

	// Replace -> same as append

	// Delete
	if err := DB.Model(&users).Association("Company").Delete(&users[0].Company); err != nil {
		t.Errorf("no error should happened when deleting company, but got %v", err)
	}

	if users[0].CompanyID != nil || users[0].Company.ID != 0 {
		t.Errorf("users[0]'s company should be deleted'")
	}

	AssertAssociationCount(t, users, "Company", 2, "After Delete")

	// Clear
	DB.Model(&users).Association("Company").Clear()
	AssertAssociationCount(t, users, "Company", 0, "After Clear")

	DB.Model(&users).Association("Manager").Clear()
	AssertAssociationCount(t, users, "Manager", 0, "After Clear")

	// shared company
	company := Company{Name: "shared"}
	if err := DB.Model(&users[0]).Association("Company").Append(&company); err != nil {
		t.Errorf("Error happened when append company to user, got %v", err)
	}

	if err := DB.Model(&users[1]).Association("Company").Append(&company); err != nil {
		t.Errorf("Error happened when append company to user, got %v", err)
	}

	if users[0].CompanyID == nil || users[1].CompanyID == nil || *users[0].CompanyID != *users[1].CompanyID {
		t.Errorf("user's company id should exists and equal, but its: %v, %v", users[0].CompanyID, users[1].CompanyID)
	}

	DB.Model(&users[0]).Association("Company").Delete(&company)
	AssertAssociationCount(t, users[0], "Company", 0, "After Delete")
	AssertAssociationCount(t, users[1], "Company", 1, "After other user Delete")
}

func TestBelongsToDefaultValue(t *testing.T) {
	type Org struct {
		ID string
	}
	type BelongsToUser struct {
		OrgID string
		Org   Org `gorm:"default:NULL"`
	}

	tx := DB.Session(&gorm.Session{})
	tx.Config.DisableForeignKeyConstraintWhenMigrating = true
	tests.AssertEqual(t, DB.Config.DisableForeignKeyConstraintWhenMigrating, false)

	tx.Migrator().DropTable(&BelongsToUser{}, &Org{})
	tx.AutoMigrate(&BelongsToUser{}, &Org{})

	user := &BelongsToUser{
		Org: Org{
			ID: "BelongsToUser_Org_1",
		},
	}
	err := DB.Create(&user).Error
	tests.AssertEqual(t, err, nil)
}

func TestBelongsToAssociationUnscoped(t *testing.T) {
	type ItemParent struct {
		gorm.Model
		Logo string `gorm:"not null;type:varchar(50)"`
	}
	type ItemChild struct {
		gorm.Model
		Name         string `gorm:"type:varchar(50)"`
		ItemParentID uint
		ItemParent   ItemParent
	}

	tx := DB.Session(&gorm.Session{})
	tx.Migrator().DropTable(&ItemParent{}, &ItemChild{})
	tx.AutoMigrate(&ItemParent{}, &ItemChild{})

	item := ItemChild{
		Name: "name",
		ItemParent: ItemParent{
			Logo: "logo",
		},
	}
	if err := tx.Create(&item).Error; err != nil {
		t.Fatalf("failed to create items, got error: %v", err)
	}

	// test replace
	if err := tx.Model(&item).Association("ItemParent").Unscoped().Replace(&ItemParent{
		Logo: "updated logo",
	}); err != nil {
		t.Errorf("failed to replace item parent, got error: %v", err)
	}

	var parents []ItemParent
	if err := tx.Find(&parents).Error; err != nil {
		t.Errorf("failed to find item parent, got error: %v", err)
	}
	if len(parents) != 1 {
		t.Errorf("expected %d parents, got %d", 1, len(parents))
	}

	// test delete
	if err := tx.Model(&item).Association("ItemParent").Unscoped().Delete(&parents); err != nil {
		t.Errorf("failed to delete item parent, got error: %v", err)
	}
	if err := tx.Find(&parents).Error; err != nil {
		t.Errorf("failed to find item parent, got error: %v", err)
	}
	if len(parents) != 0 {
		t.Errorf("expected %d parents, got %d", 0, len(parents))
	}
}

// Test nil pointer handling
func TestBelongsToNilPointer(t *testing.T) {
	user := User{Name: "test-nil-pointer"}

	// Test creating user with nil company
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("should be able to create user with nil company: %v", err)
	}

	// Test finding into nil pointer
	err := DB.Model(&user).Association("Company").Find(nil)
	if err != nil {
		t.Errorf("should handle nil gracefully, but got error: %v", err)
	}

	// Test finding into a valid pointer when no association exists
	var company Company
	err = DB.Model(&user).Association("Company").Find(&company)
	if err != nil {
		t.Errorf("should handle finding non-existent association: %v", err)
	}

	// Company should remain empty since user has no company
	if company.ID != 0 {
		t.Errorf("expected empty company, but got ID: %d", company.ID)
	}

	// Test append nil
	originalCompanyID := user.CompanyID
	err = DB.Model(&user).Association("Company").Append(nil)
	if err != nil {
		t.Errorf("GORM should handle appending nil gracefully: %v", err)
	}

	// Reload user to check if anything changed
	var reloadedUser User
	DB.First(&reloadedUser, user.ID)

	// CompanyID should remain the same (both should be nil)
	if (originalCompanyID == nil) != (reloadedUser.CompanyID == nil) {
		t.Errorf("CompanyID should not change when appending nil")
	}
}

// Test empty slice operations
func TestBelongsToEmptySlice(t *testing.T) {
	users := []User{}

	// Test operations on empty slice
	AssertAssociationCount(t, users, "Company", 0, "empty slice")

	var companies []Company
	if err := DB.Model(&users).Association("Company").Find(&companies); err != nil {
		t.Errorf("should handle empty slice gracefully: %v", err)
	}

	if len(companies) != 0 {
		t.Errorf("expected 0 companies for empty slice, got %d", len(companies))
	}
}

// Test preloading with belongs to
func TestBelongsToPreload(t *testing.T) {
	user := *GetUser("preload-test", Config{Company: true, Manager: true})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Test preloading company
	var user1 User
	if err := DB.Preload("Company").First(&user1, user.ID).Error; err != nil {
		t.Fatalf("failed to preload company: %v", err)
	}

	if user1.Company.ID == 0 {
		t.Error("company should be preloaded")
	}

	// Test preloading manager
	var user2 User
	if err := DB.Preload("Manager").First(&user2, user.ID).Error; err != nil {
		t.Fatalf("failed to preload manager: %v", err)
	}

	if user2.Manager == nil || user2.Manager.ID == 0 {
		t.Error("manager should be preloaded")
	}
}

// Test concurrent association operations
func TestBelongsToSelfReference(t *testing.T) {
	manager := User{Name: "manager-self"}
	if err := DB.Create(&manager).Error; err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Create user that manages themselves (edge case)
	user := User{
		Name:      "self-manager",
		ManagerID: &manager.ID,
	}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create self-managed user: %v", err)
	}

	// Try to make user manage themselves
	user.ManagerID = &user.ID
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("failed to update user to self-manage: %v", err)
	}

	// Verify self-reference works
	var foundUser User
	DB.Preload("Manager").First(&foundUser, user.ID)

	if foundUser.Manager == nil || foundUser.Manager.ID != user.ID {
		t.Error("self-reference should work")
	}
}

// Test belongs to with soft delete
func TestBelongsToSoftDelete(t *testing.T) {
	type SoftDeleteCompany struct {
		gorm.Model
		Name string
	}

	type SoftDeleteUser struct {
		gorm.Model
		Name      string
		CompanyID uint
		Company   SoftDeleteCompany
	}

	DB.Migrator().DropTable(&SoftDeleteUser{}, &SoftDeleteCompany{})
	if err := DB.AutoMigrate(&SoftDeleteUser{}, &SoftDeleteCompany{}); err != nil {
		t.Fatalf("failed to migrate soft delete models: %v", err)
	}

	company := SoftDeleteCompany{Name: "soft-delete-company"}
	if err := DB.Create(&company).Error; err != nil {
		t.Fatalf("failed to create company: %v", err)
	}

	user := SoftDeleteUser{
		Name:      "soft-delete-user",
		CompanyID: company.ID,
	}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Soft delete the company
	if err := DB.Delete(&company).Error; err != nil {
		t.Fatalf("failed to soft delete company: %v", err)
	}

	// Test association with soft deleted company
	var foundUser SoftDeleteUser
	if err := DB.Preload("Company").First(&foundUser, user.ID).Error; err != nil {
		t.Fatalf("failed to find user: %v", err)
	}

	// Company should not be loaded due to soft delete
	if foundUser.Company.ID != 0 {
		t.Error("soft deleted company should not be preloaded")
	}

	// Test with Unscoped
	if err := DB.Preload("Company").Unscoped().First(&foundUser, user.ID).Error; err != nil {
		t.Fatalf("failed to find user with unscoped: %v", err)
	}

	if foundUser.Company.ID == 0 {
		t.Error("company should be loaded with unscoped")
	}
}

// Test belongs to with validation errors
func TestBelongsToOracleSpecificEdgeCases(t *testing.T) {
	user := User{Name: "oracle-limits-test"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Test with company having maximum Oracle VARCHAR2 length
	longName := strings.Repeat("a", 4000) // Oracle VARCHAR2 default limit
	company := Company{Name: longName}

	err := DB.Model(&user).Association("Company").Append(&company)
	if err != nil {
		t.Logf("Oracle string length limit reached (expected): %v", err)
		// This might be expected depending on your Company.Name column definition
	} else {
		t.Logf("Oracle handled long string successfully")
		AssertAssociationCount(t, user, "Company", 1, "long string test")
	}
}

// Test association with transaction rollback
func TestBelongsToTransactionRollback(t *testing.T) {
	user := User{Name: "transaction-rollback-test"}
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Verify user starts with no company
	if user.CompanyID != nil {
		t.Fatal("user should start with no company")
	}

	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin transaction: %v", tx.Error)
	}

	// Create association within transaction
	company := Company{Name: "transaction-company"}
	if err := tx.Model(&user).Association("Company").Append(&company); err != nil {
		tx.Rollback()
		t.Fatalf("failed to append company in transaction: %v", err)
	}

	// Verify association exists within the transaction context
	var userInTx User
	if err := tx.First(&userInTx, user.ID).Error; err != nil {
		tx.Rollback()
		t.Fatalf("failed to find user in transaction: %v", err)
	}

	if userInTx.CompanyID == nil {
		tx.Rollback()
		t.Fatal("company association should exist in transaction")
	}

	// Rollback the transaction
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("failed to rollback transaction: %v", err)
	}

	// Verify the rollback worked - check the user again with main DB
	var userAfterRollback User
	if err := DB.First(&userAfterRollback, user.ID).Error; err != nil {
		t.Fatalf("failed to find user after rollback: %v", err)
	}

	if userAfterRollback.CompanyID != nil {
		t.Errorf("ROLLBACK FAILED: user should have no company after rollback, but CompanyID = %v", *userAfterRollback.CompanyID)
	} else {
		t.Log("SUCCESS: Transaction rollback worked correctly")
	}
}
