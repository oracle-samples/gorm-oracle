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
	"strconv"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestJSONKeys(t *testing.T) {
	type PathEscapingRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}
	DB.Migrator().DropTable(&PathEscapingRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&PathEscapingRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// Keys containing dots, spaces, quotes; and an array
	weirdJSON := datatypes.JSON([]byte(`{"weird.key":{"sp ace":{"quote\"":"val"}}, "arr":[10,20,30]}`))
	record := PathEscapingRecord{Doc: weirdJSON}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Escaped path read should succeed
	var escapedValue string
	if err := DB.Model(&PathEscapingRecord{}).
		Select(`JSON_VALUE("doc",'$."weird.key"."sp ace"."quote\""' )`).
		Where(`"record_id" = ?`, record.ID).
		Scan(&escapedValue).Error; err != nil {
		t.Fatalf("escaped path read failed: %v", err)
	}
	if escapedValue != "val" {
		t.Fatalf("unexpected escaped value: %q", escapedValue)
	}

	// REMOVE nested key and deep SET array index
	if err := DB.Model(&PathEscapingRecord{}).
		Where(`"record_id" = ?`, record.ID).
		Update("doc", gorm.Expr(
			`JSON_TRANSFORM("doc", REMOVE '$."weird.key"."sp ace"', SET '$.arr[1]' = ?)`, 99,
		)).Error; err != nil {
		t.Fatalf("JSON_TRANSFORM remove/set failed: %v", err)
	}

	// Verify array content
	var reloaded PathEscapingRecord
	if err := DB.First(&reloaded, `"record_id" = ?`, record.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	var docMap map[string]any
	if b, err := json.Marshal(reloaded.Doc); err != nil {
		t.Fatalf("marshal doc failed: %v", err)
	} else if err := json.Unmarshal(b, &docMap); err != nil {
		t.Fatalf("unmarshal doc failed: %v", err)
	}
	arr, ok := docMap["arr"].([]any)
	if !ok || len(arr) != 3 || arr[0] != float64(10) || arr[1] != float64(99) || arr[2] != float64(30) {
		t.Fatalf("unexpected array after transform: %#v", docMap["arr"])
	}
	// Confirm removed nested key is gone
	if weirdKeyMap, ok := docMap["weird.key"].(map[string]any); ok {
		if _, exists := weirdKeyMap["sp ace"]; exists {
			t.Fatalf("expected removed nested key, still present: %#v", weirdKeyMap)
		}
	}
}

func TestJSONQuery(t *testing.T) {
	type QueryScanRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}
	DB.Migrator().DropTable(&QueryScanRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&QueryScanRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	insertDoc := datatypes.JSON([]byte(`{"obj":{"x":42,"y":"s"}, "arr":[{"k":1},{"k":2},{"k":3}]}`))
	inserted := QueryScanRecord{Doc: insertDoc}
	if err := DB.Create(&inserted).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// JSON_QUERY -> scan into string
	var rsJson string
	if err := DB.Model(&QueryScanRecord{}).
		Select(`JSON_SERIALIZE(JSON_QUERY("doc",'$.obj') RETURNING CLOB)`).
		Where(`"record_id" = ?`, inserted.ID).
		Scan(&rsJson).Error; err != nil {
		t.Fatalf("json_query scan string failed: %v", err)
	}
	var rsObj map[string]any
	if err := json.Unmarshal([]byte(rsJson), &rsObj); err != nil {
		t.Fatalf("unmarshal frag json string failed: %v", err)
	}
	if rsObj["x"] != float64(42) || rsObj["y"] != "s" {
		t.Fatalf("unexpected obj fragment: %#v", rsObj)
	}

	// JSON_QUERY with WITH WRAPPER slice
	var sliceJSON string
	if err := DB.Model(&QueryScanRecord{}).
		Select(`JSON_SERIALIZE(JSON_QUERY("doc",'$.arr[0 to 1]' WITH WRAPPER) RETURNING CLOB)`).
		Where(`"record_id" = ?`, inserted.ID).
		Scan(&sliceJSON).Error; err != nil {
		t.Fatalf("json_query slice failed: %v", err)
	}
	var slicedArr []map[string]any
	if err := json.Unmarshal([]byte(sliceJSON), &slicedArr); err != nil {
		t.Fatalf("unmarshal slice failed: %v", err)
	}
	if len(slicedArr) != 2 || slicedArr[0]["k"] != float64(1) || slicedArr[1]["k"] != float64(2) {
		t.Fatalf("unexpected slice: %#v", slicedArr)
	}

	// JSON_VALUE number -> int
	var numberX int
	if err := DB.Model(&QueryScanRecord{}).
		Select(`JSON_VALUE("doc",'$.obj.x' RETURNING NUMBER)`).
		Where(`"record_id" = ?`, inserted.ID).
		Scan(&numberX).Error; err != nil {
		t.Fatalf("json_value number failed: %v", err)
	}
	if numberX != 42 {
		t.Fatalf("unexpected x: %d", numberX)
	}
}

func TestJSONReturning(t *testing.T) {
	type ReturningRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}
	DB.Migrator().DropTable(&ReturningRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&ReturningRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	record := ReturningRecord{Doc: datatypes.JSON([]byte(`{"n":1}`))}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// RETURNING into struct field (datatypes.JSON)
	type ReturnedDoc struct{ Doc datatypes.JSON }
	var returned ReturnedDoc
	if err := DB.
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "doc"}}}).
		Model(&ReturningRecord{}).
		Where(`"record_id" = ?`, record.ID).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.n' = 2)`)).
		Scan(&returned).Error; err != nil {
		t.Fatalf("returning into datatypes.JSON failed: %v", err)
	}
	if len(returned.Doc) == 0 {
		t.Fatalf("empty doc from returning scan into datatypes.JSON")
	}
	// Verify
	var docMap map[string]any
	if b, err := json.Marshal(returned.Doc); err != nil {
		t.Fatalf("marshal returned Doc failed: %v", err)
	} else if err := json.Unmarshal(b, &docMap); err != nil {
		t.Fatalf("unmarshal returned Doc failed: %v", err)
	}
	if nv, ok := docMap["n"].(float64); !ok || nv != 2 {
		t.Fatalf("unexpected returned doc.n: %#v", docMap["n"])
	}
	// Disable due to issue #99
	// // RETURNING into json.RawMessage via struct wrapper
	// type ReturnedRaw struct{ Doc json.RawMessage }
	// var retRaw ReturnedRaw
	// if err := DB.
	// 	Clauses(clause.Returning{Columns: []clause.Column{{Name: "doc"}}}).
	// 	Model(&ReturningRecord{}).
	// 	Where(`"record_id" = ?`, record.ID).
	// 	Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.n' = 3)`)).
	// 	Scan(&retRaw).Error; err != nil {
	// 	t.Fatalf("returning into RawMessage failed: %v", err)
	// }
	// if len(retRaw.Doc) == 0 {
	// 	t.Fatalf("empty RawMessage from returning scan")
	// }
	// // Convert datatypes.JSON to json.RawMessage
	// rawDoc := json.RawMessage(retRaw.Doc)
	// // Verify returned raw content has n=3
	// var rawMap map[string]any
	// if err := json.Unmarshal(rawDoc, &rawMap); err != nil {
	// 	t.Fatalf("unmarshal returned RawMessage failed: %v", err)
	// }
	// if nv, ok := rawMap["n"].(float64); !ok || nv != 3 {
	// 	t.Fatalf("unexpected returned raw doc.n: %#v", rawMap["n"])
	// }

	// Verify latest value
	var nVal int
	if err := DB.Model(&ReturningRecord{}).
		Select(`JSON_VALUE("doc",'$.n' RETURNING NUMBER)`).
		Where(`"record_id" = ?`, record.ID).
		Scan(&nVal).Error; err != nil {
		t.Fatalf("json_value verify failed: %v", err)
	}
	if nVal != 2 {
		t.Fatalf("unexpected n: %d", nVal)
	}
}

func TestJSONTxRollback(t *testing.T) {
	type CounterRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}
	DB.Migrator().DropTable(&CounterRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&CounterRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	record := CounterRecord{Doc: datatypes.JSON([]byte(`{"cnt":1}`))}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	tx := DB.Session(&gorm.Session{PrepareStmt: true}).Begin()
	if err := tx.Error; err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}

	// Valid transform inside tx
	if err := tx.Model(&CounterRecord{}).
		Where(`"record_id" = ?`, record.ID).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.cnt' = ?)`, 2)).Error; err != nil {
		tx.Rollback()
		t.Fatalf("tx update set failed: %v", err)
	}

	// Force an error to trigger rollback (invalid path syntax)
	txErr := tx.Model(&CounterRecord{}).
		Where(`"record_id" = ?`, record.ID).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.[bad' = 0)`)).Error
	if txErr == nil {
		tx.Rollback()
		t.Fatalf("expected JSON path syntax error inside tx, got nil")
	}
	_ = tx.Rollback()

	// Verify original row was not modified by tx (rolled back)
	var cntAfter int
	if err := DB.Model(&CounterRecord{}).
		Select(`JSON_VALUE("doc",'$.cnt' RETURNING NUMBER)`).
		Where(`"record_id" = ?`, record.ID).
		Scan(&cntAfter).Error; err != nil {
		t.Fatalf("verify after rollback failed: %v", err)
	}
	if cntAfter != 1 {
		t.Fatalf("expected cnt=1 after rollback, got %d", cntAfter)
	}
}

func TestJSONCreateInBatchesAndBulkUpdate(t *testing.T) {
	type BatchJSONRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}
	DB.Migrator().DropTable(&BatchJSONRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&BatchJSONRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	var batch []BatchJSONRecord
	for i := 1; i <= 50; i++ {
		batch = append(batch, BatchJSONRecord{Doc: datatypes.JSON([]byte(`{"a":` + strconv.Itoa(i) + `}`))})
	}
	if err := DB.CreateInBatches(&batch, 100).Error; err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}
	var ids []uint
	for _, r := range batch {
		ids = append(ids, r.ID)
	}

	// Bulk update: add key "b":"x" for all rows
	if err := DB.Model(&BatchJSONRecord{}).
		Where(`"record_id" IN ?`, ids).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.b' = ?)`, "x")).Error; err != nil {
		t.Fatalf("bulk update failed: %v", err)
	}

	// Verify all rows updated
	var updatedCount int64
	if err := DB.Model(&BatchJSONRecord{}).
		Where(`JSON_VALUE("doc",'$.b') = ?`, "x").
		Count(&updatedCount).Error; err != nil {
		t.Fatalf("count verify failed: %v", err)
	}
	if updatedCount != 50 {
		t.Fatalf("expected 5 rows with b='x', got %d", updatedCount)
	}

	// Value verification: reload and check each row
	for i, id := range ids {
		var row BatchJSONRecord
		if err := DB.First(&row, `"record_id" = ?`, id).Error; err != nil {
			t.Fatalf("reload failed for id %d: %v", id, err)
		}
		var m map[string]any
		if b, err := json.Marshal(row.Doc); err != nil {
			t.Fatalf("marshal failed for id %d: %v", id, err)
		} else if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("unmarshal failed for id %d: %v", id, err)
		}
		expectedA := float64(i + 1)
		if m["a"] != expectedA {
			t.Fatalf("unexpected value for key 'a' in id %d: got %#v, want %v", id, m["a"], expectedA)
		}
		if m["b"] != "x" {
			t.Fatalf("unexpected value for key 'b' in id %d: got %#v, want %q", id, m["b"], "x")
		}
	}
}

func TestUnicodeJSON(t *testing.T) {
	type UnicodeJSONRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}
	DB.Migrator().DropTable(&UnicodeJSONRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&UnicodeJSONRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// Insert
	unicodeValue := "😃中文𝄞"
	record := UnicodeJSONRecord{Doc: datatypes.JSON([]byte(`{"s":"` + unicodeValue + `"}`))}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}
	// Verify
	var gotStr string
	if err := DB.Model(&UnicodeJSONRecord{}).
		Select(`JSON_VALUE("doc",'$.s')`).
		Where(`"record_id" = ?`, record.ID).
		Scan(&gotStr).Error; err != nil {
		t.Fatalf("json_value unicode failed: %v", err)
	}
	if gotStr != unicodeValue {
		t.Fatalf("unexpected unicode value: %q", gotStr)
	}

	var reloaded UnicodeJSONRecord
	if err := DB.First(&reloaded, `"record_id" = ?`, record.ID).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	var obj map[string]any
	if b, err := json.Marshal(reloaded.Doc); err != nil {
		t.Fatalf("marshal failed: %v", err)
	} else if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if obj["s"] != unicodeValue {
		t.Fatalf("unexpected round-trip unicode: %#v", obj["s"])
	}
}

func TestJSONStrict(t *testing.T) {
	type StrictJSONRecord struct {
		ID      uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		DocJson datatypes.JSON `gorm:"type:CLOB;column:doc;check:doc_is_json_strict,doc IS JSON (STRICT)"`
	}
	DB.Migrator().DropTable(&StrictJSONRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&StrictJSONRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	record := StrictJSONRecord{DocJson: datatypes.JSON([]byte(`{'field': 1}`))}
	if err := DB.Create(&record).Error; err == nil {
		t.Fatalf("should raise error in strict mode")
	}
}

func TestJSONLAX(t *testing.T) {
	type LaxJSONRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"type:CLOB;column:doc;check:doc_is_json_lax,doc IS JSON(LAX)"`
	}
	DB.Migrator().DropTable(&LaxJSONRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&LaxJSONRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	record := LaxJSONRecord{Doc: datatypes.JSON([]byte(`{'field': 1}`))}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("insert failed in lax mode: %v", err)
	}
}

func TestJSONTxErrorRecovery(t *testing.T) {
	type RecoveryJSONRecord struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}
	DB.Migrator().DropTable(&RecoveryJSONRecord{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&RecoveryJSONRecord{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	record := RecoveryJSONRecord{Doc: datatypes.JSON([]byte(`{"a":1}`))}
	if err := DB.Create(&record).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// invalid json
	if err := DB.Model(&RecoveryJSONRecord{}).
		Where(`"record_id" = ?`, record.ID).
		Update("doc", datatypes.JSON([]byte(`{not-json}`))).Error; err == nil {
		t.Fatalf("expected malformed JSON update to fail")
	}

	// Following valid update should succeed
	if err := DB.Model(&RecoveryJSONRecord{}).
		Where(`"record_id" = ?`, record.ID).
		Update("doc", gorm.Expr(`JSON_TRANSFORM("doc", SET '$.a' = 2)`)).Error; err != nil {
		t.Fatalf("subsequent valid update failed: %v", err)
	}

	// Verify
	var aVal int
	if err := DB.Model(&RecoveryJSONRecord{}).
		Select(`JSON_VALUE("doc",'$.a' RETURNING NUMBER)`).
		Where(`"record_id" = ?`, record.ID).
		Scan(&aVal).Error; err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if aVal != 2 {
		t.Fatalf("unexpected a after recovery: %d", aVal)
	}
}

func TestNullJSON(t *testing.T) {
	type NullJson struct {
		ID     uint            `gorm:"primaryKey;autoIncrement;column:record_id"`
		Name   string          `gorm:"column:name"`
		Doc    datatypes.JSON  `gorm:"column:doc"`
		DocPtr *datatypes.JSON `gorm:"column:doc_ptr"`
	}

	DB.Migrator().DropTable(&NullJson{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&NullJson{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	jsonEmptyObject := datatypes.JSON([]byte(`{}`))
	jsonExplicitNull := datatypes.JSON([]byte(`null`))
	jsonNonNilPointer := datatypes.JSON([]byte(`{"a":1}`))

	records := []NullJson{
		{Name: "empty-object", Doc: jsonEmptyObject, DocPtr: nil},
		{Name: "json-null", Doc: jsonExplicitNull, DocPtr: nil},
		// Doc is DB NULL (nil), DocPtr is non-nil JSON
		{Name: "db-null", Doc: nil, DocPtr: &jsonNonNilPointer},
	}
	if err := DB.Create(&records).Error; err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	var rows []NullJson
	if err := DB.Order(`"name"`).Find(&rows).Error; err != nil {
		t.Fatalf("read back failed: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}

	byName := map[string]NullJson{}
	for _, row := range rows {
		byName[row.Name] = row
	}

	// 1) verify empty-object
	{
		row := byName["empty-object"]
		var objMap map[string]any
		if b, err := json.Marshal(row.Doc); err != nil {
			t.Fatalf("marshal empty-object failed: %v", err)
		} else if err := json.Unmarshal(b, &objMap); err != nil {
			t.Fatalf("unmarshal empty-object failed: %v", err)
		}
		if len(objMap) != 0 {
			t.Fatalf("expected empty map for empty-object, got: %#v", objMap)
		}
	}

	// 2) verify json-null
	{
		row := byName["json-null"]
		var decoded any
		if b, err := json.Marshal(row.Doc); err != nil {
			t.Fatalf("marshal json-null failed: %v", err)
		} else if err := json.Unmarshal(b, &decoded); err != nil {
			t.Fatalf("unmarshal json-null failed: %v", err)
		}
		if decoded != nil {
			t.Fatalf("expected nil value for explicit JSON null, got: %#v", decoded)
		}
	}

	// 3) verify db-null
	{
		row := byName["db-null"]
		if len(row.Doc) != 0 {
			t.Fatalf("expected DB NULL for Doc (len==0), got len=%d", len(row.Doc))
		}
		if row.DocPtr == nil || (row.DocPtr != nil && len(*row.DocPtr) == 0) {
			t.Fatalf("expected non-nil non-empty DocPtr, got: %#v", row.DocPtr)
		}
	}

	// JSON_VALUE on missing property should return NULL
	var count int64
	if err := DB.Model(&NullJson{}).
		Where(`JSON_VALUE("doc",'$.missing') IS NOT NULL`).
		Count(&count).Error; err != nil {
		t.Fatalf("JSON_VALUE missing test failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows where missing property is NOT NULL, got %d", count)
	}
}

func TestJSONNegative(t *testing.T) {
	type ErrRec struct {
		ID  uint           `gorm:"primaryKey;autoIncrement;column:record_id"`
		Doc datatypes.JSON `gorm:"column:doc"`
	}

	DB.Migrator().DropTable(&ErrRec{})
	if err := DB.Set("gorm:table_options", "TABLESPACE SYSAUX").AutoMigrate(&ErrRec{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	rec := ErrRec{Doc: datatypes.JSON([]byte(`{"a":1}`))}
	if err := DB.Create(&rec).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Invalid JSON path syntax
	var cnt int64
	err := DB.Model(&ErrRec{}).
		Where(`JSON_VALUE("doc",'$.[') IS NOT NULL`).
		Count(&cnt).Error
	if err == nil {
		t.Fatalf("expected error from invalid JSON path, got nil")
	}

	// Malformed JSON text
	if err := DB.Model(&ErrRec{}).
		Where(`"record_id" = ?`, rec.ID).
		Update("doc", datatypes.JSON([]byte(`{not-json}`))).Error; err == nil {
		t.Fatalf("update malformed JSON text should fail")
	}
	err = DB.Model(&ErrRec{}).
		Where(`JSON_VALUE("doc",'$.a') = ?`, 1).
		Count(&cnt).Error
	if err != nil {
		t.Fatalf("Query after failed update should still work, got error: %v", err)
	}
	if cnt != 1 {
		t.Fatalf("expected 1 row after failed update, got %d", cnt)
	}
}
