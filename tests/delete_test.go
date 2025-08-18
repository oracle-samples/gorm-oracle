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

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils/tests"
)

func TestDelete(t *testing.T) {
	users := []User{*GetUser("delete", Config{}), *GetUser("delete", Config{}), *GetUser("delete", Config{})}

	if err := DB.Create(&users).Error; err != nil {
		t.Errorf("errors happened when create: %v", err)
	}

	for _, user := range users {
		if user.ID == 0 {
			t.Fatalf("user's primary key should has value after create, got : %v", user.ID)
		}
	}

	if res := DB.Delete(&users[1]); res.Error != nil || res.RowsAffected != 1 {
		t.Errorf("errors happened when delete: %v, affected: %v", res.Error, res.RowsAffected)
	}

	var result User
	if err := DB.Where("\"id\" = ?", users[1].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("\"id\" = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("\"id\" = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}

	if err := DB.Delete(&users[0]).Error; err != nil {
		t.Errorf("errors happened when delete: %v", err)
	}

	if err := DB.Delete(&User{}).Error; err != gorm.ErrMissingWhereClause {
		t.Errorf("errors happened when delete: %v", err)
	}

	if err := DB.Where("\"id\" = ?", users[0].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}
}

func TestDeleteWithTable(t *testing.T) {
	type UserWithDelete struct {
		gorm.Model
		Name string
	}

	DB.Table("deleted_users").Migrator().DropTable(UserWithDelete{})
	DB.Table("deleted_users").AutoMigrate(UserWithDelete{})

	user := UserWithDelete{Name: "delete1"}
	DB.Table("deleted_users").Create(&user)

	var result UserWithDelete
	if err := DB.Table("deleted_users").First(&result).Error; err != nil {
		t.Errorf("failed to find deleted user, got error %v", err)
	}

	tests.AssertEqual(t, result, user)

	if err := DB.Table("deleted_users").Delete(&result).Error; err != nil {
		t.Errorf("failed to delete user, got error %v", err)
	}

	var result2 UserWithDelete
	if err := DB.Table("deleted_users").First(&result2, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should raise record not found error, but got error %v", err)
	}

	var result3 UserWithDelete
	if err := DB.Table("deleted_users").Unscoped().First(&result3, user.ID).Error; err != nil {
		t.Fatalf("failed to find record, got error %v", err)
	}

	if err := DB.Table("deleted_users").Unscoped().Delete(&result).Error; err != nil {
		t.Errorf("failed to delete user with unscoped, got error %v", err)
	}

	var result4 UserWithDelete
	if err := DB.Table("deleted_users").Unscoped().First(&result4, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should raise record not found error, but got error %v", err)
	}
}

func TestInlineCondDelete(t *testing.T) {
	user1 := *GetUser("inline_delete_1", Config{})
	user2 := *GetUser("inline_delete_2", Config{})
	DB.Save(&user1).Save(&user2)

	if DB.Delete(&User{}, user1.ID).Error != nil {
		t.Errorf("No error should happen when delete a record")
	} else if err := DB.Where("\"name\" = ?", user1.Name).First(&User{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("User can't be found after delete")
	}

	if err := DB.Delete(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
		t.Errorf("No error should happen when delete a record, err=%s", err)
	} else if err := DB.Where("\"name\" = ?", user2.Name).First(&User{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("User can't be found after delete")
	}
}

func TestBlockGlobalDelete(t *testing.T) {
	if err := DB.Delete(&User{}).Error; err == nil || !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Errorf("should returns missing WHERE clause while deleting error")
	}

	if err := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&User{}).Error; err != nil {
		t.Errorf("should returns no error while enable global update, but got err %v", err)
	}
}

func TestDeleteWithAssociations(t *testing.T) {
	user := GetUser("delete_with_associations", Config{Account: true, Pets: 2, Toys: 4, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 1})

	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Select(clause.Associations, "Pets.Toy").Delete(&user).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 1, "Pets": 2, "Toys": 4, "Company": 1, "Manager": 1, "Team": 1, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

func TestDeleteAssociationsWithUnscoped(t *testing.T) {
	user := GetUser("unscoped_delete_with_associations", Config{Account: true, Pets: 2, Toys: 4, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 1})

	if err := DB.Create(user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Unscoped().Select(clause.Associations, "Pets.Toy").Delete(&user).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 1, "Manager": 1, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&user).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

func TestDeleteSliceWithAssociations(t *testing.T) {
	users := []User{
		*GetUser("delete_slice_with_associations1", Config{Account: true, Pets: 4, Toys: 1, Company: true, Manager: true, Team: 1, Languages: 1, Friends: 4}),
		*GetUser("delete_slice_with_associations2", Config{Account: true, Pets: 3, Toys: 2, Company: true, Manager: true, Team: 2, Languages: 2, Friends: 3}),
		*GetUser("delete_slice_with_associations3", Config{Account: true, Pets: 2, Toys: 3, Company: true, Manager: true, Team: 3, Languages: 3, Friends: 2}),
		*GetUser("delete_slice_with_associations4", Config{Account: true, Pets: 1, Toys: 4, Company: true, Manager: true, Team: 4, Languages: 4, Friends: 1}),
	}

	if err := DB.Create(users).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	if err := DB.Select(clause.Associations).Delete(&users).Error; err != nil {
		t.Fatalf("failed to delete user, got error %v", err)
	}

	for key, value := range map[string]int64{"Account": 4, "Pets": 10, "Toys": 10, "Company": 4, "Manager": 4, "Team": 10, "Languages": 0, "Friends": 0} {
		if count := DB.Unscoped().Model(&users).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}

	for key, value := range map[string]int64{"Account": 0, "Pets": 0, "Toys": 0, "Company": 4, "Manager": 4, "Team": 0, "Languages": 0, "Friends": 0} {
		if count := DB.Model(&users).Association(key).Count(); count != value {
			t.Errorf("user's %v expects: %v, got %v", key, value, count)
		}
	}
}

// only sqlite, postgres, sqlserver support returning
func TestSoftDeleteReturning(t *testing.T) {
	t.Skip()
	users := []*User{
		GetUser("delete-returning-1", Config{}),
		GetUser("delete-returning-2", Config{}),
		GetUser("delete-returning-3", Config{}),
	}
	DB.Create(&users)

	var results []User
	DB.Where("name IN ?", []string{users[0].Name, users[1].Name}).Clauses(clause.Returning{}).Delete(&results)
	if len(results) != 2 {
		t.Errorf("failed to return delete data, got %v", results)
	}

	var count int64
	DB.Model(&User{}).Where("name IN ?", []string{users[0].Name, users[1].Name, users[2].Name}).Count(&count)
	if count != 1 {
		t.Errorf("failed to delete data, current count %v", count)
	}
}

func TestDeleteReturning(t *testing.T) {
	companies := []Company{
		{Name: "delete-returning-1"},
		{Name: "delete-returning-2"},
		{Name: "delete-returning-3"},
	}
	DB.Create(&companies)

	var results []Company
	DB.Where("\"name\" IN ?", []string{companies[0].Name, companies[1].Name}).Clauses(clause.Returning{}).Delete(&results)
	if len(results) != 2 {
		t.Errorf("failed to return delete data, got %v", results)
	}

	var count int64
	DB.Model(&Company{}).Where("\"name\" IN ?", []string{companies[0].Name, companies[1].Name, companies[2].Name}).Count(&count)
	if count != 1 {
		t.Errorf("failed to delete data, current count %v", count)
	}
}

func TestDeleteWithOnDeleteCascade(t *testing.T) {
	type Room struct {
		ID      uint
		Name    string
		HouseID uint
	}
	type House struct {
		ID   uint
		Name string
		Room []Room `gorm:"foreignKey:HouseID;constraint:OnDelete:CASCADE"`
	}

	DB.Migrator().DropTable(&Room{}, &House{})
	DB.AutoMigrate(&House{}, &Room{})

	house := House{
		Name: "house1",
		Room: []Room{
			{Name: "living room"}, {Name: "bedroom"},
		},
	}
	DB.Create(&house)
	DB.Delete(&house)

	var count int64
	DB.Model(&Room{}).Where("\"house_id\" = ?", house.ID).Count(&count)
	if count != 0 {
		t.Errorf("expected cascade delete, but found %v children", count)
	}
}
func TestUnscopedDeleteByIDs(t *testing.T) {
	users := []*User{
		GetUser("unscoped_id_1", Config{}),
		GetUser("unscoped_id_2", Config{}),
	}
	DB.Create(&users)

	var ids []uint
	for _, user := range users {
		ids = append(ids, user.ID)
	}

	if err := DB.Unscoped().Where("\"id\" IN ?", ids).Delete(&User{}).Error; err != nil {
		t.Fatalf("failed to delete by IDs: %v", err)
	}

	var count int64
	DB.Unscoped().Model(&User{}).Where("\"id\" IN ?", ids).Count(&count)
	if count != 0 {
		t.Errorf("expected all users to be deleted, found %d", count)
	}
}

func TestDeleteByPrimaryKeyOnly(t *testing.T) {
	user := *GetUser("delete_by_pk", Config{})
	DB.Create(&user)

	if err := DB.Delete(&User{}, user.ID).Error; err != nil {
		t.Fatalf("failed to delete by primary key, got error: %v", err)
	}

	var result User
	if err := DB.First(&result, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("user should be deleted by primary key, but got error: %v", err)
	}
}

func TestDeleteByWhereClause(t *testing.T) {
	user := *GetUser("delete_by_where", Config{})
	DB.Create(&user)

	if err := DB.Where("\"name\" = ?", user.Name).Delete(&User{}).Error; err != nil {
		t.Fatalf("delete by WHERE clause failed: %v", err)
	}

	var result User
	if err := DB.First(&result, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("user should be deleted, got error: %v", err)
	}
}

func TestDeleteWithCompositePrimaryKey(t *testing.T) {
	type CompositeKey struct {
		OrderID  uint `gorm:"primaryKey"`
		ItemID   uint `gorm:"primaryKey"`
		Quantity int
	}

	DB.Migrator().DropTable(&CompositeKey{})
	DB.AutoMigrate(&CompositeKey{})

	rec := CompositeKey{OrderID: 1, ItemID: 1, Quantity: 10}
	DB.Create(&rec)

	if err := DB.Delete(&rec).Error; err != nil {
		t.Errorf("failed to delete record with composite key: %v", err)
	}
	var result CompositeKey
	if err := DB.Where("\"order_id\" = ? and \"item_id\" = ? ", rec.OrderID, rec.ItemID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}
}

func TestHardDeleteAfterSoftDelete(t *testing.T) {
	type SoftDelModel struct {
		gorm.Model
		Name string
	}

	DB.Migrator().DropTable(&SoftDelModel{})
	DB.AutoMigrate(&SoftDelModel{})

	record := SoftDelModel{Name: "soft-delete"}
	DB.Create(&record)

	DB.Delete(&record) // soft delete

	var temp SoftDelModel
	if err := DB.First(&temp, record.ID).Error; err == nil {
		t.Errorf("soft deleted record should not be found")
	}

	DB.Unscoped().Delete(&record) // hard delete
	if err := DB.Unscoped().First(&temp, record.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("hard deleted record should not exist, but got: %v", err)
	}
}

func TestDeleteWithLimitAndOrder(t *testing.T) {
	RunMigrations()
	users := []User{
		*GetUser("del-limited-1", Config{}),
		*GetUser("del-limited-2", Config{}),
		*GetUser("del-limited-3", Config{}),
	}
	DB.Create(&users)

	var user User
	DB.Where("\"name\" LIKE ?", "del-limited-%").Order("\"id\" desc").Limit(1).First(&user)
	DB.Delete(&user)

	var count int64
	DB.Model(&User{}).Where("\"name\" LIKE ?", "del-limited-%").Count(&count)
	if count != 2 {
		t.Errorf("expected 2 records after limited delete, got %v", count)
	}
}

func TestRawSQLDeleteWithLimit(t *testing.T) {
	RunMigrations()
	users := []User{
		*GetUser("del-limited-1", Config{}),
		*GetUser("del-limited-2", Config{}),
		*GetUser("del-limited-3", Config{}),
	}
	DB.Create(&users)

	DB.Exec(`
		  DELETE FROM "users"
		  WHERE rowid IN (
			SELECT rowid FROM "users"
			WHERE "name" LIKE ?
			ORDER BY "id" DESC FETCH FIRST 1 ROWS ONLY
		  )
		`, "del-limited-%")

	var count int64
	DB.Model(&User{}).Where("\"name\" LIKE ?", "del-limited-%").Count(&count)
	if count != 2 {
		t.Errorf("expected 2 records after limited delete, got %v", count)
	}
}
func TestRawSQLDelete(t *testing.T) {
	user := *GetUser("raw-sql-delete", Config{})
	DB.Create(&user)

	sql := `DELETE FROM "users" WHERE "id" = ?`
	if err := DB.Exec(sql, user.ID).Error; err != nil {
		t.Fatalf("raw SQL delete failed: %v", err)
	}

	var result User
	if err := DB.First(&result, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("user should be deleted via raw SQL")
	}
}

func TestDeleteCustomTableName(t *testing.T) {
	type AltTable struct {
		ID   int
		Name string
	}

	DB.Table("alt_table").Migrator().DropTable(&AltTable{})
	DB.Table("alt_table").AutoMigrate(&AltTable{})

	rec := AltTable{Name: "to-delete"}
	DB.Table("alt_table").Create(&rec)
	DB.Table("alt_table").Delete(&rec)

	var result AltTable
	err := DB.Table("alt_table").First(&result, rec.ID).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected not found after delete on custom table, got: %v", err)
	}
}
func TestDeleteOmitAssociations(t *testing.T) {
	user := GetUser("delete_omit_associations", Config{Account: true})
	DB.Create(user)

	if err := DB.Omit(clause.Associations).Delete(user).Error; err != nil {
		t.Fatalf("delete with omit associations failed: %v", err)
	}

	if count := DB.Unscoped().Model(user).Association("Account").Count(); count != 1 {
		t.Errorf("Account should not be deleted, count: %v", count)
	}
	if count := DB.Model(user).Association("Account").Count(); count != 1 {
		t.Errorf("Account should not be deleted, count: %v", count)
	}
}
func TestDeleteWithSelectField(t *testing.T) {
	user := *GetUser("delete_with_field", Config{})
	DB.Create(&user)

	err := DB.Select("id").Delete(&user).Error
	if err != nil {
		t.Errorf("delete with specific field failed: %v", err)
	}

	var result User
	if err := DB.Where("\"id\" = ?", user.ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}
}
func TestUnscopedBatchDelete(t *testing.T) {
	users := []User{
		*GetUser("unscoped-del-1", Config{}),
		*GetUser("unscoped-del-2", Config{}),
	}
	DB.Create(&users)

	if err := DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&users).Error; err != nil {
		t.Fatalf("unscoped delete failed: %v", err)
	}

	for _, user := range users {
		var result User
		if err := DB.Unscoped().First(&result, user.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("user should be fully deleted, got err: %v", err)
		}
	}
}
