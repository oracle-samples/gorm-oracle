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
	"errors"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func TestDialectorWithErrorTranslatorSupport(t *testing.T) {
	// it shouldn't translate error when the TranslateError flag is false
	translatedErr := errors.New("translated error")
	untranslatedErr := errors.New("some random error")
	db, _ := gorm.Open(tests.DummyDialector{TranslatedErr: translatedErr})

	err := db.AddError(untranslatedErr)
	if !errors.Is(err, untranslatedErr) {
		t.Fatalf("expected err: %v got err: %v", untranslatedErr, err)
	}

	// it should translate error when the TranslateError flag is true
	db, _ = gorm.Open(tests.DummyDialector{TranslatedErr: translatedErr}, &gorm.Config{TranslateError: true})

	err = db.AddError(untranslatedErr)
	if !errors.Is(err, translatedErr) {
		t.Fatalf("expected err: %v got err: %v", translatedErr, err)
	}
}

func TestSupportedDialectorWithErrDuplicatedKey(t *testing.T) {
	type City struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	db, err := OpenTestConnection(&gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "mysql": true, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		return
	}

	DB.Migrator().DropTable(&City{})

	if err = db.AutoMigrate(&City{}); err != nil {
		t.Fatalf("failed to migrate cities table, got error: %v", err)
	}

	err = db.Create(&City{Name: "Kabul"}).Error
	if err != nil {
		t.Fatalf("failed to create record: %v", err)
	}

	err = db.Create(&City{Name: "Kabul"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrDuplicatedKey, err)
	}
}

func TestSupportedDialectorWithErrForeignKeyViolated(t *testing.T) {
	type City struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	type Museum struct {
		gorm.Model
		Name   string `gorm:"unique"`
		CityID uint
		City   City `gorm:"Constraint:OnUpdate:CASCADE,OnDelete:CASCADE;FOREIGNKEY:CityID;References:ID"`
	}

	db, err := OpenTestConnection(&gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "mysql": true, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		return
	}

	DB.Migrator().DropTable(&City{}, &Museum{})

	if err = db.AutoMigrate(&City{}, &Museum{}); err != nil {
		t.Fatalf("failed to migrate countries & cities tables, got error: %v", err)
	}

	city := City{Name: "Amsterdam"}

	err = db.Create(&city).Error
	if err != nil {
		t.Fatalf("failed to create city: %v", err)
	}

	err = db.Create(&Museum{Name: "Eye Filmmuseum", CityID: city.ID}).Error
	if err != nil {
		t.Fatalf("failed to create museum: %v", err)
	}

	err = db.Create(&Museum{Name: "Dungeon", CityID: 123}).Error
	if !errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("expected err: %v got err: %v", gorm.ErrForeignKeyViolated, err)
	}
}

func TestDialectorWithErrorTranslatorNegativeCases(t *testing.T) {
	translatedErr := errors.New("translated error")
	untranslatedErr := errors.New("some random error")

	db, _ := gorm.Open(tests.DummyDialector{TranslatedErr: translatedErr})
	err := db.AddError(nil)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	db, _ = gorm.Open(tests.DummyDialector{TranslatedErr: nil}, &gorm.Config{TranslateError: true})
	err = db.AddError(untranslatedErr)
	if err != nil {
    	t.Fatalf("expected nil when TranslatedErr is nil, got %v", err)
	}	

	db, _ = gorm.Open(tests.DummyDialector{TranslatedErr: translatedErr}, &gorm.Config{TranslateError: true})
	err = db.AddError(translatedErr)
	if !errors.Is(err, translatedErr) {
		t.Fatalf("expected translatedErr unchanged, got %v", err)
	}
}

func TestSupportedDialectorWithErrDuplicatedKeyNegative(t *testing.T) {
	type City struct {
		gorm.Model
		Name string `gorm:"unique"`
		Code string `gorm:"unique"`
	}

	db, err := OpenTestConnection(&gorm.Config{TranslateError: false})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "mysql": true, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		t.Logf("skipping test for unsupported dialector: %s", db.Dialector.Name())
		return
	}

	DB.Migrator().DropTable(&City{})
	if err = db.AutoMigrate(&City{}); err != nil {
		t.Fatalf("failed to migrate cities table, got error: %v", err)
	}

	if err := db.Create(&City{Name: "Kabul", Code: "KB"}).Error; err != nil {
		t.Fatalf("failed to create record: %v", err)
	}

	err = db.Create(&City{Name: "Kabul", Code: "KB"}).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected raw db error (no translation), got: %v", err)
	}

	db, err = OpenTestConnection(&gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}
	DB.Migrator().DropTable(&City{})
	if err = db.AutoMigrate(&City{}); err != nil {
		t.Fatalf("failed to migrate cities table, got error: %v", err)
	}

	if err := db.Create(&City{Name: "Paris", Code: "P1"}).Error; err != nil {
		t.Fatalf("failed to create record: %v", err)
	}

	err = db.Create(&City{Name: "London", Code: "P1"}).Error
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Errorf("expected ErrDuplicatedKey on unique Code, got: %v", err)
	}

	var cities []City
	if err := db.Find(&cities, "code = ?", "P1").Error; err != nil {
		t.Fatalf("failed to fetch cities: %v", err)
	}
	if len(cities) != 1 {
		t.Fatalf("expected 1 city with code 'P1', found %d", len(cities))
	}
	if cities[0].Name != "Paris" {
		t.Errorf("expected city name 'Paris', got '%s'", cities[0].Name)
	}

	if err := db.Create(&City{Name: "NullCodeCity", Code: ""}).Error; err != nil {
		t.Fatalf("failed to create record with empty Code: %v", err)
	}
	if err := db.Create(&City{Name: "NullCodeCity2"}).Error; err != nil {
		t.Fatalf("failed to create second record with NULL Code: %v", err)
	}
}

func TestSupportedDialectorWithErrForeignKeyViolatedNegative(t *testing.T) {
	type City struct {
		gorm.Model
		Name string `gorm:"unique"`
	}

	type Museum struct {
		gorm.Model
		Name   string `gorm:"unique"`
		CityID uint
		City   City `gorm:"Constraint:OnUpdate:CASCADE,OnDelete:CASCADE;FOREIGNKEY:CityID;References:ID"`
	}

	db, err := OpenTestConnection(&gorm.Config{TranslateError: false})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	dialectors := map[string]bool{"sqlite": true, "postgres": true, "mysql": true, "sqlserver": true}
	if supported, found := dialectors[db.Dialector.Name()]; !(found && supported) {
		t.Logf("skipping test for unsupported dialector: %s", db.Dialector.Name())
		return
	}

	DB.Migrator().DropTable(&Museum{}, &City{})
	if err = db.AutoMigrate(&City{}, &Museum{}); err != nil {
		t.Fatalf("failed to migrate tables, got error: %v", err)
	}

	city := City{Name: "Berlin"}
	if err := db.Create(&city).Error; err != nil {
		t.Fatalf("failed to create city: %v", err)
	}

	err = db.Create(&Museum{Name: "Invalid FK Museum", CityID: 99999}).Error
	if errors.Is(err, gorm.ErrForeignKeyViolated) {
		t.Fatalf("expected raw DB error without translation, got: %v", err)
	}

	db, err = OpenTestConnection(&gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}
	DB.Migrator().DropTable(&Museum{}, &City{})
	if err = db.AutoMigrate(&City{}, &Museum{}); err != nil {
		t.Fatalf("failed to migrate tables, got error: %v", err)
	}

	city2 := City{Name: "Vienna"}
	if err := db.Create(&city2).Error; err != nil {
		t.Fatalf("failed to create city: %v", err)
	}
	museum := Museum{Name: "Vienna Museum", CityID: city2.ID}
	if err := db.Create(&museum).Error; err != nil {
		t.Fatalf("failed to create museum: %v", err)
	}

	if err := db.Delete(&city2).Error; err != nil {
		t.Fatalf("expected cascade delete success, got error: %v", err)
	}

	if err := db.Create(&Museum{Name: "Orphan Museum"}).Error; err != nil {
		t.Fatalf("expected orphan museum insert to succeed, got: %v", err)
	}
}
