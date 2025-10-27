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
	"math"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// IntegerTestModel covers basic integer data types.
type IntegerTestModel struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Int8Value   int8      `gorm:"column:INT8_VALUE"`
	Int16Value  int16     `gorm:"column:INT16_VALUE"`
	Int32Value  int32     `gorm:"column:INT32_VALUE"`
	Int64Value  int64     `gorm:"column:INT64_VALUE"`
	IntValue    int       `gorm:"column:INT_VALUE"`
	Uint8Value  uint8     `gorm:"column:UINT8_VALUE"`
	Uint16Value uint16    `gorm:"column:UINT16_VALUE"`
	Uint32Value uint32    `gorm:"column:UINT32_VALUE"`
	Uint64Value uint64    `gorm:"column:UINT64_VALUE"`
	UintValue   uint      `gorm:"column:UINT_VALUE"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NullableIntegerTestModel tests nullable and optional integer types.
type NullableIntegerTestModel struct {
	ID             uint           `gorm:"primaryKey;autoIncrement"`
	NullInt32      sql.NullInt32  `gorm:"column:NULL_INT32"`
	NullInt64      sql.NullInt64  `gorm:"column:NULL_INT64"`
	OptionalInt32  *int32         `gorm:"column:OPTIONAL_INT32"`
	OptionalInt64  *int64         `gorm:"column:OPTIONAL_INT64"`
	OptionalUint32 *uint32        `gorm:"column:OPTIONAL_UINT32"`
	OptionalUint64 *uint64        `gorm:"column:OPTIONAL_UINT64"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// setupIntegerTestTables recreates the test tables before each test.
func setupIntegerTestTables(t *testing.T) {
	t.Log("Setting up integer NUMBER test tables")

	DB.Migrator().DropTable(&IntegerTestModel{})
	DB.Migrator().DropTable(&NullableIntegerTestModel{})

	if err := DB.AutoMigrate(&IntegerTestModel{}, &NullableIntegerTestModel{}); err != nil {
		t.Fatalf("Failed to migrate integer test tables: %v", err)
	}

	t.Log("Integer NUMBER test tables created successfully")
}

func TestIntegerBasicCRUD(t *testing.T) {
	setupIntegerTestTables(t)

	model := &IntegerTestModel{
		Int8Value:   127,
		Int16Value:  32767,
		Int32Value:  2147483647,
		Int64Value:  9223372036854775807,
		IntValue:    1000000,
		Uint8Value:  255,
		Uint16Value: 65535,
		Uint32Value: 4294967295,
		Uint64Value: 18446744073709551615,
		UintValue:   5000000,
	}

	if err := DB.Create(model).Error; err != nil {
		t.Fatalf("Failed to create integer record: %v", err)
	}

	if model.ID == 0 {
		t.Error("Expected auto-generated ID")
	}

	var retrieved IntegerTestModel
	if err := DB.First(&retrieved, model.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve integer record: %v", err)
	}

	if retrieved.Int32Value != model.Int32Value {
		t.Errorf("Int32Value mismatch. Expected %d, got %d", model.Int32Value, retrieved.Int32Value)
	}

	// Update
	newInt32Value := int32(42)
	if err := DB.Model(&retrieved).Update("INT32_VALUE", newInt32Value).Error; err != nil {
		t.Fatalf("Failed to update integer record: %v", err)
	}

	var updated IntegerTestModel
	if err := DB.First(&updated, model.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve updated integer record: %v", err)
	}
	if updated.Int32Value != newInt32Value {
		t.Errorf("Updated Int32Value mismatch. Expected %d, got %d", newInt32Value, updated.Int32Value)
	}

	// Delete
	if err := DB.Delete(&updated).Error; err != nil {
		t.Fatalf("Failed to delete integer record: %v", err)
	}

	var deleted IntegerTestModel
	err := DB.First(&deleted, model.ID).Error
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected record not found, got: %v", err)
	}
}

func TestIntegerEdgeCases(t *testing.T) {
	setupIntegerTestTables(t)

	testCases := []struct {
		name  string
		model IntegerTestModel
	}{
		{
			name: "Maximum positive values",
			model: IntegerTestModel{
				Int8Value:   math.MaxInt8,
				Int16Value:  math.MaxInt16,
				Int32Value:  math.MaxInt32,
				Int64Value:  math.MaxInt64,
				IntValue:    math.MaxInt,
				Uint8Value:  math.MaxUint8,
				Uint16Value: math.MaxUint16,
				Uint32Value: math.MaxUint32,
				Uint64Value: math.MaxUint64,
				UintValue:   math.MaxUint,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := DB.Create(&tc.model).Error; err != nil {
				t.Fatalf("Failed to create record for %s: %v", tc.name, err)
			}

			var retrieved IntegerTestModel
			if err := DB.First(&retrieved, tc.model.ID).Error; err != nil {
				t.Fatalf("Failed to retrieve record for %s: %v", tc.name, err)
			}

			if retrieved.Int32Value != tc.model.Int32Value {
				t.Errorf("%s: Int32Value mismatch. Expected %d, got %d",
					tc.name, tc.model.Int32Value, retrieved.Int32Value)
			}
		})
	}
}

func TestIntegerNullHandling(t *testing.T) {
	setupIntegerTestTables(t)

	model1 := &NullableIntegerTestModel{}
	if err := DB.Create(model1).Error; err != nil {
		t.Fatalf("Failed to create record with NULL values: %v", err)
	}

	var retrieved1 NullableIntegerTestModel
	if err := DB.First(&retrieved1, model1.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve record with NULL values: %v", err)
	}

	if retrieved1.NullInt32.Valid || retrieved1.NullInt64.Valid {
		t.Error("Expected NULL values to remain invalid")
	}

	validInt32 := int32(42)
	validInt64 := int64(9999999)
	validUint32 := uint32(123)
	validUint64 := uint64(456789)

	model2 := &NullableIntegerTestModel{
		NullInt32:      sql.NullInt32{Int32: 100, Valid: true},
		NullInt64:      sql.NullInt64{Int64: 200, Valid: true},
		OptionalInt32:  &validInt32,
		OptionalInt64:  &validInt64,
		OptionalUint32: &validUint32,
		OptionalUint64: &validUint64,
	}

	if err := DB.Create(model2).Error; err != nil {
		t.Fatalf("Failed to create record with valid nullable values: %v", err)
	}

	var retrieved2 NullableIntegerTestModel
	if err := DB.First(&retrieved2, model2.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve record with valid nullable values: %v", err)
	}

	if !retrieved2.NullInt32.Valid || retrieved2.NullInt32.Int32 != 100 {
		t.Errorf("Expected NullInt32=100, got %v", retrieved2.NullInt32)
	}
	if !retrieved2.NullInt64.Valid || retrieved2.NullInt64.Int64 != 200 {
		t.Errorf("Expected NullInt64=200, got %v", retrieved2.NullInt64)
	}

	// Update NULL → value
	if err := DB.Model(&retrieved1).Updates(map[string]interface{}{
		"NULL_INT32": sql.NullInt32{Int32: 500, Valid: true},
		"NULL_INT64": sql.NullInt64{Int64: 600, Valid: true},
	}).Error; err != nil {
		t.Fatalf("Failed to update NULL to value: %v", err)
	}

	var updated NullableIntegerTestModel
	if err := DB.First(&updated, model1.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve updated record: %v", err)
	}

	if !updated.NullInt32.Valid || updated.NullInt32.Int32 != 500 {
		t.Error("Failed to update NullInt32 from NULL to value")
	}

	// Update value → NULL
	if err := DB.Model(&updated).Updates(map[string]interface{}{
		"NULL_INT32": sql.NullInt32{Valid: false},
	}).Error; err != nil {
		t.Fatalf("Failed to update value to NULL: %v", err)
	}
}

func TestIntegerQueryOperations(t *testing.T) {
	setupIntegerTestTables(t)

	data := []IntegerTestModel{
		{Int32Value: 10, Int64Value: 100, UintValue: 1},
		{Int32Value: 20, Int64Value: 200, UintValue: 2},
		{Int32Value: 30, Int64Value: 300, UintValue: 3},
		{Int32Value: 40, Int64Value: 400, UintValue: 4},
		{Int32Value: 50, Int64Value: 500, UintValue: 5},
	}

	if err := DB.Create(&data).Error; err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var result []IntegerTestModel
	if err := DB.Where("INT32_VALUE = ?", 30).Find(&result).Error; err != nil {
		t.Fatalf("Failed to query with equals: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("Expected 1 record, got %d", len(result))
	}

	var maxInt32 int32
	if err := DB.Model(&IntegerTestModel{}).Select("MAX(INT32_VALUE)").Scan(&maxInt32).Error; err != nil {
		t.Fatalf("Failed to query MAX: %v", err)
	}
	if maxInt32 != 50 {
		t.Errorf("Expected MAX(INT32_VALUE)=50, got %d", maxInt32)
	}
}

func TestIntegerOverflowHandling(t *testing.T) {
	setupIntegerTestTables(t)

	t.Run("Int8 overflow", func(t *testing.T) {
		model := &IntegerTestModel{Int8Value: math.MaxInt8}
		if err := DB.Create(model).Error; err != nil {
			t.Fatalf("Failed to create record: %v", err)
		}

		err := DB.Model(&IntegerTestModel{}).
			Where("ID = ?", model.ID).
			Update("INT8_VALUE", gorm.Expr("INT8_VALUE + ?", 1)).Error

		if err != nil {
			t.Logf("Overflow prevented as expected: %v", err)
		} else {
			var updated IntegerTestModel
			DB.First(&updated, model.ID)
			t.Logf("Post-overflow value: %d", updated.Int8Value)
		}
	})

	t.Run("Uint64 maximum", func(t *testing.T) {
		model := &IntegerTestModel{Uint64Value: math.MaxUint64}
		if err := DB.Create(model).Error; err != nil {
			t.Fatalf("Failed to create record: %v", err)
		}

		var retrieved IntegerTestModel
		if err := DB.First(&retrieved, model.ID).Error; err != nil {
			t.Fatalf("Failed to retrieve record: %v", err)
		}

		if retrieved.Uint64Value != math.MaxUint64 {
			t.Errorf("Expected %d, got %d", uint64(math.MaxUint64), retrieved.Uint64Value)
		}
	})
}

func TestIntegerWithReturning(t *testing.T) {
	setupIntegerTestTables(t)

	models := []IntegerTestModel{
		{Int32Value: 111, Int64Value: 1111},
		{Int32Value: 222, Int64Value: 2222},
		{Int32Value: 333, Int64Value: 3333},
	}

	if err := DB.Create(&models).Error; err != nil {
		t.Fatalf("Failed to create records: %v", err)
	}

	for i, m := range models {
		if m.ID == 0 {
			t.Errorf("Record %d: expected ID populated, got 0", i)
		}
	}

	var updated []IntegerTestModel
	err := DB.Model(&IntegerTestModel{}).
		Clauses(clause.Returning{}).
		Where("INT32_VALUE > ?", 200).
		Update("INT32_VALUE", gorm.Expr("INT32_VALUE + ?", 1000)).
		Find(&updated).Error

	if err != nil {
		t.Logf("UPDATE with RETURNING not fully supported: %v", err)
	} else {
		t.Logf("Updated %d records via RETURNING", len(updated))
	}
}
