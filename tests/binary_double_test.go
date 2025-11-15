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
)

const (
	// doubleComparisonEpsilon is used for direct value comparison
	doubleComparisonEpsilon = 1e-15
	// doubleAggregateEpsilon is used for aggregate operations (SUM, AVG, etc.)
	doubleAggregateEpsilon = 1e-10
)

type BinaryDoubleTest struct {
	ID             uint         `gorm:"column:ID;primaryKey"`
	DoubleValue    float64      `gorm:"column:DOUBLE_VALUE;type:BINARY_DOUBLE"`
	NullableDouble *float64     `gorm:"column:NULLABLE_DOUBLE;type:BINARY_DOUBLE"`
	SQLNullFloat   sql.NullFloat64 `gorm:"column:SQL_NULL_FLOAT;type:BINARY_DOUBLE"`
}

func (BinaryDoubleTest) TableName() string {
	return "BINARY_DOUBLE_TESTS"
}

func TestBinaryDoubleBasicCRUD(t *testing.T) {
	DB.Migrator().DropTable(&BinaryDoubleTest{})
	if err := DB.AutoMigrate(&BinaryDoubleTest{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// CREATE - Insert basic double value
	testValue := 3.141592653589793
	bd1 := BinaryDoubleTest{DoubleValue: testValue}
	if err := DB.Create(&bd1).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// READ - Fetch and verify
	var got BinaryDoubleTest
	if err := DB.First(&got, bd1.ID).Error; err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	
	if math.Abs(got.DoubleValue - testValue) > doubleComparisonEpsilon {
		t.Errorf("expected %v, got %v", testValue, got.DoubleValue)
	}

	// UPDATE - Modify the value
	newValue := 2.718281828459045
	if err := DB.Model(&got).Update("DoubleValue", newValue).Error; err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Verify update
	var updated BinaryDoubleTest
	if err := DB.First(&updated, bd1.ID).Error; err != nil {
		t.Fatalf("fetch after update failed: %v", err)
	}
	if math.Abs(updated.DoubleValue - newValue) > doubleComparisonEpsilon {
		t.Errorf("expected %v after update, got %v", newValue, updated.DoubleValue)
	}

	// DELETE
	if err := DB.Delete(&updated).Error; err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Verify deletion
	var deleted BinaryDoubleTest
	err := DB.First(&deleted, bd1.ID).Error
	if err == nil {
		t.Error("expected record to be deleted")
	}
}

func TestBinaryDoubleSpecialValues(t *testing.T) {
	DB.Migrator().DropTable(&BinaryDoubleTest{})
	DB.AutoMigrate(&BinaryDoubleTest{})

	testCases := []struct {
		name  string
		value float64
	}{
		{"Positive Infinity", math.Inf(1)},
		{"Negative Infinity", math.Inf(-1)},
		{"NaN", math.NaN()},
		{"Max Float64", math.MaxFloat64},
		{"Min Float64", -math.MaxFloat64},
		{"Smallest Positive", math.SmallestNonzeroFloat64},
		{"Zero", 0.0},
		{"Negative Zero", -0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bd := BinaryDoubleTest{DoubleValue: tc.value}
			if err := DB.Create(&bd).Error; err != nil {
				t.Fatalf("failed to insert %s: %v", tc.name, err)
			}

			var got BinaryDoubleTest
			if err := DB.First(&got, bd.ID).Error; err != nil {
				t.Fatalf("failed to fetch %s: %v", tc.name, err)
			}

			if math.IsNaN(tc.value) {
				if !math.IsNaN(got.DoubleValue) {
					t.Errorf("expected NaN, got %v", got.DoubleValue)
				}
			} else if math.IsInf(tc.value, 1) {
				if !math.IsInf(got.DoubleValue, 1) {
					t.Errorf("expected +Inf, got %v", got.DoubleValue)
				}
			} else if math.IsInf(tc.value, -1) {
				if !math.IsInf(got.DoubleValue, -1) {
					t.Errorf("expected -Inf, got %v", got.DoubleValue)
				}
			} else if tc.value == 0.0 || tc.value == -0.0 {
				if got.DoubleValue != 0.0 {
					t.Errorf("expected 0, got %v", got.DoubleValue)
				}
			} else {
				if math.Abs(got.DoubleValue - tc.value) > doubleComparisonEpsilon {
					t.Errorf("expected %v, got %v", tc.value, got.DoubleValue)
				}
			}
		})
	}
}

func TestBinaryDoubleNullableColumn(t *testing.T) {
	DB.Migrator().DropTable(&BinaryDoubleTest{})
	DB.AutoMigrate(&BinaryDoubleTest{})

	// Test NULL value
	bd1 := BinaryDoubleTest{
		DoubleValue:    1.23,
		NullableDouble: nil,
	}
	if err := DB.Create(&bd1).Error; err != nil {
		t.Fatalf("failed to insert with NULL: %v", err)
	}

	var got1 BinaryDoubleTest
	if err := DB.First(&got1, bd1.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got1.NullableDouble != nil {
		t.Errorf("expected NULL, got %v", *got1.NullableDouble)
	}

	// Test with non-NULL value
	val := 456.789
	bd2 := BinaryDoubleTest{
		DoubleValue:    2.34,
		NullableDouble: &val,
	}
	if err := DB.Create(&bd2).Error; err != nil {
		t.Fatalf("failed to insert with value: %v", err)
	}

	var got2 BinaryDoubleTest
	if err := DB.First(&got2, bd2.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got2.NullableDouble == nil {
		t.Error("expected non-NULL value")
	} else if math.Abs(*got2.NullableDouble - val) > doubleComparisonEpsilon {
		t.Errorf("expected %v, got %v", val, *got2.NullableDouble)
	}

	// Update to NULL
	if err := DB.Model(&got2).Update("NullableDouble", nil).Error; err != nil {
		t.Fatalf("failed to update to NULL: %v", err)
	}

	var got3 BinaryDoubleTest
	if err := DB.First(&got3, bd2.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got3.NullableDouble != nil {
		t.Errorf("expected NULL after update, got %v", *got3.NullableDouble)
	}
}

func TestBinaryDoubleSQLNullFloat(t *testing.T) {
	DB.Migrator().DropTable(&BinaryDoubleTest{})
	DB.AutoMigrate(&BinaryDoubleTest{})

	// Test with valid value
	bd1 := BinaryDoubleTest{
		DoubleValue:  1.0,
		SQLNullFloat: sql.NullFloat64{Float64: 123.456, Valid: true},
	}
	if err := DB.Create(&bd1).Error; err != nil {
		t.Fatalf("failed to create with sql.NullFloat64: %v", err)
	}

	var got1 BinaryDoubleTest
	if err := DB.First(&got1, bd1.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !got1.SQLNullFloat.Valid {
		t.Error("expected Valid to be true")
	}
	if math.Abs(got1.SQLNullFloat.Float64 - 123.456) > doubleComparisonEpsilon {
		t.Errorf("expected 123.456, got %v", got1.SQLNullFloat.Float64)
	}

	// Test with invalid (NULL) value
	bd2 := BinaryDoubleTest{
		DoubleValue:  2.0,
		SQLNullFloat: sql.NullFloat64{Valid: false},
	}
	if err := DB.Create(&bd2).Error; err != nil {
		t.Fatalf("failed to create with NULL sql.NullFloat64: %v", err)
	}

	var got2 BinaryDoubleTest
	if err := DB.First(&got2, bd2.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got2.SQLNullFloat.Valid {
		t.Error("expected Valid to be false for NULL")
	}
}

func TestBinaryDoubleArithmeticOperations(t *testing.T) {
	DB.Migrator().DropTable(&BinaryDoubleTest{})
	DB.AutoMigrate(&BinaryDoubleTest{})

	// Insert test data
	values := []float64{10.5, 20.3, 30.7, 40.1, 50.9}
	for _, v := range values {
		bd := BinaryDoubleTest{DoubleValue: v}
		if err := DB.Create(&bd).Error; err != nil {
			t.Fatalf("failed to insert %v: %v", v, err)
		}
	}

	// Test SUM
	var sum float64
	if err := DB.Model(&BinaryDoubleTest{}).Select("SUM(DOUBLE_VALUE)").Scan(&sum).Error; err != nil {
		t.Fatalf("failed to calculate SUM: %v", err)
	}
	expectedSum := 10.5 + 20.3 + 30.7 + 40.1 + 50.9
	if math.Abs(sum - expectedSum) > doubleAggregateEpsilon {
		t.Errorf("expected sum %v, got %v", expectedSum, sum)
	}

	// Test AVG
	var avg float64
	if err := DB.Model(&BinaryDoubleTest{}).Select("AVG(DOUBLE_VALUE)").Scan(&avg).Error; err != nil {
		t.Fatalf("failed to calculate AVG: %v", err)
	}
	expectedAvg := expectedSum / 5
	if math.Abs(avg - expectedAvg) > doubleAggregateEpsilon {
		t.Errorf("expected avg %v, got %v", expectedAvg, avg)
	}

	// Test MIN/MAX
	var min, max float64
	if err := DB.Model(&BinaryDoubleTest{}).Select("MIN(DOUBLE_VALUE)").Scan(&min).Error; err != nil {
		t.Fatalf("failed to calculate MIN: %v", err)
	}
	if err := DB.Model(&BinaryDoubleTest{}).Select("MAX(DOUBLE_VALUE)").Scan(&max).Error; err != nil {
		t.Fatalf("failed to calculate MAX: %v", err)
	}
	if math.Abs(min - 10.5) > doubleAggregateEpsilon {
		t.Errorf("expected min 10.5, got %v", min)
	}
	if math.Abs(max - 50.9) > doubleAggregateEpsilon {
		t.Errorf("expected max 50.9, got %v", max)
	}
}

func TestBinaryDoubleRangeQueries(t *testing.T) {
	DB.Migrator().DropTable(&BinaryDoubleTest{})
	DB.AutoMigrate(&BinaryDoubleTest{})

	// Insert test data with various ranges
	testData := []float64{-100.5, -50.0, 0.0, 25.5, 50.0, 75.75, 100.0, 150.25}
	for _, v := range testData {
		bd := BinaryDoubleTest{DoubleValue: v}
		if err := DB.Create(&bd).Error; err != nil {
			t.Fatalf("failed to insert %v: %v", v, err)
		}
	}

	// Test BETWEEN query
	var results []BinaryDoubleTest
	if err := DB.Where("DOUBLE_VALUE BETWEEN ? AND ?", 0.0, 100.0).Find(&results).Error; err != nil {
		t.Fatalf("BETWEEN query failed: %v", err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}

	// Test greater than query
	var gtResults []BinaryDoubleTest
	if err := DB.Where("DOUBLE_VALUE > ?", 50.0).Find(&gtResults).Error; err != nil {
		t.Fatalf("> query failed: %v", err)
	}
	if len(gtResults) != 3 {
		t.Errorf("expected 3 results for > 50.0, got %d", len(gtResults))
	}

	// Test less than or equal query
	var lteResults []BinaryDoubleTest
	if err := DB.Where("DOUBLE_VALUE <= ?", 0.0).Find(&lteResults).Error; err != nil {
		t.Fatalf("<= query failed: %v", err)
	}
	if len(lteResults) != 3 {
		t.Errorf("expected 3 results for <= 0.0, got %d", len(lteResults))
	}

	// Test equality with floating point
	var eqResults []BinaryDoubleTest
	if err := DB.Where("DOUBLE_VALUE = ?", 25.5).Find(&eqResults).Error; err != nil {
		t.Fatalf("= query failed: %v", err)
	}
	if len(eqResults) != 1 {
		t.Errorf("expected 1 result for = 25.5, got %d", len(eqResults))
	}
}