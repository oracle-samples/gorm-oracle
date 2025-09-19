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
	"fmt"
	"regexp"
	"testing"

	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
)

func TestUpsert(t *testing.T) {
	lang := Language{Code: "upsert", Name: "Upsert"}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&lang).Error; err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	lang2 := Language{Code: "upsert", Name: "Upsert"}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&lang2).Error; err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	var langs []Language
	if err := DB.Find(&langs, "\"code\" = ?", lang.Code).Error; err != nil {
		t.Fatalf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs) != 1 {
		t.Fatalf("should only find only 1 languages, but got %+v", langs)
	}

	lang3 := Language{Code: "upsert", Name: "Upsert"}
	if err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"name": "upsert-new"}),
	}).Create(&lang3).Error; err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	if err := DB.Find(&langs, "\"code\" = ?", lang.Code).Error; err != nil {
		t.Fatalf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs) != 1 {
		t.Fatalf("should only find only 1 languages, but got %+v", langs)
	} else if langs[0].Name != "upsert-new" {
		t.Fatalf("should update name on conflict, but got name %+v", langs[0].Name)
	}

	lang = Language{Code: "upsert", Name: "Upsert-Newname"}
	if err := DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&lang).Error; err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	var result Language
	if err := DB.Find(&result, "\"code\" = ?", lang.Code).Error; err != nil || result.Name != lang.Name {
		t.Fatalf("failed to upsert, got name %v", result.Name)
	}

	type RestrictedLanguage struct {
		Code string `gorm:"primarykey"`
		Name string
		Lang string `gorm:"<-:create"`
	}

	r := DB.Session(&gorm.Session{DryRun: true}).Clauses(clause.OnConflict{UpdateAll: true}).Create(&RestrictedLanguage{Code: "upsert_code", Name: "upsert_name", Lang: "upsert_lang"})
	if !regexp.MustCompile(`MERGE INTO "restricted_languages".*WHEN MATCHED THEN UPDATE SET "name"="excluded"."name".*INSERT \("code","name","lang"\)`).MatchString(r.Statement.SQL.String()) {
		t.Fatalf("Table with escape character, got %v", r.Statement.SQL.String())
	}

	user := *GetUser("upsert_on_conflict", Config{})
	user.Age = 20
	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	var user2 User
	DB.First(&user2, user.ID)
	user2.Age = 30
	time.Sleep(time.Second)
	if err := DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&user2).Error; err != nil {
		t.Fatalf("failed to onconflict create user, got error %v", err)
	} else {
		var user3 User
		DB.First(&user3, user.ID)
		fmt.Printf("%d\n", user3.UpdatedAt.UnixNano())
		fmt.Printf("%d\n", user2.UpdatedAt.UnixNano())
		if user3.UpdatedAt.UnixNano() == user2.UpdatedAt.UnixNano() {
			t.Fatalf("failed to update user's updated_at, old: %v, new: %v", user2.UpdatedAt, user3.UpdatedAt)
		}
	}
}

func TestUpsertSlice(t *testing.T) {
	langs := []Language{
		{Code: "upsert-slice1", Name: "Upsert-slice1"},
		{Code: "upsert-slice2", Name: "Upsert-slice2"},
		{Code: "upsert-slice3", Name: "Upsert-slice3"},
	}
	DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&langs)

	var langs2 []Language
	if err := DB.Find(&langs2, "\"code\" LIKE ?", "upsert-slice%").Error; err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs2) != 3 {
		t.Errorf("should only find only 3 languages, but got %+v", langs2)
	}

	DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&langs)
	var langs3 []Language
	if err := DB.Find(&langs3, "\"code\" LIKE ?", "upsert-slice%").Error; err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs3) != 3 {
		t.Errorf("should only find only 3 languages, but got %+v", langs3)
	}

	for idx, lang := range langs {
		lang.Name = lang.Name + "_new"
		langs[idx] = lang
	}

	if err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(&langs).Error; err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	for _, lang := range langs {
		var results []Language
		if err := DB.Find(&results, "\"code\" = ?", lang.Code).Error; err != nil {
			t.Errorf("no error should happen when find languages with code, but got %v", err)
		} else if len(results) != 1 {
			t.Errorf("should only find only 1 languages, but got %+v", langs)
		} else if results[0].Name != lang.Name {
			t.Errorf("should update name on conflict, but got name %+v", results[0].Name)
		}
	}
}

func TestUpsertWithSave(t *testing.T) {
	langs := []Language{
		{Code: "upsert-save-1", Name: "Upsert-save-1"},
		{Code: "upsert-save-2", Name: "Upsert-save-2"},
	}

	if err := DB.Save(&langs).Error; err != nil {
		t.Errorf("Failed to create, got error %v", err)
	}

	for _, lang := range langs {
		var result Language
		if err := DB.First(&result, "\"code\" = ?", lang.Code).Error; err != nil {
			t.Errorf("Failed to query lang, got error %v", err)
		} else {
			tests.AssertEqual(t, result, lang)
		}
	}

	for idx, lang := range langs {
		lang.Name += "_new"
		langs[idx] = lang
	}

	if err := DB.Save(&langs).Error; err != nil {
		t.Errorf("Failed to upsert, got error %v", err)
	}

	for _, lang := range langs {
		var result Language
		if err := DB.First(&result, "\"code\" = ?", lang.Code).Error; err != nil {
			t.Errorf("Failed to query lang, got error %v", err)
		} else {
			tests.AssertEqual(t, result, lang)
		}
	}

	lang := Language{Code: "upsert-save-3", Name: "Upsert-save-3"}
	if err := DB.Save(&lang).Error; err != nil {
		t.Errorf("Failed to create, got error %v", err)
	}

	var result Language
	if err := DB.First(&result, "\"code\" = ?", lang.Code).Error; err != nil {
		t.Errorf("Failed to query lang, got error %v", err)
	} else {
		tests.AssertEqual(t, result, lang)
	}

	lang.Name += "_new"
	if err := DB.Save(&lang).Error; err != nil {
		t.Errorf("Failed to create, got error %v", err)
	}

	var result2 Language
	if err := DB.First(&result2, "\"code\" = ?", lang.Code).Error; err != nil {
		t.Errorf("Failed to query lang, got error %v", err)
	} else {
		tests.AssertEqual(t, result2, lang)
	}
}

func TestFindOrInitialize(t *testing.T) {
	var user1, user2, user3, user4, user5, user6 User
	if err := DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user1).Error; err != nil {
		t.Errorf("no error should happen when FirstOrInit, but got %v", err)
	}

	if user1.Name != "find or init" || user1.ID != 0 || user1.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.Where(User{Name: "find or init", Age: 33}).FirstOrInit(&user2)
	if user2.Name != "find or init" || user2.ID != 0 || user2.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.FirstOrInit(&user3, map[string]interface{}{"name": "find or init 2"})
	if user3.Name != "find or init 2" || user3.ID != 0 {
		t.Errorf("user should be initialized with inline search value")
	}

	DB.Where(&User{Name: "find or init"}).Attrs(User{Age: 44}).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.ID != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and attrs")
	}

	DB.Where(&User{Name: "find or init"}).Assign("age", 44).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.ID != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and assign attrs")
	}

	DB.Save(&User{Name: "find or init", Age: 33})
	DB.Where(&User{Name: "find or init"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or init" || user5.ID == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.ID == 0 || user6.Age != 33 {
		t.Errorf("user should be found with FirstOrInit")
	}

	DB.Where(&User{Name: "find or init"}).Assign(User{Age: 44}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.ID == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}
}

func TestFindOrCreate(t *testing.T) {
	var user1, user2, user3, user4, user5, user6, user7, user8 User
	if err := DB.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user1).Error; err != nil {
		t.Errorf("no error should happen when FirstOrInit, but got %v", err)
	}

	if user1.Name != "find or create" || user1.ID == 0 || user1.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	DB.Where(&User{Name: "find or create", Age: 33}).FirstOrCreate(&user2)
	if user1.ID != user2.ID || user2.Name != "find or create" || user2.ID == 0 || user2.Age != 33 {
		t.Errorf("user should be created with search value")
	}

	DB.FirstOrCreate(&user3, map[string]interface{}{"name": "find or create 2"})
	if user3.Name != "find or create 2" || user3.ID == 0 {
		t.Errorf("user should be created with inline search value")
	}

	DB.Where(&User{Name: "find or create 3"}).Attrs("age", 44).FirstOrCreate(&user4)
	if user4.Name != "find or create 3" || user4.ID == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and attrs")
	}

	updatedAt1 := user4.UpdatedAt
	DB.Where(&User{Name: "find or create 3"}).Assign("age", 55).FirstOrCreate(&user4)

	if user4.Age != 55 {
		t.Errorf("Failed to set change to 55, got %v", user4.Age)
	}

	if updatedAt1.Format(time.RFC3339Nano) == user4.UpdatedAt.Format(time.RFC3339Nano) {
		t.Errorf("UpdateAt should be changed when update values with assign")
	}

	DB.Where(&User{Name: "find or create 4"}).Assign(User{Age: 44}).FirstOrCreate(&user4)
	if user4.Name != "find or create 4" || user4.ID == 0 || user4.Age != 44 {
		t.Errorf("user should be created with search value and assigned attrs")
	}

	DB.Where(&User{Name: "find or create"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or create" || user5.ID == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	DB.Where(&User{Name: "find or create"}).Assign(User{Age: 44}).FirstOrCreate(&user6)
	if user6.Name != "find or create" || user6.ID == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	DB.Where(&User{Name: "find or create"}).Find(&user7)
	if user7.Name != "find or create" || user7.ID == 0 || user7.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}

	DB.Where(&User{Name: "find or create embedded struct"}).Assign(User{Age: 44, Account: Account{AccountNumber: "1231231231"}, Pets: []*Pet{{Name: "first_or_create_pet1"}, {Name: "first_or_create_pet2"}}}).FirstOrCreate(&user8)
	if err := DB.Where("\"name\" = ?", "first_or_create_pet1").First(&Pet{}).Error; err != nil {
		t.Errorf("has many association should be saved")
	}

	if err := DB.Where("\"account_number\" = ?", "1231231231").First(&Account{}).Error; err != nil {
		t.Errorf("belongs to association should be saved")
	}
}

func TestUpdateWithMissWhere(t *testing.T) {
	type User struct {
		ID   uint   `gorm:"column:id;<-:create"`
		Name string `gorm:"column:name"`
	}
	user := User{ID: 1, Name: "king"}
	tx := DB.Session(&gorm.Session{DryRun: true}).Save(&user)

	if err := tx.Error; err != nil {
		t.Fatalf("failed to update user,missing where condition,err=%+v", err)
	}

	if !regexp.MustCompile("WHERE .id. = [^ ]+$").MatchString(tx.Statement.SQL.String()) {
		t.Fatalf("invalid updating SQL, got %v", tx.Statement.SQL.String())
	}
}

type CompositeLang struct {
	Code string `gorm:"primaryKey;size:100"`
	Lang string `gorm:"primaryKey;size:10"`
	Name string
}

func TestUpsertCompositePK(t *testing.T) {
	langs := []CompositeLang{
		{Code: "c1", Lang: "en", Name: "English"},
		{Code: "c1", Lang: "fr", Name: "French"},
	}

	DB.Migrator().DropTable(&CompositeLang{})
	DB.Migrator().AutoMigrate(&CompositeLang{})

	if err := DB.Create(&langs).Error; err != nil {
		t.Fatalf("failed to insert composite PK: %v", err)
	}

	for i := range langs {
		langs[i].Name = langs[i].Name + "_updated"
	}

	if err := DB.Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&langs).Error; err != nil {
		t.Fatalf("failed to upsert composite PK: %v", err)
	}

	for _, expected := range langs {
		var result CompositeLang
		if err := DB.First(&result, "\"code\" = ? AND \"lang\" = ?", expected.Code, expected.Lang).Error; err != nil {
			t.Fatalf("failed to fetch row for %+v: %v", expected, err)
		}
		if result.Name != expected.Name {
			t.Fatalf("expected %v, got %v", expected.Name, result.Name)
		}
	}

	DB.Migrator().DropTable(&CompositeLang{})
}

func TestUpsertPrimaryKeyNotUpdated(t *testing.T) {
	lang := Language{Code: "pk1", Name: "Name1"}
	DB.Create(&lang)

	lang.Code = "pk2" // try changing PK
	DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&lang)

	var result Language
	DB.First(&result, "\"code\" = ?", "pk1")
	if result.Name != "Name1" {
		t.Fatalf("expected original row untouched, got %v", result)
	}
}

type LangWithIgnore struct {
	Code string `gorm:"primaryKey"`
	Name string
	Lang string `gorm:"<-:create"` // should not be updated
}

func TestUpsertIgnoreColumn(t *testing.T) {
	DB.Migrator().DropTable(&LangWithIgnore{})
	DB.Migrator().AutoMigrate(&LangWithIgnore{})
	lang := LangWithIgnore{Code: "upsert_ignore", Name: "OldName", Lang: "en"}
	DB.Create(&lang)

	lang.Name = "NewName"
	lang.Lang = "fr"
	DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&lang)

	var result LangWithIgnore
	DB.First(&result, "\"code\" = ?", lang.Code)
	if result.Name != "NewName" {
		t.Fatalf("expected Name updated, got %v", result.Name)
	}
	if result.Lang != "en" {
		t.Fatalf("Lang should not be updated, got %v", result.Lang)
	}
	DB.Migrator().DropTable(&LangWithIgnore{})
}

func TestUpsertNullValues(t *testing.T) {
	lang := Language{Code: "upsert_null", Name: ""}
	DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&lang)

	var result Language
	DB.First(&result, "\"code\" = ?", lang.Code)
	if result.Name != "" {
		t.Fatalf("expected empty Name, got %v", result.Name)
	}
}

func TestUpsertWithNullUnique(t *testing.T) {
	type NullLang struct {
		Code *string `gorm:"uniqueIndex"`
		Name string
	}
	DB.Migrator().DropTable(&NullLang{})
	DB.Migrator().AutoMigrate(&NullLang{})

	DB.Create(&NullLang{Code: nil, Name: "First"})

	if err := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(&NullLang{Code: nil, Name: "Second"}).Error; err != nil {
		t.Fatalf("unexpected error on upsert with NULL: %v", err)
	}

	var count int64
	DB.Model(&NullLang{}).Count(&count)
	if count != 2 {
		t.Fatalf("expected 2 rows due to NULL uniqueness, got %d", count)
	}
}

func TestUpsertSliceMixed(t *testing.T) {
	DB.Create(&Language{Code: "m1", Name: "Old1"})
	langs := []Language{
		{Code: "m1", Name: "New1"}, // exists
		{Code: "m2", Name: "New2"}, // new
	}

	DB.Clauses(clause.OnConflict{UpdateAll: true}).Create(&langs)

	var l1, l2 Language
	DB.First(&l1, "\"code\" = ?", "m1")
	DB.First(&l2, "\"code\" = ?", "m2")
	if l1.Name != "New1" || l2.Name != "New2" {
		t.Fatalf("batch mixed upsert failed: %+v, %+v", l1, l2)
	}
}

func TestUpsertWithExpressions(t *testing.T) {
	lang := Language{Code: "expr1", Name: "Name1"}
	DB.Create(&lang)

	DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "code"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr("UPPER(?)", "newname"),
		}),
	}).Create(&lang)

	var result Language
	DB.First(&result, "\"code\" = ?", "expr1")
	if result.Name != "NEWNAME" {
		t.Fatalf("expected NEWNAME, got %v", result.Name)
	}
}

func TestUpsertLargeBatch(t *testing.T) {
	var langs []Language
	for i := 0; i < 1000; i++ {
		langs = append(langs, Language{Code: fmt.Sprintf("lb_%d", i), Name: "Name"})
	}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&langs).Error; err != nil {
		t.Fatalf("failed large batch insert: %v", err)
	}
}

func TestUpsertFromSubquery(t *testing.T) {
	DB.Migrator().DropTable(&Language{})
	if err := DB.AutoMigrate(&Language{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	initial := []Language{
		{Code: "en", Name: "English"},
		{Code: "fr", Name: "French - Old"},  // Will be updated
		{Code: "es", Name: "Spanish - Old"}, // Will be updated
	}
	if err := DB.Create(&initial).Error; err != nil {
		t.Fatalf("failed to seed: %v", err)
	}

	updates := []Language{
		{Code: "fr", Name: "French - Updated"},
		{Code: "es", Name: "Spanish - Updated"},
		{Code: "de", Name: "German"}, // New record
	}

	for _, update := range updates {
		err := DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "code"}},
			DoUpdates: clause.AssignmentColumns([]string{"name"}),
		}).Create(&update).Error

		if err != nil {
			t.Fatalf("failed upsert: %v", err)
		}
	}

	var results []Language
	if err := DB.Order("\"code\"").Find(&results).Error; err != nil {
		t.Fatalf("failed to query results: %v", err)
	}

	expected := []Language{
		{Code: "de", Name: "German"},            // inserted
		{Code: "en", Name: "English"},           // unchanged
		{Code: "es", Name: "Spanish - Updated"}, // updated
		{Code: "fr", Name: "French - Updated"},  // updated
	}

	if len(results) != len(expected) {
		t.Errorf("expected %d rows, got %d", len(expected), len(results))
	}

	for i := range expected {
		if i >= len(results) {
			t.Errorf("missing row %d: expected (%s, %s)", i, expected[i].Code, expected[i].Name)
			continue
		}
		if results[i].Code != expected[i].Code || results[i].Name != expected[i].Name {
			t.Errorf("row %d mismatch: expected (%s, %s), got (%s, %s)",
				i, expected[i].Code, expected[i].Name,
				results[i].Code, results[i].Name)
		}
	}
}
