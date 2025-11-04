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
	"time"

	"gorm.io/gorm"
)

type DateModel struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	EventName   string    `gorm:"size:100"`
	EventDate   time.Time `gorm:"column:EVENT_DATE;type:DATE"`
	OptionalCol *time.Time `gorm:"column:OPTIONAL_DATE"`
}

func setupDateTests(t *testing.T) {
	t.Log("Setting up DATE test table")

	// Drop the table safely
	dropSQL := `
	BEGIN
		EXECUTE IMMEDIATE 'DROP TABLE "date_models" CASCADE CONSTRAINTS';
	EXCEPTION
		WHEN OTHERS THEN
			IF SQLCODE != -942 THEN RAISE; END IF;
	END;`
	if err := DB.Exec(dropSQL).Error; err != nil {
		t.Fatalf("Failed to drop table: %v", err)
	}

	// migrate
	if err := DB.AutoMigrate(&DateModel{}); err != nil {
		t.Fatalf("Failed to migrate date_models: %v", err)
	}

	t.Log("DATE test table created successfully")
}

func TestDate_InsertAndRetrieve(t *testing.T) {
	setupDateTests(t)

	now := time.Now().Truncate(time.Second)
	record := DateModel{
		EventName: "Launch",
		EventDate: now,
	}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	var out DateModel
	err := DB.First(&out, record.ID).Error
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if !out.EventDate.Equal(now) {
		t.Errorf("Expected %v, got %v", now, out.EventDate)
	}
}

func TestDate_NullAndOptionalColumns(t *testing.T) {
	setupDateTests(t)

	record := DateModel{
		EventName: "NullDate",
	}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("Insert with null date failed: %v", err)
	}

	var fetched DateModel
	err := DB.First(&fetched, record.ID).Error
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if fetched.OptionalCol != nil {
		t.Errorf("Expected OptionalCol to be nil, got %v", fetched.OptionalCol)
	}
}

func TestDate_UpdateAndCompare(t *testing.T) {
	setupDateTests(t)

	initial := DateModel{
		EventName: "CompareTest",
		EventDate: time.Date(2025, 10, 30, 10, 10, 10, 0, time.UTC),
	}
	if err := DB.Create(&initial).Error; err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updateTime := initial.EventDate.Add(2 * time.Hour)
	if err := DB.Model(&initial).Update("EVENT_DATE", updateTime).Error; err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	var fetched DateModel
	err := DB.First(&fetched, initial.ID).Error
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if !fetched.EventDate.Equal(updateTime) {
		t.Errorf("Expected %v, got %v", updateTime, fetched.EventDate)
	}
}

func TestDate_QueryByDateRange(t *testing.T) {
	setupDateTests(t)

	base := time.Now().Truncate(time.Second)
	records := []DateModel{
		{EventName: "Day1", EventDate: base.Add(-24 * time.Hour)},
		{EventName: "Day2", EventDate: base},
		{EventName: "Day3", EventDate: base.Add(24 * time.Hour)},
	}
	if err := DB.Create(&records).Error; err != nil {
		t.Fatalf("Bulk insert failed: %v", err)
	}

	var results []DateModel
	err := DB.Where("\"EVENT_DATE\" BETWEEN ? AND ?", base.Add(-12*time.Hour), base.Add(12*time.Hour)).Find(&results).Error
	if err != nil {
		t.Fatalf("Range query failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 record, got %d", len(results))
	}
}

func TestDate_InvalidDateHandling(t *testing.T) {
	setupDateTests(t)

	// insert invalid date
	err := DB.Exec(`INSERT INTO "DATE_MODELS" ("EVENT_NAME","EVENT_DATE") VALUES ('BadDate', TO_DATE('2025-13-40','YYYY-MM-DD'))`).Error
	if err == nil {
		t.Error("Expected error inserting invalid date, got nil")
	}
}

func TestDate_TimePrecisionLoss(t *testing.T) {
	setupDateTests(t)

	withMillis := time.Now().Truncate(time.Millisecond)
	record := DateModel{
		EventName: "PrecisionLoss",
		EventDate: withMillis,
	}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	var fetched DateModel
	_ = DB.First(&fetched, record.ID).Error
	if !fetched.EventDate.Truncate(time.Second).Equal(withMillis.Truncate(time.Second)) {
		t.Errorf("Expected time match up to seconds precision, got %v", fetched.EventDate)
	}
}

func TestDate_ZeroDateHandling(t *testing.T) {
	setupDateTests(t)

	zeroTime := time.Time{}
	record := DateModel{
		EventName: "ZeroDate",
		EventDate: zeroTime,
	}
	err := DB.Create(&record).Error
	if err != nil && err != gorm.ErrInvalidData {
		t.Errorf("Unexpected error inserting zero date: %v", err)
	}
}
