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
	"gorm.io/gorm/utils/tests"
)

func TestCustomizeColumn(t *testing.T) {
	type CustomizeColumn struct {
		ID   int64      `gorm:"column:mapped_id; primary_key:yes"`
		Name string     `gorm:"column:mapped_name"`
		Date *time.Time `gorm:"column:mapped_time"`
	}

	DB.Migrator().DropTable(&CustomizeColumn{})
	DB.AutoMigrate(&CustomizeColumn{})

	expected := "foo"
	now := time.Now()
	cc := CustomizeColumn{ID: 666, Name: expected, Date: &now}

	if count := DB.Create(&cc).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}

	var cc1 CustomizeColumn
	DB.First(&cc1, "\"mapped_name\" = ?", "foo")

	if cc1.Name != expected {
		t.Errorf("Failed to query CustomizeColumn")
	}

	cc.Name = "bar"
	DB.Save(&cc)

	var cc2 CustomizeColumn
	DB.First(&cc2, "\"mapped_id\" = ?", 666)
	if cc2.Name != "bar" {
		t.Errorf("Failed to query CustomizeColumn")
	}
}

func TestCustomColumnAndIgnoredFieldClash(t *testing.T) {
	// Make sure an ignored field does not interfere with another field's custom
	// column name that matches the ignored field.
	type CustomColumnAndIgnoredFieldClash struct {
		Body    string `gorm:"-"`
		RawBody string `gorm:"column:body"`
	}

	DB.Migrator().DropTable(&CustomColumnAndIgnoredFieldClash{})

	if err := DB.AutoMigrate(&CustomColumnAndIgnoredFieldClash{}); err != nil {
		t.Errorf("Should not raise error: %v", err)
	}

	if DB.Migrator().HasColumn(&CustomColumnAndIgnoredFieldClash{}, "Body") {
		t.Errorf("Body should not be created")
	}
}

func TestCustomizeField(t *testing.T) {
	type CustomizeFieldStruct struct {
		gorm.Model
		Name                    string
		FieldAllowCreate        string `gorm:"<-:create"`
		FieldAllowUpdate        string `gorm:"<-:update"`
		FieldAllowSave          string `gorm:"<-"`
		FieldAllowSave2         string `gorm:"<-:create,update"`
		FieldAllowSave3         string `gorm:"->:false;<-:create"`
		FieldReadonly           string `gorm:"->"`
		FieldIgnore             string `gorm:"-"`
		AutoUnixCreateTime      int32  `gorm:"autocreatetime"`
		AutoUnixMilliCreateTime int    `gorm:"autocreatetime:milli"`
		AutoUnixNanoCreateTime  int64  `gorm:"autocreatetime:nano"`
		AutoUnixUpdateTime      uint32 `gorm:"autoupdatetime"`
		AutoUnixMilliUpdateTime int    `gorm:"autoupdatetime:milli"`
		AutoUnixNanoUpdateTime  uint64 `gorm:"autoupdatetime:nano"`
	}

	DB.Migrator().DropTable(&CustomizeFieldStruct{})

	if err := DB.AutoMigrate(&CustomizeFieldStruct{}); err != nil {
		t.Errorf("Failed to migrate, got error: %v", err)
	}

	if DB.Migrator().HasColumn(&CustomizeFieldStruct{}, "FieldIgnore") {
		t.Errorf("FieldIgnore should not be created")
	}

	if DB.Migrator().HasColumn(&CustomizeFieldStruct{}, "field_ignore") {
		t.Errorf("FieldIgnore should not be created")
	}

	generateStruct := func(name string) *CustomizeFieldStruct {
		return &CustomizeFieldStruct{
			Name:             name,
			FieldAllowCreate: name + "_allow_create",
			FieldAllowUpdate: name + "_allow_update",
			FieldAllowSave:   name + "_allow_save",
			FieldAllowSave2:  name + "_allow_save2",
			FieldAllowSave3:  name + "_allow_save3",
			FieldReadonly:    name + "_allow_readonly",
			FieldIgnore:      name + "_allow_ignore",
		}
	}

	create := generateStruct("create")
	DB.Create(&create)

	var result CustomizeFieldStruct
	DB.Find(&result, "\"name\" = ?", "create")

	tests.AssertObjEqual(t, result, create, "Name", "FieldAllowCreate", "FieldAllowSave", "FieldAllowSave2")

	if result.FieldAllowUpdate != "" || result.FieldReadonly != "" || result.FieldIgnore != "" || result.FieldAllowSave3 != "" {
		t.Fatalf("invalid result: %#v", result)
	}

	if int(result.AutoUnixCreateTime) != int(result.AutoUnixUpdateTime) || result.AutoUnixCreateTime == 0 {
		t.Fatalf("invalid create/update unix time: %#v", result)
	}

	if int(result.AutoUnixMilliCreateTime) != int(result.AutoUnixMilliUpdateTime) || result.AutoUnixMilliCreateTime == 0 || int(result.AutoUnixMilliCreateTime)/int(result.AutoUnixCreateTime) < 1e3 {
		t.Fatalf("invalid create/update unix milli time: %#v", result)
	}

	if int(result.AutoUnixNanoCreateTime) != int(result.AutoUnixNanoUpdateTime) || result.AutoUnixNanoCreateTime == 0 || int(result.AutoUnixNanoCreateTime)/int(result.AutoUnixCreateTime) < 1e6 {
		t.Fatalf("invalid create/update unix nano time: %#v", result)
	}

	result.FieldAllowUpdate = "field_allow_update_updated"
	result.FieldReadonly = "field_readonly_updated"
	result.FieldIgnore = "field_ignore_updated"
	DB.Save(&result)

	var result2 CustomizeFieldStruct
	DB.Find(&result2, "\"name\" = ?", "create")

	if result2.FieldAllowUpdate != result.FieldAllowUpdate || result2.FieldReadonly != "" || result2.FieldIgnore != "" {
		t.Fatalf("invalid updated result: %#v", result2)
	}

	if err := DB.Where(CustomizeFieldStruct{Name: create.Name, FieldReadonly: create.FieldReadonly, FieldIgnore: create.FieldIgnore}).First(&CustomizeFieldStruct{}).Error; err == nil {
		t.Fatalf("Should failed to find result")
	}

	if err := DB.Table("customize_field_structs").Where("1 = 1").UpdateColumn("field_readonly", "readonly").Error; err != nil {
		t.Fatalf("failed to update field_readonly column")
	}

	if err := DB.Where(CustomizeFieldStruct{Name: create.Name, FieldReadonly: "readonly", FieldIgnore: create.FieldIgnore}).First(&CustomizeFieldStruct{}).Error; err != nil {
		t.Fatalf("Should find result")
	}

	var result3 CustomizeFieldStruct
	DB.Find(&result3, "\"name\" = ?", "create")

	if result3.FieldReadonly != "readonly" {
		t.Fatalf("invalid updated result: %#v", result3)
	}

	var result4 CustomizeFieldStruct
	if err := DB.First(&result4, "\"field_allow_save3\" = ?", create.FieldAllowSave3).Error; err != nil {
		t.Fatalf("failed to query with inserted field, got error %v", err)
	}

	tests.AssertEqual(t, result3, result4)

	createWithDefaultTime := generateStruct("create_with_default_time")
	createWithDefaultTime.AutoUnixCreateTime = 100
	createWithDefaultTime.AutoUnixUpdateTime = 100
	createWithDefaultTime.AutoUnixMilliCreateTime = 100
	createWithDefaultTime.AutoUnixMilliUpdateTime = 100
	createWithDefaultTime.AutoUnixNanoCreateTime = 100
	createWithDefaultTime.AutoUnixNanoUpdateTime = 100
	DB.Create(&createWithDefaultTime)

	var createWithDefaultTimeResult CustomizeFieldStruct
	DB.Find(&createWithDefaultTimeResult, "\"name\" = ?", createWithDefaultTime.Name)

	if int(createWithDefaultTimeResult.AutoUnixCreateTime) != int(createWithDefaultTimeResult.AutoUnixUpdateTime) || createWithDefaultTimeResult.AutoUnixCreateTime != 100 {
		t.Fatalf("invalid create/update unix time: %#v", createWithDefaultTimeResult)
	}

	if int(createWithDefaultTimeResult.AutoUnixMilliCreateTime) != int(createWithDefaultTimeResult.AutoUnixMilliUpdateTime) || createWithDefaultTimeResult.AutoUnixMilliCreateTime != 100 {
		t.Fatalf("invalid create/update unix milli time: %#v", createWithDefaultTimeResult)
	}

	if int(createWithDefaultTimeResult.AutoUnixNanoCreateTime) != int(createWithDefaultTimeResult.AutoUnixNanoUpdateTime) || createWithDefaultTimeResult.AutoUnixNanoCreateTime != 100 {
		t.Fatalf("invalid create/update unix nano time: %#v", createWithDefaultTimeResult)
	}
}
