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
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBasicCRUD_JSONText(t *testing.T) {
	type JsonRecord struct {
		ID         uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name       string         `gorm:"column:name"`
		Properties datatypes.JSON `gorm:"column:properties"`
	}

	DB.Migrator().DropTable(&JsonRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// INSERT
	rec := JsonRecord{
		Name:       "json-text",
		Properties: datatypes.JSON([]byte(`{"env":"prod","owner":"team-x"}`)),
	}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID to be set")
	}

	// UPDATE (with RETURNING)
	var ret JsonRecord
	if err := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "record_id"},
				{Name: "name"},
				{Name: "properties"},
			},
		}).
		Model(&ret).
		Where("\"record_id\" = ?", rec.ID).
		Updates(map[string]any{
			"name":       "json-text-upd",
			"properties": datatypes.JSON([]byte(`{"env":"staging","owner":"team-y","flag":true}`)),
		}).Error; err != nil {
		t.Fatalf("update returning failed: %v", err)
	}
	if ret.ID != rec.ID || ret.Name != "json-text-upd" || len(ret.Properties) == 0 {
		t.Fatalf("unexpected returning row: %#v", ret)
	}

	// DELETE (with RETURNING)
	var deleted []JsonRecord
	if err := DB.
		Where("\"record_id\" = ?", rec.ID).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "record_id"},
				{Name: "name"},
				{Name: "properties"},
			},
		}).
		Delete(&deleted).Error; err != nil {
		t.Fatalf("delete returning failed: %v", err)
	}
	if len(deleted) != 1 || deleted[0].ID != rec.ID {
		t.Fatalf("unexpected deleted rows: %#v", deleted)
	}

	// verify gone
	var check JsonRecord
	err := DB.First(&check, "\"record_id\" = ?", rec.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestBasicCRUD_RawMessage(t *testing.T) {
	type RawRecord struct {
		ID         uint            `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name       string          `gorm:"column:name"`
		Properties json.RawMessage `gorm:"column:properties"`
	}

	DB.Migrator().DropTable(&RawRecord{})
	if err := DB.AutoMigrate(&RawRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// INSERT
	rec := RawRecord{
		Name:       "raw-json",
		Properties: json.RawMessage(`{"a":1,"b":"x"}`),
	}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID to be set")
	}

	// UPDATE (with RETURNING)
	var ret RawRecord
	if err := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "record_id"},
				{Name: "name"},
				{Name: "properties"},
			},
		}).
		Model(&ret).
		Where("\"record_id\" = ?", rec.ID).
		Updates(map[string]any{
			"name":       "raw-json-upd",
			"properties": json.RawMessage(`{"a":2,"c":true}`),
		}).Error; err != nil {
		t.Fatalf("update returning failed: %v", err)
	}
	if ret.ID != rec.ID || ret.Name != "raw-json-upd" || len(ret.Properties) == 0 {
		t.Fatalf("unexpected returning row: %#v", ret)
	}

	// DELETE (with RETURNING)
	var deleted []RawRecord
	if err := DB.
		Where("\"record_id\" = ?", rec.ID).
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "record_id"},
				{Name: "name"},
				{Name: "properties"},
			},
		}).
		Delete(&deleted).Error; err != nil {
		t.Fatalf("delete returning failed: %v", err)
	}
	if len(deleted) != 1 || deleted[0].ID != rec.ID {
		t.Fatalf("unexpected deleted rows: %#v", deleted)
	}

	// verify gone
	var check RawRecord
	err := DB.First(&check, "\"record_id\" = ?", rec.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}
