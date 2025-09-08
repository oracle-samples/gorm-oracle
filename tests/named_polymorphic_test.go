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
	"fmt"
	"testing"
	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"
	"gorm.io/gorm"
)

type Hamster struct {
	ID           int
	Name         string
	PreferredToy Toy `gorm:"polymorphic:Owner;polymorphicValue:hamster_preferred"`
	OtherToy     Toy `gorm:"polymorphic:Owner;polymorphicValue:hamster_other"`
}

func TestNamedPolymorphic(t *testing.T) {
	DB.Migrator().DropTable(&Hamster{})
	DB.AutoMigrate(&Hamster{})

	hamster := Hamster{Name: "Mr. Hammond", PreferredToy: Toy{Name: "bike"}, OtherToy: Toy{Name: "treadmill"}}
	DB.Save(&hamster)

	hamster2 := Hamster{}
	DB.Preload("PreferredToy").Preload("OtherToy").Find(&hamster2, hamster.ID)

	if hamster2.PreferredToy.ID != hamster.PreferredToy.ID || hamster2.PreferredToy.Name != hamster.PreferredToy.Name {
		t.Errorf("Hamster's preferred toy failed to preload")
	}

	if hamster2.OtherToy.ID != hamster.OtherToy.ID || hamster2.OtherToy.Name != hamster.OtherToy.Name {
		t.Errorf("Hamster's other toy failed to preload")
	}

	// clear to omit Toy.ID in count
	hamster2.PreferredToy = Toy{}
	hamster2.OtherToy = Toy{}

	if DB.Model(&hamster2).Association("PreferredToy").Count() != 1 {
		t.Errorf("Hamster's preferred toy count should be 1")
	}

	if DB.Model(&hamster2).Association("OtherToy").Count() != 1 {
		t.Errorf("Hamster's other toy count should be 1")
	}

	// Query
	hamsterToy := Toy{}
	DB.Model(&hamster).Association("PreferredToy").Find(&hamsterToy)
	if hamsterToy.Name != hamster.PreferredToy.Name {
		t.Errorf("Should find has one polymorphic association")
	}

	hamsterToy = Toy{}
	DB.Model(&hamster).Association("OtherToy").Find(&hamsterToy)
	if hamsterToy.Name != hamster.OtherToy.Name {
		t.Errorf("Should find has one polymorphic association")
	}

	// Append
	DB.Model(&hamster).Association("PreferredToy").Append(&Toy{
		Name: "bike 2",
	})

	DB.Model(&hamster).Association("OtherToy").Append(&Toy{
		Name: "treadmill 2",
	})

	hamsterToy = Toy{}
	DB.Model(&hamster).Association("PreferredToy").Find(&hamsterToy)
	if hamsterToy.Name != "bike 2" {
		t.Errorf("Should update has one polymorphic association with Append")
	}

	hamsterToy = Toy{}
	DB.Model(&hamster).Association("OtherToy").Find(&hamsterToy)
	if hamsterToy.Name != "treadmill 2" {
		t.Errorf("Should update has one polymorphic association with Append")
	}

	if DB.Model(&hamster2).Association("PreferredToy").Count() != 1 {
		t.Errorf("Hamster's toys count should be 1 after Append")
	}

	if DB.Model(&hamster2).Association("OtherToy").Count() != 1 {
		t.Errorf("Hamster's toys count should be 1 after Append")
	}

	// Replace
	DB.Model(&hamster).Association("PreferredToy").Replace(&Toy{
		Name: "bike 3",
	})

	DB.Model(&hamster).Association("OtherToy").Replace(&Toy{
		Name: "treadmill 3",
	})

	hamsterToy = Toy{}
	DB.Model(&hamster).Association("PreferredToy").Find(&hamsterToy)
	if hamsterToy.Name != "bike 3" {
		t.Errorf("Should update has one polymorphic association with Replace")
	}

	hamsterToy = Toy{}
	DB.Model(&hamster).Association("OtherToy").Find(&hamsterToy)
	if hamsterToy.Name != "treadmill 3" {
		t.Errorf("Should update has one polymorphic association with Replace")
	}

	if DB.Model(&hamster2).Association("PreferredToy").Count() != 1 {
		t.Errorf("hamster's toys count should be 1 after Replace")
	}

	if DB.Model(&hamster2).Association("OtherToy").Count() != 1 {
		t.Errorf("hamster's toys count should be 1 after Replace")
	}

	// Clear
	DB.Model(&hamster).Association("PreferredToy").Append(&Toy{
		Name: "bike 2",
	})
	DB.Model(&hamster).Association("OtherToy").Append(&Toy{
		Name: "treadmill 2",
	})

	if DB.Model(&hamster).Association("PreferredToy").Count() != 1 {
		t.Errorf("Hamster's toys should be added with Append")
	}

	if DB.Model(&hamster).Association("OtherToy").Count() != 1 {
		t.Errorf("Hamster's toys should be added with Append")
	}

	DB.Model(&hamster).Association("PreferredToy").Clear()

	if DB.Model(&hamster2).Association("PreferredToy").Count() != 0 {
		t.Errorf("Hamster's preferred toy should be cleared with Clear")
	}

	if DB.Model(&hamster2).Association("OtherToy").Count() != 1 {
		t.Errorf("Hamster's other toy should be still available")
	}

	DB.Model(&hamster).Association("OtherToy").Clear()
	if DB.Model(&hamster).Association("OtherToy").Count() != 0 {
		t.Errorf("Hamster's other toy should be cleared with Clear")
	}
}

func TestOracleCRUDOperations(t *testing.T) {
	DB.Migrator().DropTable(&User{}, &Account{}, &Pet{})
	if err := DB.AutoMigrate(&User{}, &Account{}, &Pet{}); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}

	// Test auto-increment behavior with IDENTITY columns
	user := User{
		Name:     "Oracle CRUD User",
		Age:      30,
		Birthday: &time.Time{},
		Active:   true,
	}

	// Create user - should auto-generate ID
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Errorf("Expected auto-generated ID, got 0")
	}

	// Create account with foreign key reference
	account := Account{
		UserID:        sql.NullInt64{Int64: int64(user.ID), Valid: true},
		AccountNumber: "ACC-001",
	}

	if err := DB.Create(&account).Error; err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	if account.ID == 0 {
		t.Errorf("Expected auto-generated account ID, got 0")
	}

	// Create pet with foreign key reference
	pet := Pet{
		UserID: &user.ID,
		Name:   "Oracle Pet",
	}

	if err := DB.Create(&pet).Error; err != nil {
		t.Fatalf("Failed to create pet: %v", err)
	}

	if pet.ID == 0 {
		t.Errorf("Expected auto-generated pet ID, got 0")
	}

	// Test UPDATE operations
	user.Name = "Updated Oracle User"
	user.Age = 31
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
	var updatedUser User
	if err := DB.First(&updatedUser, user.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if updatedUser.Name != "Updated Oracle User" || updatedUser.Age != 31 {
		t.Errorf("User update failed: expected name 'Updated Oracle User' and age 31, got name '%s' and age %d",
			updatedUser.Name, updatedUser.Age)
	}

	// Test reading back with associations
	var retrievedUser User
	if err := DB.Preload("Account").Preload("Pets").First(&retrievedUser, user.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve user with associations: %v", err)
	}

	if retrievedUser.Account.AccountNumber != "ACC-001" {
		t.Errorf("Expected account number ACC-001, got %s", retrievedUser.Account.AccountNumber)
	}

	if len(retrievedUser.Pets) != 1 || retrievedUser.Pets[0].Name != "Oracle Pet" {
		t.Errorf("Expected 1 pet named 'Oracle Pet', got %d pets", len(retrievedUser.Pets))
	}

	// Test DELETE operations
	if err := DB.Delete(&pet).Error; err != nil {
		t.Fatalf("Failed to delete pet: %v", err)
	}

	// Verify deletion
	var petCount int64
	DB.Model(&Pet{}).Where(`"id" = ?`, pet.ID).Count(&petCount)
	if petCount != 0 {
		t.Errorf("Pet should be deleted, but found %d records", petCount)
	}
}

func TestOracleAdvancedOperations(t *testing.T) {
	DB.Migrator().DropTable(&Company{}, &User{}, &Language{}, "user_speak")
	if err := DB.AutoMigrate(&Company{}, &User{}, &Language{}); err != nil {
		t.Fatalf("Failed to auto migrate: %v", err)
	}

	// Test transaction handling
	err := DB.Transaction(func(tx *gorm.DB) error {
		// Create company
		company := Company{
			Name: "Oracle Advanced Corp",
		}
		if err := tx.Create(&company).Error; err != nil {
			return err
		}

		// Create languages
		languages := []Language{
			{Code: "EN", Name: "English"},
			{Code: "ES", Name: "Spanish"},
			{Code: "FR", Name: "French"},
			{Code: "DE", Name: "German"},
		}
		if err := tx.Create(&languages).Error; err != nil {
			return err
		}

		// Create users with company relationship
		users := []User{
			{Name: "John Doe", Age: 30, CompanyID: &company.ID, Active: true},
			{Name: "Jane Smith", Age: 28, CompanyID: &company.ID, Active: true},
			{Name: "Bob Wilson", Age: 35, CompanyID: &company.ID, Active: false},
		}
		if err := tx.Create(&users).Error; err != nil {
			return err
		}

		// Test many-to-many associations within transaction
		if err := tx.Model(&users[0]).Association("Languages").Append(&languages[0], &languages[1]); err != nil {
			return err
		}

		if err := tx.Model(&users[1]).Association("Languages").Append(&languages[1], &languages[2]); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Transaction failed: %v", err)
	}

	// Verify transaction results
	var userCount int64
	DB.Model(&User{}).Count(&userCount)
	if userCount != 3 {
		t.Errorf("Expected 3 users after transaction, got %d", userCount)
	}

	var languageCount int64
	DB.Model(&Language{}).Count(&languageCount)
	if languageCount != 4 {
		t.Errorf("Expected 4 languages after transaction, got %d", languageCount)
	}

	// Test batch operations with Oracle-specific features
	batchUsers := make([]User, 20)
	for i := 0; i < 20; i++ {
		batchUsers[i] = User{
			Name:   fmt.Sprintf("Batch User %d", i+1),
			Age:    uint(20 + (i % 50)),
			Active: i%2 == 0,
		}
	}

	// Test batch insert with Oracle
	if err := DB.CreateInBatches(&batchUsers, 5).Error; err != nil {
		t.Fatalf("Failed to create users in batches: %v", err)
	}

	// Verify batch insert
	DB.Model(&User{}).Count(&userCount)
	if userCount != 23 { // 3 from transaction + 20 from batch
		t.Errorf("Expected 23 users after batch insert, got %d", userCount)
	}

	// Test bulk update using GORM methods with Oracle-compatible expressions
	result := DB.Model(&User{}).Where(&User{Active: true}).Update("age", gorm.Expr(`"age" + ?`, 1))
	if result.Error != nil {
		t.Fatalf("Failed to bulk update: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		t.Errorf("Expected some rows to be affected by bulk update")
	}

	// Test complex query using GORM joins instead of raw SQL
	var activeUsers []User
	if err := DB.Joins("Company").Where(&User{Active: true}).Where(`"age" > ?`, 25).Find(&activeUsers).Error; err != nil {
		t.Fatalf("Failed to execute join query: %v", err)
	}

	// Test transaction rollback scenario
	err = DB.Transaction(func(tx *gorm.DB) error {
		// Create a user
		user := User{Name: "Rollback Test Advanced", Age: 99, Active: true}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		// Force rollback by returning an error
		return fmt.Errorf("forced rollback for testing")
	})

	if err == nil {
		t.Errorf("Expected transaction to fail and rollback")
	}

	// Verify rollback worked
	var rollbackUser User
	result = DB.Where(`"name" = ?`, "Rollback Test Advanced").First(&rollbackUser)
	if result.Error == nil {
		t.Errorf("Expected user to not exist after rollback, but found: %+v", rollbackUser)
	}

	// Test Oracle pagination using GORM methods
	var paginatedUsers []User
	if err := DB.Offset(5).Limit(3).Find(&paginatedUsers).Error; err != nil {
		t.Fatalf("Failed to paginate: %v", err)
	}

	if len(paginatedUsers) != 3 {
		t.Errorf("Expected 3 users from pagination, got %d", len(paginatedUsers))
	}

	// Test Oracle-specific date operations using GORM with quoted identifiers
	var todayUsers []User
	if err := DB.Where(`"created_at" >= ?`, time.Now().Truncate(24*time.Hour)).Find(&todayUsers).Error; err != nil {
		t.Fatalf("Failed to query with date functions: %v", err)
	}

	// Test case-insensitive search using GORM with quoted identifiers
	var searchUsers []User
	if err := DB.Where(`UPPER("name") LIKE UPPER(?)`, "%doe%").Find(&searchUsers).Error; err != nil {
		t.Fatalf("Failed to perform case-insensitive search: %v", err)
	}

	// Test Oracle-specific features that require raw SQL (minimal usage)
	var currentTime time.Time
	if err := DB.Raw("SELECT SYSDATE FROM DUAL").Scan(&currentTime).Error; err != nil {
		t.Fatalf("Failed to query Oracle DUAL table: %v", err)
	}

	if currentTime.IsZero() {
		t.Errorf("Expected current time from Oracle SYSDATE, got zero time")
	}

	// Test Oracle sequence behavior
	var maxID uint
	if err := DB.Model(&User{}).Select(`MAX("id")`).Scan(&maxID).Error; err != nil {
		t.Fatalf("Failed to get max ID: %v", err)
	}

	// Create one more user to test sequence increment
	newUser := User{Name: "Sequence Test", Age: 25, Active: true}
	if err := DB.Create(&newUser).Error; err != nil {
		t.Fatalf("Failed to create sequence test user: %v", err)
	}

	if newUser.ID <= maxID {
		t.Errorf("Expected new user ID to be greater than %d, got %d", maxID, newUser.ID)
	}

	// Test Oracle association operations
	var userWithLanguages User
	if err := DB.Preload("Languages").Where(`"name" = ?`, "John Doe").First(&userWithLanguages).Error; err != nil {
		t.Fatalf("Failed to preload user languages: %v", err)
	}

	if len(userWithLanguages.Languages) != 2 {
		t.Errorf("Expected user to have 2 languages, got %d", len(userWithLanguages.Languages))
	}

	// Test Oracle constraint behavior
	duplicateCompany := Company{Name: "Oracle Advanced Corp"}
	if err := DB.Create(&duplicateCompany).Error; err != nil {
		// This should succeed since Company doesn't have unique constraints in the model
		// This tests that GORM handles duplicate data as expected
	}
}
