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
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBasicCRUD_JSONText(t *testing.T) {
	type JsonRecord struct {
		ID            uint            `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name          string          `gorm:"column:name"`
		Properties    datatypes.JSON  `gorm:"column:properties"`
		PropertiesPtr *datatypes.JSON `gorm:"column:propertiesPtr"`
	}

	DB.Migrator().DropTable(&JsonRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// INSERT
	json := datatypes.JSON([]byte(`{"env":"prod","owner":"team-x"}`))
	rec := JsonRecord{
		Name:          "json-text",
		Properties:    json,
		PropertiesPtr: &json,
	}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID to be set")
	}

	// UPDATE (with RETURNING)
	updateJson := datatypes.JSON([]byte(`{"env":"staging","owner":"team-y","flag":true}`))
	var ret JsonRecord
	if err := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "record_id"},
				{Name: "name"},
				{Name: "properties"},
				{Name: "propertiesPtr"},
			},
		}).
		Model(&ret).
		Where("\"record_id\" = ?", rec.ID).
		Updates(map[string]any{
			"name":          "json-text-upd",
			"properties":    updateJson,
			"propertiesPtr": &updateJson,
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
				{Name: "propertiesPtr"},
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

func TestJSONFunctions(t *testing.T) {
	type JsonNative struct {
		ID  uint           `gorm:"column:id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}

	DB.Migrator().DropTable(&JsonNative{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonNative{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	var records = []JsonNative{
		{ID: 1, Doc: datatypes.JSON([]byte(`{"x":1,"s":"a"}`))},
		{ID: 2, Doc: datatypes.JSON([]byte(`{"x":2,"s":"ab"}`))},
		{ID: 3, Doc: datatypes.JSON([]byte(`{"x":3,"s":"abc"}`))},
	}

	if err := DB.Table("json_natives").Save(&records).Error; err != nil {
		t.Fatalf("insert into native JSON column failed: %v", err)
	}

	// Check ColumnTypes
	if cols, err := DB.Migrator().ColumnTypes("json_natives"); err == nil {
		found := false
		for _, c := range cols {
			if strings.Contains(strings.ToUpper(c.DatabaseTypeName()), "JSON") {
				found = true
				t.Logf("DatabaseTypeName for 'doc' column: %s", c.DatabaseTypeName())
				break
			}
		}
		if !found {
			t.Fatalf("DatabaseTypeName did not report JSON")
		}
	}

	// Filter by JSON_VALUE
	var cnt int64
	if err := DB.Table("json_natives").
		Where(`JSON_VALUE("doc",'$.x') = ?`, 1).
		Count(&cnt).Error; err != nil {
		t.Fatalf("JSON_VALUE on native JSON column failed: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 row, got %d", cnt)
	}

	cnt = 0
	if err := DB.Table("\"json_natives\" jn").
		Where(`JSON_VALUE("doc",'$.x') >= ?`, 1).
		Count(&cnt).Error; err != nil {
		t.Fatalf("JSON_VALUE on native JSON column failed: %v", err)
	}
	if cnt != 3 {
		t.Fatalf("expected 3 row, got %d", cnt)
	}

	// Update with JSON_TRANSFORM
	if err := DB.Table("json_natives").
		Where(`JSON_VALUE("doc",'$.x') = ?`, 1).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.s' = ?)`, "b")).Error; err != nil {
		t.Fatalf("JSON_TRANSFORM on native JSON failed: %v", err)
	}

	// Check the updated value
	var updated JsonNative
	if err := DB.Table("json_natives").First(&updated, `"id" = ?`, 1).Error; err != nil {
		t.Fatalf("fetch after update failed: %v", err)
	}
	var m map[string]any
	if b, err := json.Marshal(updated.Doc); err != nil {
		t.Fatalf("marshal updated doc failed: %v", err)
	} else if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal updated doc failed: %v", err)
	}
	if m["x"] != float64(1) || m["s"] != "b" {
		t.Fatalf("unexpected updated doc: %#v", m)
	}

	// Clean up
	DB.Migrator().DropTable(&JsonNative{})
}

func TestJSONArrayFunction(t *testing.T) {
	type JsonArrRec struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Arr datatypes.JSON `gorm:"column:arr"`
	}

	DB.Migrator().DropTable(&JsonArrRec{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonArrRec{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	rec := JsonArrRec{Arr: datatypes.JSON([]byte(`[]`))}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Set array using JSON_ARRAY and serialize to CLOB for CLOB-backed column
	if err := DB.Model(&JsonArrRec{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("arr", gorm.Expr(`JSON_SERIALIZE(JSON_ARRAY(?, ?, ?) RETURNING CLOB)`, 1, 2, 3)).Error; err != nil {
		t.Fatalf("JSON_ARRAY set failed: %v", err)
	}

	var out JsonArrRec
	if err := DB.First(&out, `"record_id" = ?`, rec.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	var arr []int
	if b, err := json.Marshal(out.Arr); err != nil {
		t.Fatalf("marshal arr failed: %v", err)
	} else if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatalf("unmarshal arr failed: %v", err)
	}
	if len(arr) != 3 || arr[0] != 1 || arr[1] != 2 || arr[2] != 3 {
		t.Fatalf("unexpected arr: %#v", arr)
	}
}

func TestJSONTableWithAlias(t *testing.T) {
	type JsonTableAliasRec struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}

	// Ensure clean table
	DB.Migrator().DropTable(&JsonTableAliasRec{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonTableAliasRec{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// Insert two rows with a JSON array under "items"
	rec1 := JsonTableAliasRec{
		Doc: datatypes.JSON([]byte(`{"items":[{"id":1,"name":"a"},{"id":2,"name":"b"}]}`)),
	}
	rec2 := JsonTableAliasRec{
		Doc: datatypes.JSON([]byte(`{"items":[{"id":3,"name":"c"}]}`)),
	}
	if err := DB.Create(&[]JsonTableAliasRec{rec1, rec2}).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	rows, err := DB.
		Table(`"json_table_alias_recs" t`).
		Select(`t."record_id", jt."item_id", jt."item_name"`).
		Joins(`CROSS JOIN JSON_TABLE(t."doc", '$.items[*]' COLUMNS ("item_id" NUMBER PATH '$.id', "item_name" VARCHAR2(100) PATH '$.name')) jt`).
		Where(`jt."item_name" = ?`, "b").
		Rows()
	if err != nil {
		t.Fatalf("JSON_TABLE alias query failed: %v", err)
	}
	defer rows.Close()

	var cnt int
	for rows.Next() {
		var rid, itemID int64
		var itemName string
		if err := rows.Scan(&rid, &itemID, &itemName); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		cnt++
	}
	if cnt != 1 {
		t.Fatalf("expected 1 projected row with name='b', got %d", cnt)
	}

	rows2, err := DB.
		Table(`"json_table_alias_recs" t`).
		Select(`t."record_id", jt."item_id", jt."item_name"`).
		Joins(`CROSS JOIN JSON_TABLE(t."doc", '$.items[*]' COLUMNS ("item_id" NUMBER PATH '$.id', "item_name" VARCHAR2(100) PATH '$.name')) jt`).
		Rows()
	if err != nil {
		t.Fatalf("JSON_TABLE alias full projection failed: %v", err)
	}
	defer rows2.Close()

	total := 0
	for rows2.Next() {
		var rid, itemID int64
		var itemName string
		if err := rows2.Scan(&rid, &itemID, &itemName); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		total++
	}
	if total != 3 {
		t.Fatalf("expected 3 projected items total, got %d", total)
	}

	rows3, err := DB.
		Table(`"json_table_alias_recs"`).
		Select(`"record_id", jt."item_id", jt."item_name"`).
		Joins(`CROSS JOIN JSON_TABLE("doc", '$.items[*]' COLUMNS ("item_id" NUMBER PATH '$.id', "item_name" VARCHAR2(100) PATH '$.name')) jt`).
		Rows()
	if err != nil {
		t.Fatalf("JSON_TABLE auto-alias full projection failed: %v", err)
	}
	defer rows3.Close()

	total = 0
	for rows3.Next() {
		var rid, itemID int64
		var itemName string
		if err := rows3.Scan(&rid, &itemID, &itemName); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		total++
	}
	if total != 3 {
		t.Fatalf("expected 3 projected items total with auto alias, got %d", total)
	}
}

func TestJSONObject(t *testing.T) {
	t.Skip("Skipping test due to issue #96")
	type JsonGenericObjectOnly struct {
		ID     uint                                `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name   string                              `gorm:"column:name"`
		Obj    datatypes.JSONType[map[string]any]  `gorm:"column:obj"`
		ObjPtr *datatypes.JSONType[map[string]any] `gorm:"column:obj_ptr"`
	}

	DB.Migrator().DropTable(&JsonGenericObjectOnly{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonGenericObjectOnly{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// INSERT
	obj := datatypes.NewJSONType(map[string]any{"env": "qa", "count": 7})
	obj2 := datatypes.NewJSONType(map[string]any{"owner": "team-z"})
	rec := JsonGenericObjectOnly{
		Name:   "json-generic-obj-only",
		Obj:    obj,
		ObjPtr: &obj2,
	}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID set")
	}

	// Check the inserted object
	var byExists JsonGenericObjectOnly
	if err := DB.Where(`JSON_EXISTS("obj", '$?(@.env == "qa")')`).First(&byExists).Error; err != nil {
		t.Fatalf("JSON_EXISTS on obj failed: %v", err)
	}
	if byExists.ID != rec.ID {
		t.Fatalf("unexpected row found by JSON_EXISTS(obj): %#v", byExists)
	}

	// UPDATE object using JSON_TRANSFORM: set/overwrite a property
	if err := DB.
		Model(&JsonGenericObjectOnly{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("obj", gorm.Expr(`JSON_TRANSFORM("obj", SET '$.count' = ?)`, 8)).Error; err != nil {
		t.Fatalf("JSON_TRANSFORM(obj SET ...) failed: %v", err)
	}

	// VERIFY
	var countVal int
	if err := DB.Model(&JsonGenericObjectOnly{}).
		Select(`JSON_VALUE("obj", '$.count')`).
		Where(`"record_id" = ?`, rec.ID).
		Scan(&countVal).Error; err != nil {
		t.Fatalf("scan JSON_VALUE(obj.count) failed: %v", err)
	}
	if countVal != 8 {
		t.Fatalf("unexpected obj.count: %d", countVal)
	}

	// DELETE
	var deleted []JsonGenericObjectOnly
	if err := DB.
		Where(`"record_id" = ?`, rec.ID).
		Clauses(clause.Returning{Columns: []clause.Column{
			{Name: "record_id"}, {Name: "name"}, {Name: "obj"}, {Name: "obj_ptr"},
		}}).
		Delete(&deleted).Error; err != nil {
		t.Fatalf("delete returning failed: %v", err)
	}
	if len(deleted) != 1 || deleted[0].ID != rec.ID {
		t.Fatalf("unexpected deleted rows: %#v", deleted)
	}

	// VERIFY gone
	var check JsonGenericObjectOnly
	err := DB.First(&check, `"record_id" = ?`, rec.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestJSONSlice(t *testing.T) {
	t.Skip("Skipping test due to issue #97")
	type JsonGenericSliceOnly struct {
		ID     uint                         `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name   string                       `gorm:"column:name"`
		Arr    datatypes.JSONSlice[string]  `gorm:"column:arr"`
		ArrPtr *datatypes.JSONSlice[string] `gorm:"column:arr_ptr"`
	}

	DB.Migrator().DropTable(&JsonGenericSliceOnly{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonGenericSliceOnly{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// INSERT
	arr := datatypes.NewJSONSlice([]string{"m", "n"})
	arr2 := datatypes.NewJSONSlice([]string{"x"})
	rec := JsonGenericSliceOnly{
		Name:   "json-generic-slice-only",
		Arr:    arr,
		ArrPtr: &arr2,
	}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID set")
	}

	// APPEND to array using JSON_TRANSFORM
	if err := DB.Model(&JsonGenericSliceOnly{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("arr", gorm.Expr(`JSON_TRANSFORM("arr", APPEND '$' = ?)`, "o")).Error; err != nil {
		t.Fatalf("append with JSON_TRANSFORM(arr) failed: %v", err)
	}

	// VERIFY by reading back and unmarshalling
	var out JsonGenericSliceOnly
	if err := DB.First(&out, `"record_id" = ?`, rec.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	var gotArr []string
	if b, err := json.Marshal(out.Arr); err != nil {
		t.Fatalf("marshal Arr failed: %v", err)
	} else if err := json.Unmarshal(b, &gotArr); err != nil {
		t.Fatalf("unmarshal Arr failed: %v", err)
	}
	if len(gotArr) != 3 || gotArr[0] != "m" || gotArr[1] != "n" || gotArr[2] != "o" {
		t.Fatalf("unexpected Arr: %#v", gotArr)
	}

	// DELETE with RETURNING
	var deleted []JsonGenericSliceOnly
	if err := DB.
		Where(`"record_id" = ?`, rec.ID).
		Clauses(clause.Returning{Columns: []clause.Column{
			{Name: "record_id"}, {Name: "name"}, {Name: "arr"}, {Name: "arr_ptr"},
		}}).
		Delete(&deleted).Error; err != nil {
		t.Fatalf("delete returning failed: %v", err)
	}
	if len(deleted) != 1 || deleted[0].ID != rec.ID {
		t.Fatalf("unexpected deleted rows: %#v", deleted)
	}

	// VERIFY gone
	var check JsonGenericSliceOnly
	err := DB.First(&check, `"record_id" = ?`, rec.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestJSONMixedTypes(t *testing.T) {
	t.Skip("Skipping test due to issue #96 and #97")
	type JsonGenericRecord struct {
		ID     uint                                `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name   string                              `gorm:"column:name"`
		Obj    datatypes.JSONType[map[string]any]  `gorm:"column:obj"`
		ObjPtr *datatypes.JSONType[map[string]any] `gorm:"column:objPtr"`
		Arr    datatypes.JSONSlice[string]         `gorm:"column:arr"`
		ArrPtr *datatypes.JSONSlice[string]        `gorm:"column:arrPtr"`
		Doc    datatypes.JSON                      `gorm:"column:doc"`
	}

	DB.Migrator().DropTable(&JsonGenericRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&JsonGenericRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// Build values using datatypes JSON generics
	obj := datatypes.NewJSONType(map[string]any{"env": "dev", "count": 1})
	obj2 := datatypes.NewJSONType(map[string]any{"env": "prod", "owner": "core"})
	arr := datatypes.NewJSONSlice([]string{"a", "b"})
	arr2 := datatypes.NewJSONSlice([]string{"x", "y", "z"})
	doc := datatypes.JSON([]byte(`{"k": "v", "n": 42}`))

	rec := JsonGenericRecord{
		Name:   "json-generics",
		Obj:    obj,
		ObjPtr: &obj2,
		Arr:    arr,
		ArrPtr: &arr2,
		Doc:    doc,
	}

	// INSERT
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create generics failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID set")
	}

	// SELECT
	var byValue JsonGenericRecord
	if err := DB.Where(`JSON_VALUE("doc", '$.k') = ?`, "v").First(&byValue).Error; err != nil {
		t.Fatalf("query by JSON_VALUE(doc) failed: %v", err)
	}
	if byValue.ID != rec.ID {
		t.Fatalf("unexpected row found by JSON_VALUE(doc): %#v", byValue)
	}

	var byExists JsonGenericRecord
	if err := DB.Where(`JSON_EXISTS("obj", '$?(@.env == "dev")')`).First(&byExists).Error; err != nil {
		t.Fatalf("query by JSON_EXISTS(obj) failed: %v", err)
	}
	if byExists.ID != rec.ID {
		t.Fatalf("unexpected row found by JSON_EXISTS(obj): %#v", byExists)
	}

	// UPDATE
	var ret JsonGenericRecord
	if err := DB.
		Clauses(clause.Returning{Columns: []clause.Column{
			{Name: "record_id"}, {Name: "name"}, {Name: "doc"}, {Name: "obj"}, {Name: "arr"},
		}}).
		Model(&ret).
		Where(`"record_id" = ?`, rec.ID).
		Updates(map[string]any{
			// JSON_SET(doc, '$.newProp', 'set-ok') -> Oracle JSON_TRANSFORM
			"doc": gorm.Expr(`JSON_TRANSFORM("doc", SET '$.newProp' = ?)`, "set-ok"),
			// JSON_SET(obj, '$.count', 2)
			"obj": gorm.Expr(`JSON_TRANSFORM("obj", SET '$.count' = ?)`, 2),
		}).Error; err != nil {
		t.Fatalf("update JSON_TRANSFORM failed: %v", err)
	}

	// Verify
	var newProp string
	if err := DB.Table("json_generic_records").
		Select(`JSON_VALUE("doc", '$.newProp')`).
		Where(`"record_id" = ?`, rec.ID).
		Scan(&newProp).Error; err != nil {
		t.Fatalf("scan JSON_VALUE(doc.newProp) failed: %v", err)
	}
	if newProp != "set-ok" {
		t.Fatalf("unexpected doc.newProp: %q", newProp)
	}
	var countVal int
	if err := DB.Table("json_generic_records").
		Select(`JSON_VALUE("obj", '$.count')`).
		Where(`"record_id" = ?`, rec.ID).
		Scan(&countVal).Error; err != nil {
		t.Fatalf("scan JSON_VALUE(obj.count) failed: %v", err)
	}
	if countVal != 2 {
		t.Fatalf("unexpected obj.count: %d", countVal)
	}

	// Append to array using JSON_TRANSFORM APPEND (JSONArray behavior)
	if err := DB.Model(&JsonGenericRecord{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("arr", gorm.Expr(`JSON_TRANSFORM("arr", APPEND '$' = ?)`, "c")).Error; err != nil {
		t.Fatalf("append with JSON_TRANSFORM(arr) failed: %v", err)
	}

	// Verify
	var rec2 JsonGenericRecord
	if err := DB.First(&rec2, `"record_id" = ?`, rec.ID).Error; err != nil {
		t.Fatalf("reload after append failed: %v", err)
	}
	var gotArr []string
	if b, err := json.Marshal(rec2.Arr); err != nil {
		t.Fatalf("marshal Arr failed: %v", err)
	} else if err := json.Unmarshal(b, &gotArr); err != nil {
		t.Fatalf("unmarshal Arr failed: %v", err)
	}
	if len(gotArr) != 3 || gotArr[0] != "a" || gotArr[1] != "b" || gotArr[2] != "c" {
		t.Fatalf("unexpected Arr: %#v", gotArr)
	}

	// DELETE
	var deleted []JsonGenericRecord
	if err := DB.
		Where(`"record_id" = ?`, rec.ID).
		Clauses(clause.Returning{Columns: []clause.Column{
			{Name: "record_id"}, {Name: "name"}, {Name: "doc"}, {Name: "obj"}, {Name: "arr"},
		}}).
		Delete(&deleted).Error; err != nil {
		t.Fatalf("delete returning failed: %v", err)
	}
	if len(deleted) != 1 || deleted[0].ID != rec.ID {
		t.Fatalf("unexpected deleted rows: %#v", deleted)
	}

	// verify gone
	var check JsonGenericRecord
	err := DB.First(&check, `"record_id" = ?`, rec.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}
}

func TestBasicCRUD_RawMessage(t *testing.T) {
	type RawRecord struct {
		ID            uint             `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name          string           `gorm:"column:name"`
		Properties    json.RawMessage  `gorm:"column:properties"`
		PropertiesPtr *json.RawMessage `gorm:"column:propertiesPtr"`
	}

	DB.Migrator().DropTable(&RawRecord{})
	if err := DB.AutoMigrate(&RawRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	rawMsg := json.RawMessage(`{"a":1,"b":"x"}`)
	// INSERT
	rec := RawRecord{
		Name:          "raw-json",
		Properties:    rawMsg,
		PropertiesPtr: &rawMsg,
	}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID to be set")
	}

	// UPDATE (with RETURNING)
	upatedRawMsg := json.RawMessage(`{"a":2,"c":true}`)
	var ret RawRecord
	if err := DB.
		Clauses(clause.Returning{
			Columns: []clause.Column{
				{Name: "record_id"},
				{Name: "name"},
				{Name: "properties"},
				{Name: "propertiesPtr"},
			},
		}).
		Model(&ret).
		Where("\"record_id\" = ?", rec.ID).
		Updates(map[string]any{
			"name":          "raw-json-upd",
			"properties":    upatedRawMsg,
			"propertiesPtr": &upatedRawMsg,
		}).Error; err != nil {
		t.Fatalf("update returning failed: %v", err)
	}
	if ret.ID != rec.ID ||
		ret.Name != "raw-json-upd" ||
		len(ret.Properties) == 0 ||
		ret.PropertiesPtr == nil || (ret.PropertiesPtr != nil && len(*ret.PropertiesPtr) == 0) {
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
				{Name: "propertiesPtr"},
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

func TestLargeJSON(t *testing.T) {
	// Mini stress test using large JSON payloads:
	// - doc: 1000 key/value pairs
	// - arr: 1000 numeric items
	// Exercise JSON_VALUE, JSON_EXISTS, JSON_TRANSFORM (SET/APPEND)
	type StressRec struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
		Arr datatypes.JSON `gorm:"column:arr"`
	}

	DB.Migrator().DropTable(&StressRec{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&StressRec{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// Build a big object with 1000 key/value pairs and an array with 1000 items
	bigObj := make(map[string]string, 1000)
	for i := 1; i <= 1000; i++ {
		k := fmt.Sprintf("k%03d", i)
		v := "v" + strings.Repeat("x", i)
		bigObj[k] = v
	}
	bdoc, err := json.Marshal(bigObj)
	if err != nil {
		t.Fatalf("marshal big obj failed: %v", err)
	}
	var bigArr []int
	for i := 0; i < 1000; i++ {
		bigArr = append(bigArr, i)
	}
	barr, err := json.Marshal(bigArr)
	if err != nil {
		t.Fatalf("marshal big arr failed: %v", err)
	}

	rec := StressRec{
		Doc: datatypes.JSON(bdoc),
		Arr: datatypes.JSON(barr),
	}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if rec.ID == 0 {
		t.Fatalf("expected ID set")
	}

	// JSON_VALUE on large object
	var got string
	if err := DB.Table("stress_recs").
		Select(`JSON_VALUE("doc",'$.k050')`).
		Where(`"record_id" = ?`, rec.ID).
		Scan(&got).Error; err != nil {
		t.Fatalf("JSON_VALUE(doc.k050) failed: %v", err)
	}
	expected := "v" + strings.Repeat("x", 50)
	if got != expected {
		t.Fatalf("unexpected JSON_VALUE(doc.k050): got %q want %q", got, expected)
	}

	// JSON_EXISTS on large object
	var cnt int64
	if err := DB.Model(&StressRec{}).
		Where(`JSON_EXISTS("doc",'$.k075')`).
		Count(&cnt).Error; err != nil {
		t.Fatalf("JSON_EXISTS(doc.k075) failed: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 row from JSON_EXISTS on k075, got %d", cnt)
	}

	// Update JSON document: add new key and update an existing one
	if err := DB.Model(&StressRec{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.k1001' = ?, SET '$.k050' = ?)`, "v1001", "v050-upd")).Error; err != nil {
		t.Fatalf("JSON_TRANSFORM(doc SET ...) failed: %v", err)
	}

	// Append to large array
	if err := DB.Model(&StressRec{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("arr", gorm.Expr(`JSON_TRANSFORM("arr", APPEND '$' = ?)`, 100)).Error; err != nil {
		t.Fatalf("JSON_TRANSFORM(arr APPEND) failed: %v", err)
	}

	// Verify updated keys via JSON_VALUE
	var v1001, v050 string
	if err := DB.Table("stress_recs").
		Select(`JSON_VALUE("doc",'$.k1001')`).
		Where(`"record_id" = ?`, rec.ID).
		Scan(&v1001).Error; err != nil {
		t.Fatalf("JSON_VALUE(doc.k101) failed: %v", err)
	}
	if err := DB.Table("stress_recs").
		Select(`JSON_VALUE("doc",'$.k050')`).
		Where(`"record_id" = ?`, rec.ID).
		Scan(&v050).Error; err != nil {
		t.Fatalf("JSON_VALUE(doc.k050) failed: %v", err)
	}
	if v1001 != "v1001" || v050 != "v050-upd" {
		t.Fatalf("unexpected updated values: k101=%q k050=%q", v1001, v050)
	}

	// Verify array length and last element
	var out StressRec
	if err := DB.First(&out, `"record_id" = ?`, rec.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	var arr []int
	if b, err := json.Marshal(out.Arr); err != nil {
		t.Fatalf("marshal arr failed: %v", err)
	} else if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatalf("unmarshal arr failed: %v", err)
	}
	if len(arr) != 1001 || arr[1000] != 100 {
		t.Fatalf("unexpected arr state: len=%d last=%d", len(arr), arr[len(arr)-1])
	}

	// Cleanup single row
	if err := DB.Where(`"record_id" = ?`, rec.ID).Delete(&StressRec{}).Error; err != nil {
		t.Fatalf("cleanup delete failed: %v", err)
	}

	// Check the deleted row is gone
	var check StressRec
	err = DB.First(&check, `"record_id" = ?`, rec.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected not found after delete, got: %v", err)
	}

	// Cleanup table
	DB.Migrator().DropTable(&StressRec{})
}

func TestJSONReturningMultipleRows(t *testing.T) {
	t.Skip("Skipping test due to issue #98")
	type RetRec struct {
		ID   uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc  datatypes.JSON `gorm:"column:doc"`
		Info datatypes.JSON `gorm:"column:info"`
	}

	DB.Migrator().DropTable(&RetRec{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&RetRec{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	ins := []RetRec{
		{Doc: datatypes.JSON([]byte(`{"k":1}`)), Info: datatypes.JSON([]byte(`[10]`))},
		{Doc: datatypes.JSON([]byte(`{"k":2}`)), Info: datatypes.JSON([]byte(`[20]`))},
		{Doc: datatypes.JSON([]byte(`{"k":3}`)), Info: datatypes.JSON([]byte(`[30]`))},
	}
	if err := DB.Create(&ins).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	var deleted []RetRec
	if err := DB.
		Clauses(clause.Returning{Columns: []clause.Column{
			{Name: "record_id"}, {Name: "doc"}, {Name: "info"},
		}}).
		Where(`"record_id" IN ?`, []uint{ins[0].ID, ins[1].ID, ins[2].ID}).
		Delete(&deleted).Error; err != nil {
		t.Fatalf("delete with returning failed: %v", err)
	}

	if len(deleted) != 3 {
		t.Fatalf("expected 3 returned rows from delete, got %d", len(deleted))
	}
	for _, r := range deleted {
		if r.ID == 0 {
			t.Fatalf("unexpected zero ID in returned row: %#v", r)
		}
		if len(r.Doc) == 0 {
			t.Fatalf("expected non-empty JSON in returned Doc for ID=%d", r.ID)
		}
		if len(r.Info) == 0 {
			t.Fatalf("expected non-empty JSON in returned Info for ID=%d", r.ID)
		}
	}

	var cnt int64
	if err := DB.Model(&RetRec{}).Count(&cnt).Error; err != nil {
		t.Fatalf("count after delete failed: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("expected table empty after delete, found %d rows", cnt)
	}
}

func TestJSONRootArray(t *testing.T) {
	type TransRec struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}

	DB.Migrator().DropTable(&TransRec{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&TransRec{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// insert a null JSON object as root
	rec := TransRec{Doc: datatypes.JSON([]byte(`{}`))}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// APPEND to non-array root should fail with ORA-40769: value not a JSON array
	err := DB.Model(&TransRec{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", APPEND '$' = ?)`, 1)).Error
	if err == nil {
		t.Fatalf("expected error appending to non-array root JSON, got nil")
	}

	// Set root to an empty array, then append
	if err := DB.Model(&TransRec{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("doc", gorm.Expr(`JSON_SERIALIZE(JSON_ARRAY() RETURNING CLOB)`)).Error; err != nil {
		t.Fatalf("set root to empty array failed: %v", err)
	}

	if err := DB.Model(&TransRec{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", APPEND '$' = ?, APPEND '$' = ?)`, 1, 2)).Error; err != nil {
		t.Fatalf("append after fixing root failed: %v", err)
	}

	var out TransRec
	if err := DB.First(&out, `"record_id" = ?`, rec.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	var arr []int
	if b, err := json.Marshal(out.Doc); err != nil {
		t.Fatalf("marshal doc failed: %v", err)
	} else if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatalf("unmarshal as array failed: %v", err)
	}
	if len(arr) != 2 || arr[0] != 1 || arr[1] != 2 {
		t.Fatalf("unexpected array content after appends: %#v", arr)
	}
}

func TestCustomJSON(t *testing.T) {
	type CustomJSONModel struct {
		Blah string       `gorm:"primaryKey"`
		Data AttributeMap `gorm:"type:json"`
	}

	type test struct {
		model any
		fn    func(model any) error
	}
	tests := map[string]test{
		"Single": {
			model: []CustomJSONModel{
				{
					Blah: "1",
					Data: AttributeMap{"Data": strings.Repeat("X", 32768)},
				},
			},
			fn: func(model any) error {
				return DB.Clauses(clause.OnConflict{
					UpdateAll: true,
				}).CreateInBatches(model, 1000).Error
			},
		},
		"SingleBatch": {
			model: []CustomJSONModel{
				{
					Blah: "1",
					Data: AttributeMap{"Data": strings.Repeat("X", 32768)},
				},
				{
					Blah: "2",
					Data: AttributeMap{"Data": strings.Repeat("Y", 3)},
				},
			},
			fn: func(model any) error {
				return DB.Clauses(clause.OnConflict{
					UpdateAll: true,
				}).CreateInBatches(model, 1000).Error
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			DB.Migrator().DropTable(&CustomJSONModel{})
			if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&CustomJSONModel{}); err != nil {
				t.Fatalf("migrate failed: %v", err)
			}
			err := tc.fn(tc.model)
			if err != nil {
				t.Fatalf("Failed to create CLOB record with ON CONFLICT: %v", err)
			}
		})
	}
}

func scanBytes(src interface{}) ([]byte, bool) {
	if stringer, ok := src.(fmt.Stringer); ok {
		return []byte(stringer.String()), true
	}
	bytes, ok := src.([]byte)
	if !ok {
		return nil, false
	}
	return bytes, true
}

type AttributeMap map[string]interface{}

func (a AttributeMap) Value() (driver.Value, error) {
	attrs := a
	if attrs == nil {
		attrs = AttributeMap{}
	}
	value, err := json.Marshal(attrs)
	return value, err
}

func (a *AttributeMap) Scan(src interface{}) error {
	bytes, ok := scanBytes(src)
	if !ok {
		return fmt.Errorf("failed to scan attribute map")
	}
	var raw interface{}
	err := json.Unmarshal(bytes, &raw)
	if err != nil {
		return err
	}

	if raw == nil {
		*a = map[string]interface{}{}
		return nil
	}
	*a, ok = raw.(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to convert attribute map from json")
	}
	return nil
}
