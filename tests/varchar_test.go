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
	"time"

	"gorm.io/gorm"
)

type VarcharTestModel struct {
	ID               uint      `gorm:"primaryKey;autoIncrement"`
	SmallText        string    `gorm:"size:50"`
	MediumText       string    `gorm:"size:500"`
	LargeText        string    `gorm:"size:4000"`
	DefaultSizeText  string
	OptionalText     *string   `gorm:"size:100"`
	NotNullText      string    `gorm:"size:200;not null"`
	UniqueText       string    `gorm:"size:100;uniqueIndex"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func setupVarcharTestTables(t *testing.T) {
	t.Log("Setting up VARCHAR test tables")

	DB.Migrator().DropTable(&VarcharTestModel{})

	err := DB.AutoMigrate(&VarcharTestModel{})
	if err != nil {
		t.Fatalf("Failed to migrate VARCHAR test tables: %v", err)
	}

	t.Log("VARCHAR test tables created successfully")
}

func TestVarcharBasicCRUD(t *testing.T) {
	setupVarcharTestTables(t)

	// CREATE
	model := &VarcharTestModel{
		SmallText:       "Small text content",
		MediumText:      "Medium text content with more characters to test 500 byte limit",
		LargeText:       strings.Repeat("Large ", 100), // ~600 characters
		DefaultSizeText: "Default size text",
		NotNullText:     "Required text",
		UniqueText:      "unique_value_001",
	}

	err := DB.Create(model).Error
	if err != nil {
		t.Fatalf("Failed to create VARCHAR record: %v", err)
	}

	if model.ID == 0 {
		t.Error("Expected auto-generated ID")
	}

	// READ
	var retrieved VarcharTestModel
	err = DB.First(&retrieved, model.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve VARCHAR record: %v", err)
	}

	if retrieved.SmallText != model.SmallText {
		t.Errorf("SmallText mismatch. Expected '%s', got '%s'", model.SmallText, retrieved.SmallText)
	}
	if retrieved.MediumText != model.MediumText {
		t.Errorf("MediumText mismatch. Expected '%s', got '%s'", model.MediumText, retrieved.MediumText)
	}
	if retrieved.LargeText != model.LargeText {
		t.Errorf("LargeText mismatch. Expected length %d, got length %d", len(model.LargeText), len(retrieved.LargeText))
	}

	// UPDATE
	newSmallText := "Updated small text"
	err = DB.Model(&retrieved).Update("small_text", newSmallText).Error
	if err != nil {
		t.Fatalf("Failed to update VARCHAR record: %v", err)
	}

	var updated VarcharTestModel
	err = DB.First(&updated, model.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve updated VARCHAR record: %v", err)
	}

	if updated.SmallText != newSmallText {
		t.Errorf("Updated SmallText mismatch. Expected '%s', got '%s'", newSmallText, updated.SmallText)
	}

	// DELETE
	err = DB.Delete(&updated).Error
	if err != nil {
		t.Fatalf("Failed to delete VARCHAR record: %v", err)
	}

	var deleted VarcharTestModel
	err = DB.First(&deleted, model.ID).Error
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected record not found, got: %v", err)
	}
}

func TestVarcharMaximumSize(t *testing.T) {
	setupVarcharTestTables(t)

	// Test at 4000 bytes
	text4000 := strings.Repeat("X", 4000)
	
	model := &VarcharTestModel{
		LargeText:   text4000,
		NotNullText: "Required",
		UniqueText:  "unique_max_001",
	}

	err := DB.Create(model).Error
	if err != nil {
		t.Fatalf("Failed to create record with 4000-byte VARCHAR: %v", err)
	}

	var retrieved VarcharTestModel
	err = DB.First(&retrieved, model.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve record with 4000-byte VARCHAR: %v", err)
	}

	if len(retrieved.LargeText) != 4000 {
		t.Errorf("Expected 4000 bytes, got %d bytes", len(retrieved.LargeText))
	}

	if retrieved.LargeText != text4000 {
		t.Error("4000-byte VARCHAR content mismatch")
	}

	// Test at boundary: 3999, 4000, 4001 bytes
	testCases := []struct {
		name     string
		size     int
		shouldFit bool
	}{
		{"3999 bytes", 3999, true},
		{"4000 bytes", 4000, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			text := strings.Repeat("A", tc.size)
			model := &VarcharTestModel{
				LargeText:   text,
				NotNullText: "Required",
				UniqueText:  "unique_boundary_" + tc.name,
			}

			err := DB.Create(model).Error
			if tc.shouldFit && err != nil {
				t.Errorf("Expected %s to fit, but got error: %v", tc.name, err)
			}

			if tc.shouldFit {
				var check VarcharTestModel
				DB.First(&check, model.ID)
				if len(check.LargeText) != tc.size {
					t.Errorf("Expected %d bytes, got %d bytes", tc.size, len(check.LargeText))
				}
			}
		})
	}
}

func TestVarcharNullAndEmpty(t *testing.T) {
	setupVarcharTestTables(t)

	// Test with NULL (nil pointer) in optional field
	model1 := &VarcharTestModel{
		SmallText:    "Test",
		NotNullText:  "Required",
		UniqueText:   "unique_null_001",
		OptionalText: nil, // NULL
	}

	err := DB.Create(model1).Error
	if err != nil {
		t.Fatalf("Failed to create record with NULL optional field: %v", err)
	}

	var retrieved1 VarcharTestModel
	err = DB.First(&retrieved1, model1.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve record with NULL: %v", err)
	}

	if retrieved1.OptionalText != nil && *retrieved1.OptionalText != "" {
		t.Errorf("Expected NULL or empty, got '%v'", *retrieved1.OptionalText)
	}

	// Test with empty string
	emptyString := ""
	model2 := &VarcharTestModel{
		SmallText:    "",
		MediumText:   "",
		LargeText:    "",
		NotNullText:  "Required",
		UniqueText:   "unique_empty_001",
		OptionalText: &emptyString,
	}

	err = DB.Create(model2).Error
	if err != nil {
		t.Fatalf("Failed to create record with empty strings: %v", err)
	}

	var retrieved2 VarcharTestModel
	err = DB.First(&retrieved2, model2.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve record with empty strings: %v", err)
	}

	if retrieved2.SmallText != "" {
		t.Errorf("Expected empty string, got '%s'", retrieved2.SmallText)
	}
	if retrieved2.OptionalText == nil || *retrieved2.OptionalText != "" {
		t.Error("Expected empty string in OptionalText")
	}

	// Test updating to NULL
	err = DB.Model(&retrieved2).Update("optional_text", nil).Error
	if err != nil {
		t.Fatalf("Failed to update to NULL: %v", err)
	}

	var retrieved3 VarcharTestModel
	err = DB.First(&retrieved3, model2.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve after NULL update: %v", err)
	}

	if retrieved3.OptionalText != nil && *retrieved3.OptionalText != "" {
		t.Error("Expected NULL or empty string after update")
	}
}

func TestVarcharSizeExceeded(t *testing.T) {
	setupVarcharTestTables(t)

	// insert text larger than SmallText limit (50)
	tooLargeForSmall := strings.Repeat("X", 100) // 100 chars, but limit is 50

	model := &VarcharTestModel{
		SmallText:   tooLargeForSmall,
		NotNullText: "Required",
		UniqueText:  "unique_exceed_001",
	}

	err := DB.Create(model).Error
	if err == nil {
		var retrieved VarcharTestModel
		DB.First(&retrieved, model.ID)
		if len(retrieved.SmallText) > 50 {
			t.Errorf("Expected truncation or error, but stored %d characters", len(retrieved.SmallText))
		}
		t.Logf("Data was stored (possibly truncated) with length: %d", len(retrieved.SmallText))
	} else {
		t.Logf("Expected error when exceeding size limit: %v", err)
	}

	// Test with MediumText (500 byte limit)
	tooLargeForMedium := strings.Repeat("Y", 600)
	model2 := &VarcharTestModel{
		MediumText:  tooLargeForMedium,
		NotNullText: "Required",
		UniqueText:  "unique_exceed_002",
	}

	err = DB.Create(model2).Error
	if err == nil {
		var retrieved VarcharTestModel
		DB.First(&retrieved, model2.ID)
		if len(retrieved.MediumText) > 500 {
			t.Errorf("Expected truncation or error, but stored %d characters", len(retrieved.MediumText))
		}
		t.Logf("MediumText was stored (possibly truncated) with length: %d", len(retrieved.MediumText))
	}
}

func TestVarcharSpecialCharacters(t *testing.T) {
	setupVarcharTestTables(t)

	testCases := []struct {
		name string
		text string
	}{
		{"Special chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"Unicode", "Hello 世界 🌍 Привет مرحبا"},
		{"Whitespace", "  Leading and trailing spaces  "},
		{"Tabs and newlines", "Line1\nLine2\tTabbed"},
		{"Mixed", "Test\n!@#\t世界 123"},
		{"Quotes", `Single ' and double " quotes`},
		{"Backslashes", `Path\to\file\test.txt`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := &VarcharTestModel{
				MediumText:  tc.text,
				NotNullText: "Required",
				UniqueText:  "unique_special_" + tc.name,
			}

			err := DB.Create(model).Error
			if err != nil {
				t.Fatalf("Failed to create record with %s: %v", tc.name, err)
			}

			var retrieved VarcharTestModel
			err = DB.First(&retrieved, model.ID).Error
			if err != nil {
				t.Fatalf("Failed to retrieve record with %s: %v", tc.name, err)
			}

			if retrieved.MediumText != tc.text {
				t.Errorf("%s mismatch.\nExpected: '%s'\nGot: '%s'", tc.name, tc.text, retrieved.MediumText)
			}
		})
	}
}

func TestVarcharBulkAndConstraints(t *testing.T) {
	setupVarcharTestTables(t)

	// bulk insert with multiple records
	records := []VarcharTestModel{
		{SmallText: "Bulk 1", NotNullText: "Required 1", UniqueText: "bulk_unique_001"},
		{SmallText: "Bulk 2", NotNullText: "Required 2", UniqueText: "bulk_unique_002"},
		{SmallText: "Bulk 3", NotNullText: "Required 3", UniqueText: "bulk_unique_003"},
		{SmallText: "Bulk 4", NotNullText: "Required 4", UniqueText: "bulk_unique_004"},
		{SmallText: "Bulk 5", NotNullText: "Required 5", UniqueText: "bulk_unique_005"},
	}

	err := DB.Create(&records).Error
	if err != nil {
		t.Fatalf("Failed to bulk create records: %v", err)
	}

	// Verify all records
	var count int64
	DB.Model(&VarcharTestModel{}).Count(&count)
	if count < 5 {
		t.Errorf("Expected at least 5 records, got %d", count)
	}

	// unique constraint violation
	duplicateModel := &VarcharTestModel{
		SmallText:   "Duplicate test",
		NotNullText: "Required",
		UniqueText:  "bulk_unique_001", // Duplicate of first record
	}

	err = DB.Create(duplicateModel).Error
	if err == nil {
		t.Error("Expected error for unique constraint violation, got nil")
	} else {
		t.Logf("Correctly received unique constraint error: %v", err)
	}

	// Test NOT NULL constraint violation
	invalidModel := &VarcharTestModel{
		SmallText:  "Test",
		UniqueText: "unique_notnull_test",
		// NotNullText is missing
	}

	err = DB.Create(invalidModel).Error
	if err == nil {
		t.Error("Expected error for NOT NULL constraint violation, got nil")
	} else {
		t.Logf("Correctly received NOT NULL constraint error: %v", err)
	}

	// Test batch update
	err = DB.Model(&VarcharTestModel{}).
		Where("\"small_text\" LIKE ?", "Bulk%").
		Update("medium_text", "Batch updated").Error
	if err != nil {
		t.Fatalf("Failed to batch update: %v", err)
	}

	// Verify batch update
	var updatedRecords []VarcharTestModel
	DB.Where("\"medium_text\" = ?", "Batch updated").Find(&updatedRecords)
	if len(updatedRecords) != 5 {
		t.Errorf("Expected 5 batch updated records, got %d", len(updatedRecords))
	}
}