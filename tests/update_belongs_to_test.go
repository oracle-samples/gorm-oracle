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
	"testing"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
)

func TestUpdateBelongsTo(t *testing.T) {
	user := *GetUser("update-belongs-to", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Company = Company{Name: "company-belongs-to-association"}
	user.Manager = &User{Name: "manager-belongs-to-association"}
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Company").Preload("Manager").Find(&user2, "\"id\" = ?", user.ID)
	CheckUserSkipUpdatedAt(t, user2, user)

	user.Company.Name += "new"
	user.Manager.Name += "new"
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user3 User
	DB.Preload("Company").Preload("Manager").Find(&user3, "\"id\" = ?", user.ID)
	CheckUserSkipUpdatedAt(t, user2, user3)

	if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user4 User
	DB.Preload("Company").Preload("Manager").Find(&user4, "\"id\" = ?", user.ID)
	CheckUserSkipUpdatedAt(t, user4, user)

	user.Company.Name += "new2"
	user.Manager.Name += "new2"
	if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Select("`Company`").Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user5 User
	DB.Preload("Company").Preload("Manager").Find(&user5, "\"id\" = ?", user.ID)
	if user5.Manager.Name != user4.Manager.Name {
		t.Errorf("should not update user's manager")
	} else {
		user.Manager.Name = user4.Manager.Name
	}
	CheckUserSkipUpdatedAt(t, user, user5)
}

func TestUpdateBelongsToWithReturning(t *testing.T) {
	user := *GetUser("update-belongs-to-returning", Config{})

	// Test that RETURNING clauses work properly when updating with associations
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	originalUpdatedAt := user.UpdatedAt
	user.Company = Company{Name: "returning-test-company"}
	user.Manager = &User{Name: "returning-test-manager"}

	// Save and verify that UpdatedAt was properly returned from db
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update with returning: %v", err)
	}

	// Verify RETURNING clause populated the UpdatedAt field
	if !user.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("expected UpdatedAt to be updated via RETURNING clause")
	}

	// Verify the associations were created properly
	var result User
	if err := DB.Preload("Company").Preload("Manager").First(&result, user.ID).Error; err != nil {
		t.Fatalf("failed to load user with associations: %v", err)
	}

	if result.Company.Name != "returning-test-company" {
		t.Errorf("expected company name to be saved correctly")
	}
	if result.Manager.Name != "returning-test-manager" {
		t.Errorf("expected manager name to be saved correctly")
	}
}

func TestUpdateBelongsToWithNullValues(t *testing.T) {
	user := *GetUser("update-belongs-to-null", Config{})

	// Create user with associations
	user.Company = Company{Name: "initial-company"}
	user.Manager = &User{Name: "initial-manager"}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	// Verify associations were created
	if user.CompanyID == nil {
		t.Fatalf("expected CompanyID to be set after create")
	}
	if user.ManagerID == nil {
		t.Fatalf("expected ManagerID to be set after create")
	}

	// Test setting foreign keys to NULL - clear both the foreign keys AND the association objects
	user.CompanyID = nil     // Clear foreign key
	user.ManagerID = nil     // Clear foreign key
	user.Company = Company{} // Clear association object (zero value)
	user.Manager = nil       // Clear association pointer

	if err := DB.Model(&user).Updates(map[string]interface{}{
		"company_id": nil,
		"manager_id": nil,
	}).Error; err != nil {
		t.Fatalf("errors happened when setting associations to null: %v", err)
	}

	var result User
	if err := DB.First(&result, user.ID).Error; err != nil {
		t.Fatalf("failed to load user: %v", err)
	}

	// Verify foreign keys are properly NULL
	if result.CompanyID != nil {
		t.Errorf("expected CompanyID to be NULL, got %v", *result.CompanyID)
	}
	if result.ManagerID != nil {
		t.Errorf("expected ManagerID to be NULL, got %v", *result.ManagerID)
	}

	// Try to load with preload to ensure NULL handling works
	var resultWithPreload User
	if err := DB.Preload("Company").Preload("Manager").First(&resultWithPreload, user.ID).Error; err != nil {
		t.Fatalf("failed to load user with preload: %v", err)
	}

	// When foreign keys are NULL, preloaded associations should be zero values
	if resultWithPreload.Company.ID != 0 {
		t.Errorf("expected Company to be zero value when foreign key is NULL")
	}
	if resultWithPreload.Manager != nil {
		t.Errorf("expected Manager to be nil when foreign key is NULL")
	}
}
