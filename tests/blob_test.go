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
	"bytes"
	"crypto/rand"
	"testing"
	"time"

	"gorm.io/gorm"
)

type BlobTestModel struct {
	ID           uint    `gorm:"primaryKey;autoIncrement"`
	Name         string  `gorm:"size:100;not null"`
	Data         []byte  `gorm:"type:blob"`
	OptionalData *[]byte `gorm:"type:blob"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BlobVariantModel struct {
	ID        uint   `gorm:"primaryKey"`
	SmallBlob []byte `gorm:"type:blob"`
	LargeBlob []byte `gorm:"type:blob"`
}

func setupBlobTestTables(t *testing.T) {
	t.Log("Setting up BLOB test tables")

	DB.Migrator().DropTable(&BlobTestModel{}, &BlobVariantModel{})

	err := DB.AutoMigrate(&BlobTestModel{}, &BlobVariantModel{})
	if err != nil {
		t.Fatalf("Failed to migrate BLOB test tables: %v", err)
	}

	t.Log("BLOB test tables created successfully")
}

func TestBlobBasicCRUD(t *testing.T) {
	setupBlobTestTables(t)

	testData := []byte("Hello, Oracle BLOB!")

	model := &BlobTestModel{
		Name: "Basic CRUD Test",
		Data: testData,
	}

	err := DB.Create(model).Error
	if err != nil {
		t.Fatalf("Failed to create BLOB record: %v", err)
	}

	if model.ID == 0 {
		t.Error("Expected auto-generated ID")
	}

	// READ
	var retrieved BlobTestModel
	err = DB.First(&retrieved, model.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve BLOB record: %v", err)
	}

	if !bytes.Equal(retrieved.Data, testData) {
		t.Errorf("BLOB data mismatch. Expected %v, got %v", testData, retrieved.Data)
	}

	// UPDATE
	newData := []byte("Updated BLOB data")
	err = DB.Model(&retrieved).Update("data", newData).Error
	if err != nil {
		t.Fatalf("Failed to update BLOB record: %v", err)
	}

	// Verify update
	var updated BlobTestModel
	err = DB.First(&updated, model.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve updated BLOB record: %v", err)
	}

	if !bytes.Equal(updated.Data, newData) {
		t.Errorf("Updated BLOB data mismatch. Expected %v, got %v", newData, updated.Data)
	}

	// DELETE
	err = DB.Delete(&updated).Error
	if err != nil {
		t.Fatalf("Failed to delete BLOB record: %v", err)
	}

	// Verify deletion
	var deleted BlobTestModel
	err = DB.First(&deleted, model.ID).Error
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected record not found, got: %v", err)
	}
}

func TestBlobNullAndEmpty(t *testing.T) {
	setupBlobTestTables(t)

	// Test with null/nil BLOB
	model1 := &BlobTestModel{
		Name: "Null BLOB Test",
		Data: nil,
	}

	err := DB.Create(model1).Error
	if err != nil {
		t.Fatalf("Failed to create record with nil BLOB: %v", err)
	}

	var retrieved1 BlobTestModel
	err = DB.First(&retrieved1, model1.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve record with nil BLOB: %v", err)
	}

	if retrieved1.Data != nil && len(retrieved1.Data) != 0 {
		t.Errorf("Expected nil or empty BLOB, got %v", retrieved1.Data)
	}

	// Test with empty BLOB
	model2 := &BlobTestModel{
		Name: "Empty BLOB Test",
		Data: []byte{},
	}

	err = DB.Create(model2).Error
	if err != nil {
		t.Fatalf("Failed to create record with empty BLOB: %v", err)
	}

	var retrieved2 BlobTestModel
	err = DB.First(&retrieved2, model2.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve record with empty BLOB: %v", err)
	}

	if retrieved2.Data != nil && len(retrieved2.Data) != 0 {
		t.Errorf("Expected nil or empty BLOB, got %v", retrieved2.Data)
	}

	// Test with optional BLOB field
	optionalData := []byte("optional data")
	model3 := &BlobTestModel{
		Name:         "Optional BLOB Test",
		Data:         []byte("required data"),
		OptionalData: &optionalData,
	}

	err = DB.Create(model3).Error
	if err != nil {
		t.Fatalf("Failed to create record with optional BLOB: %v", err)
	}

	var retrieved3 BlobTestModel
	err = DB.First(&retrieved3, model3.ID).Error
	if err != nil {
		t.Fatalf("Failed to retrieve record with optional BLOB: %v", err)
	}

	if retrieved3.OptionalData == nil {
		t.Error("Expected optional BLOB data, got nil")
	} else if !bytes.Equal(*retrieved3.OptionalData, optionalData) {
		t.Errorf("Optional BLOB data mismatch. Expected %v, got %v", optionalData, *retrieved3.OptionalData)
	}
}

func TestBlobDifferentSizes(t *testing.T) {
	setupBlobTestTables(t)

	testCases := []struct {
		name string
		size int
	}{
		{"Small BLOB (10 bytes)", 10},
		{"Medium BLOB (1KB)", 1024},
		{"Large BLOB (64KB)", 64 * 1024},
		{"Very Large BLOB (1MB)", 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate test data
			testData := make([]byte, tc.size)
			_, err := rand.Read(testData)
			if err != nil {
				t.Fatalf("Failed to generate random test data: %v", err)
			}

			model := &BlobTestModel{
				Name: tc.name,
				Data: testData,
			}

			// Create
			err = DB.Create(model).Error
			if err != nil {
				t.Fatalf("Failed to create %s: %v", tc.name, err)
			}

			// Read back
			var retrieved BlobTestModel
			err = DB.First(&retrieved, model.ID).Error
			if err != nil {
				t.Fatalf("Failed to retrieve %s: %v", tc.name, err)
			}

			// Verify data integrity
			if !bytes.Equal(retrieved.Data, testData) {
				t.Errorf("%s: data integrity check failed", tc.name)
			}

			if len(retrieved.Data) != tc.size {
				t.Errorf("%s: size mismatch. Expected %d, got %d", tc.name, tc.size, len(retrieved.Data))
			}
		})
	}
}

func TestBlobDataPatterns(t *testing.T) {
	setupBlobTestTables(t)

	testCases := []struct {
		name     string
		dataFunc func(size int) []byte
		size     int
	}{
		{
			name: "All zeros",
			dataFunc: func(size int) []byte {
				return make([]byte, size)
			},
			size: 1024,
		},
		{
			name: "All ones",
			dataFunc: func(size int) []byte {
				data := make([]byte, size)
				for i := range data {
					data[i] = 0xFF
				}
				return data
			},
			size: 1024,
		},
		{
			name: "Alternating pattern",
			dataFunc: func(size int) []byte {
				data := make([]byte, size)
				for i := range data {
					if i%2 == 0 {
						data[i] = 0xAA
					} else {
						data[i] = 0x55
					}
				}
				return data
			},
			size: 1024,
		},
		{
			name: "Sequential bytes",
			dataFunc: func(size int) []byte {
				data := make([]byte, size)
				for i := range data {
					data[i] = byte(i % 256)
				}
				return data
			},
			size: 1024,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testData := tc.dataFunc(tc.size)

			model := &BlobTestModel{
				Name: tc.name,
				Data: testData,
			}

			err := DB.Create(model).Error
			if err != nil {
				t.Fatalf("Failed to create record with %s: %v", tc.name, err)
			}

			var retrieved BlobTestModel
			err = DB.First(&retrieved, model.ID).Error
			if err != nil {
				t.Fatalf("Failed to retrieve record with %s: %v", tc.name, err)
			}

			if !bytes.Equal(retrieved.Data, testData) {
				t.Errorf("%s: data pattern verification failed", tc.name)
			}
		})
	}
}

func TestBlobUpdateScenarios(t *testing.T) {
	setupBlobTestTables(t)

	// Create initial record
	initialData := []byte("Initial BLOB data")
	model := &BlobTestModel{
		Name: "Update Test",
		Data: initialData,
	}

	err := DB.Create(model).Error
	if err != nil {
		t.Fatalf("Failed to create initial record: %v", err)
	}

	testCases := []struct {
		name    string
		newData []byte
	}{
		{"Update to larger data", []byte("This is a much longer BLOB data content for testing updates")},
		{"Update to smaller data", []byte("Small")},
		{"Update to nil", nil},
		{"Update to empty", []byte{}},
		{"Update back to normal", []byte("Normal data again")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Update
			err := DB.Model(model).Update("data", tc.newData).Error
			if err != nil {
				t.Fatalf("Failed to update BLOB: %v", err)
			}

			// Verify
			var retrieved BlobTestModel
			err = DB.First(&retrieved, model.ID).Error
			if err != nil {
				t.Fatalf("Failed to retrieve updated record: %v", err)
			}

			if tc.newData == nil {
				if retrieved.Data != nil && len(retrieved.Data) != 0 {
					t.Errorf("Expected nil/empty BLOB after nil update, got %v", retrieved.Data)
				}
			} else if !bytes.Equal(retrieved.Data, tc.newData) {
				t.Errorf("Update verification failed. Expected %v, got %v", tc.newData, retrieved.Data)
			}
		})
	}
}

func TestBlobWithReturning(t *testing.T) {
	setupBlobTestTables(t)

	testData := []byte("BLOB data for RETURNING test")

	// Test single record with RETURNING
	model := &BlobTestModel{
		Name: "RETURNING Test",
		Data: testData,
	}

	err := DB.Clauses().Create(model).Error
	if err != nil {
		t.Fatalf("Failed to create record with RETURNING: %v", err)
	}

	// Verify the returned data
	if !bytes.Equal(model.Data, testData) {
		t.Errorf("RETURNING data mismatch. Expected %v, got %v", testData, model.Data)
	}

	if model.ID == 0 {
		t.Error("Expected auto-generated ID in RETURNING")
	}

	if model.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt timestamp in RETURNING")
	}
}

func TestBlobErrorHandling(t *testing.T) {
	setupBlobTestTables(t)

	err := DB.Table("non_existent_table").Create(&BlobTestModel{
		Name: "Error Test",
		Data: []byte("test data"),
	}).Error

	if err == nil {
		t.Error("Expected error when inserting into non-existent table")
	}

	largeData := make([]byte, 50*1024*1024) // 50MB

	model := &BlobTestModel{
		Name: "Large BLOB Test",
		Data: largeData,
	}

	err = DB.Create(model).Error
	if err != nil {
		t.Logf("Large BLOB insert failed (this may be expected): %v", err)
	} else {
		t.Log("Large BLOB insert succeeded")

		var retrieved BlobTestModel
		err = DB.First(&retrieved, model.ID).Error
		if err != nil {
			t.Errorf("Failed to retrieve large BLOB: %v", err)
		} else if len(retrieved.Data) != len(largeData) {
			t.Errorf("Large BLOB size mismatch. Expected %d, got %d", len(largeData), len(retrieved.Data))
		}
	}
}

// Helper function to find record index by ID
func getRecordIndex(models []BlobTestModel, id uint) int {
	for i, model := range models {
		if model.ID == id {
			return i
		}
	}
	return -1
}
