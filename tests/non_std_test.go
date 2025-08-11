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
)

type Animal struct {
	Counter    uint64 `gorm:"primary_key:yes"`
	Name       string `gorm:"DEFAULT:'galeone'"`
	From       string // test reserved sql keyword as field name
	Age        *time.Time
	unexported string // unexported value
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func TestNonStdPrimaryKeyAndDefaultValues(t *testing.T) {
	DB.Migrator().DropTable(&Animal{})
	if err := DB.AutoMigrate(&Animal{}); err != nil {
		t.Fatalf("no error should happen when migrate but got %v", err)
	}

	animal := Animal{Name: "Ferdinand"}
	DB.Save(&animal)
	updatedAt1 := animal.UpdatedAt

	DB.Save(&animal).Update("name", "Francis")
	if updatedAt1.Format(time.RFC3339Nano) == animal.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("UpdatedAt should be updated")
	}

	var animals []Animal
	DB.Find(&animals)
	if count := DB.Model(Animal{}).Where("1=1").Update("CreatedAt", time.Now().Add(2*time.Hour)).RowsAffected; count != int64(len(animals)) {
		t.Error("RowsAffected should be correct when do batch update")
	}

	animal = Animal{From: "somewhere"}              // No name fields, should be filled with the default value (galeone)
	DB.Save(&animal).Update("From", "a nice place") // The name field should be untouched
	DB.First(&animal, animal.Counter)
	if animal.Name != "galeone" {
		t.Errorf("Name fields shouldn't be changed if untouched, but got %v", animal.Name)
	}

	// When changing a field with a default value, the change must occur
	animal.Name = "amazing horse"
	DB.Save(&animal)
	DB.First(&animal, animal.Counter)
	if animal.Name != "amazing horse" {
		t.Errorf("Update a filed with a default value should occur. But got %v\n", animal.Name)
	}

	// When changing a field with a default value with blank value
	animal.Name = ""
	DB.Save(&animal)
	DB.First(&animal, animal.Counter)
	if animal.Name != "" {
		t.Errorf("Update a filed to blank with a default value should occur. But got %v\n", animal.Name)
	}
}
