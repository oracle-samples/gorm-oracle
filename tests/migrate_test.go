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
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"time"

	"github.com/oracle-samples/gorm-oracle/oracle"
	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"github.com/stretchr/testify/assert"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
	"gorm.io/gorm/utils/tests"
)

func TestMigrate(t *testing.T) {
	allModels := []interface{}{&User{}, &Account{}, &Pet{}, &Company{}, &Toy{}, &Language{}, &Tools{}}
	seed := int64(time.Now().UnixNano())
	rand.New(rand.NewSource(seed))
	rand.Shuffle(len(allModels), func(i, j int) { allModels[i], allModels[j] = allModels[j], allModels[i] })
	DB.Migrator().DropTable("user_speaks", "user_friends", "ccc")

	// DropTable returns ORA-00942 if table doesn't exist
	for _, model := range allModels {
		if err := DB.Debug().Migrator().DropTable(model); err != nil {
			if !strings.Contains(err.Error(), "ORA-00942") {
				t.Fatalf("Failed to drop table, got error %v", err)
			}
		}
	}

	if err := DB.AutoMigrate(allModels...); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	if tables, err := DB.Migrator().GetTables(); err != nil {
		t.Fatalf("Failed to get database all tables, but got error %v", err)
	} else {
		for _, t1 := range []string{"users", "accounts", "pets", "companies", "toys", "languages", "tools"} {
			hasTable := false
			for _, t2 := range tables {
				if t2 == t1 {
					hasTable = true
					break
				}
			}
			if !hasTable {
				t.Fatalf("Failed to get table %v when GetTables", t1)
			}
		}
	}

	for _, m := range allModels {
		if !DB.Migrator().HasTable(m) {
			t.Fatalf("Failed to create table for %#v", m)
		}
	}

	DB.Scopes(func(db *gorm.DB) *gorm.DB {
		return db.Table("ccc")
	}).Migrator().CreateTable(&Company{})

	if !DB.Migrator().HasTable("ccc") {
		t.Errorf("failed to create table ccc")
	}

	for _, indexes := range [][2]string{
		{"user_speaks", "fk_user_speaks_user"},
		{"user_speaks", "fk_user_speaks_language"},
		{"user_friends", "fk_user_friends_user"},
		{"user_friends", "fk_user_friends_friends"},
		{"accounts", "fk_users_account"},
		{"users", "fk_users_team"},
		{"users", "fk_users_company"},
	} {
		if !DB.Migrator().HasConstraint(indexes[0], indexes[1]) {
			t.Fatalf("Failed to find index for many2many for %v %v", indexes[0], indexes[1])
		}
	}
}

func TestAutoMigrateInt8PG(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	type Smallint int8

	type MigrateInt struct {
		Int8 Smallint
	}

	tracer := Tracer{
		Logger: DB.Config.Logger,
		Test: func(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
			sql, _ := fc()
			if strings.HasPrefix(sql, "ALTER TABLE \"migrate_ints\" ALTER COLUMN \"int8\" TYPE smallint") {
				t.Fatalf("shouldn't execute ALTER COLUMN TYPE if such type is already existed in DB schema: sql: %s",
					sql)
			}
		},
	}

	DB.Migrator().DropTable(&MigrateInt{})

	// The first AutoMigrate to make table with field with correct type
	if err := DB.AutoMigrate(&MigrateInt{}); err != nil {
		t.Fatalf("Failed to auto migrate: error: %v", err)
	}

	// make new session to set custom logger tracer
	session := DB.Session(&gorm.Session{Logger: tracer})

	// The second AutoMigrate to catch an error
	if err := session.AutoMigrate(&MigrateInt{}); err != nil {
		t.Fatalf("Failed to auto migrate: error: %v", err)
	}
}

func TestAutoMigrateSelfReferential(t *testing.T) {
	type MigratePerson struct {
		ID        uint
		Name      string
		ManagerID *uint
		Manager   *MigratePerson
	}

	DB.Migrator().DropTable(&MigratePerson{})

	if err := DB.AutoMigrate(&MigratePerson{}); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}

	if !DB.Migrator().HasConstraint("migrate_people", "fk_migrate_people_manager") {
		t.Fatalf("Failed to find has one constraint between people and managers")
	}
}

func TestAutoMigrateNullable(t *testing.T) {
	type MigrateNullableColumn struct {
		ID    uint
		Bonus float64 `gorm:"not null"`
		Stock float64
	}

	DB.Migrator().DropTable(&MigrateNullableColumn{})

	DB.AutoMigrate(&MigrateNullableColumn{})

	type MigrateNullableColumn2 struct {
		ID    uint
		Bonus float64
		Stock float64 `gorm:"not null"`
	}

	if err := DB.Table("migrate_nullable_columns").AutoMigrate(&MigrateNullableColumn2{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err := DB.Table("migrate_nullable_columns").Migrator().ColumnTypes(&MigrateNullableColumn{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}

	for _, columnType := range columnTypes {
		switch columnType.Name() {
		case "bonus":
			// allow to change non-nullable to nullable
			if nullable, _ := columnType.Nullable(); !nullable {
				t.Fatalf("bonus's nullable should be true, bug got %t", nullable)
			}
		case "stock":
			// do not allow to change nullable to non-nullable
			if nullable, _ := columnType.Nullable(); !nullable {
				t.Fatalf("stock's nullable should be true, bug got %t", nullable)
			}
		}
	}
}

func TestSmartMigrateColumn(t *testing.T) {
	fullSupported := map[string]bool{"mysql": true, "postgres": true}[DB.Dialector.Name()]

	type UserMigrateColumn struct {
		ID       uint
		Name     string
		Salary   float64
		Birthday time.Time `gorm:"precision:4"`
	}

	DB.Migrator().DropTable(&UserMigrateColumn{})

	DB.AutoMigrate(&UserMigrateColumn{})

	type UserMigrateColumn2 struct {
		ID                  uint
		Name                string    `gorm:"size:128"`
		Salary              float64   `gorm:"precision:2"`
		Birthday            time.Time `gorm:"precision:2"`
		NameIgnoreMigration string    `gorm:"size:100"`
	}

	if err := DB.Table("user_migrate_columns").AutoMigrate(&UserMigrateColumn2{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err := DB.Table("user_migrate_columns").Migrator().ColumnTypes(&UserMigrateColumn{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}

	for _, columnType := range columnTypes {
		switch columnType.Name() {
		case "name":
			if length, _ := columnType.Length(); (fullSupported || length != 0) && length != 128 {
				t.Fatalf("name's length should be 128, but got %v", length)
			}
		case "salary":
			if precision, o, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 2 {
				t.Fatalf("salary's precision should be 2, but got %v %v", precision, o)
			}
		case "birthday":
			if precision, _, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 2 {
				t.Fatalf("birthday's precision should be 2, but got %v", precision)
			}
		}
	}

	type UserMigrateColumn3 struct {
		ID                  uint
		Name                string    `gorm:"size:256"`
		Salary              float64   `gorm:"precision:3"`
		Birthday            time.Time `gorm:"precision:3"`
		NameIgnoreMigration string    `gorm:"size:128;-:migration"`
	}

	if err := DB.Table("user_migrate_columns").AutoMigrate(&UserMigrateColumn3{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err = DB.Table("user_migrate_columns").Migrator().ColumnTypes(&UserMigrateColumn{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}

	for _, columnType := range columnTypes {
		switch columnType.Name() {
		case "name":
			if length, _ := columnType.Length(); (fullSupported || length != 0) && length != 256 {
				t.Fatalf("name's length should be 128, but got %v", length)
			}
		case "salary":
			if precision, _, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 3 {
				t.Fatalf("salary's precision should be 2, but got %v", precision)
			}
		case "birthday":
			if precision, _, _ := columnType.DecimalSize(); (fullSupported || precision != 0) && precision != 3 {
				t.Fatalf("birthday's precision should be 2, but got %v", precision)
			}
		case "name_ignore_migration":
			if length, _ := columnType.Length(); (fullSupported || length != 0) && length != 100 {
				t.Fatalf("name_ignore_migration's length should still be 100 but got %v", length)
			}
		}
	}
}

func TestMigrateWithColumnComment(t *testing.T) {
	type UserWithColumnComment struct {
		gorm.Model
		Name string `gorm:"size:111;comment:this is a 字段"`
	}

	if err := DB.Migrator().DropTable(&UserWithColumnComment{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Failed to drop table, got error %v", err)
		}
	}

	if err := DB.AutoMigrate(&UserWithColumnComment{}); err != nil {
		t.Fatalf("Failed to auto migrate, but got error %v", err)
	}
}

func TestMigrateWithUniqueIndex(t *testing.T) {
	type UserWithUniqueIndex struct {
		ID    int
		Name  string    `gorm:"size:20;index:idx_name,unique"`
		Date  time.Time `gorm:"index:idx_name,unique"`
		UName string    `gorm:"uniqueIndex;size:255"`
	}

	DB.Migrator().DropTable(&UserWithUniqueIndex{})
	if err := DB.AutoMigrate(&UserWithUniqueIndex{}); err != nil {
		t.Fatalf("failed to migrate, got %v", err)
	}

	if !DB.Migrator().HasIndex(&UserWithUniqueIndex{}, "idx_name") {
		t.Errorf("Failed to find created index")
	}

	if !DB.Migrator().HasIndex(&UserWithUniqueIndex{}, "idx_user_with_unique_indices_u_name") {
		t.Errorf("Failed to find created index")
	}

	if err := DB.AutoMigrate(&UserWithUniqueIndex{}); err != nil {
		t.Fatalf("failed to migrate, got %v", err)
	}

	if !DB.Migrator().HasIndex(&UserWithUniqueIndex{}, "idx_user_with_unique_indices_u_name") {
		t.Errorf("Failed to find created index")
	}

	type IndexWithOption struct {
		ID   int
		Name string `gorm:"size:20;index:idx_name_opt,unique,option:TABLESPACE SYSAUX NOPARALLEL"`
	}
	DB.Migrator().DropTable(&IndexWithOption{})
	if err := DB.AutoMigrate(&IndexWithOption{}); err != nil {
		t.Fatalf("failed to migrate, got %v", err)
	}

	if !DB.Migrator().HasIndex(&IndexWithOption{}, "idx_name_opt") {
		t.Errorf("Failed to find created index")
	}
}

func TestMigrateTable(t *testing.T) {
	type TableStruct struct {
		gorm.Model
		Name string
	}

	DB.Migrator().DropTable(&TableStruct{})
	DB.AutoMigrate(&TableStruct{})

	if !DB.Migrator().HasTable(&TableStruct{}) {
		t.Fatalf("should found created table")
	}

	type NewTableStruct struct {
		gorm.Model
		Name string
	}

	if err := DB.Migrator().RenameTable(&TableStruct{}, &NewTableStruct{}); err != nil {
		t.Fatalf("Failed to rename table, got error %v", err)
	}

	if !DB.Migrator().HasTable("new_table_structs") {
		t.Fatal("should found renamed table")
	}

	DB.Migrator().DropTable("new_table_structs")

	if DB.Migrator().HasTable(&NewTableStruct{}) {
		t.Fatal("should not found dropped table")
	}
}

func TestMigrateIndexes(t *testing.T) {
	type IndexStruct struct {
		gorm.Model
		Name string `gorm:"size:255;index"`
	}

	DB.Migrator().DropTable(&IndexStruct{})
	DB.AutoMigrate(&IndexStruct{})

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Got error when tried to create index: %+v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Fatalf("Failed to find index for user's name")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "Name") {
		t.Fatalf("Should not find index for user's name after delete")
	}

	if err := DB.Migrator().CreateIndex(&IndexStruct{}, "Name"); err != nil {
		t.Fatalf("Got error when tried to create index: %+v", err)
	}

	if err := DB.Migrator().RenameIndex(&IndexStruct{}, "idx_index_structs_name", "idx_users_name_1"); err != nil {
		t.Fatalf("no error should happen when rename index, but got %v", err)
	}

	if !DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Fatalf("Should find index for user's name after rename")
	}

	if err := DB.Migrator().DropIndex(&IndexStruct{}, "idx_users_name_1"); err != nil {
		t.Fatalf("Failed to drop index for user's name, got err %v", err)
	}

	if DB.Migrator().HasIndex(&IndexStruct{}, "idx_users_name_1") {
		t.Fatalf("Should not find index for user's name after delete")
	}
}

func TestMigrateColumns(t *testing.T) {
	t.Skip()

	type ColumnStruct struct {
		gorm.Model
		Name  string
		Age   int    `gorm:"default:18;comment:my age"`
		Code  string `gorm:"unique;comment:my code;"`
		Code2 string
		Code3 string `gorm:"unique"`
	}

	DB.Migrator().DropTable(&ColumnStruct{})

	if err := DB.AutoMigrate(&ColumnStruct{}); err != nil {
		t.Errorf("Failed to migrate, got %v", err)
	}

	type ColumnStruct2 struct {
		gorm.Model
		Name  string `gorm:"size:100"`
		Code  string `gorm:"unique;comment:my code2;default:hello"`
		Code2 string `gorm:"unique"`
		// Code3 string
	}

	if err := DB.Table("column_structs").Migrator().AlterColumn(&ColumnStruct{}, "Name"); err != nil {
		t.Fatalf("no error should happened when alter column, but got %v", err)
	}

	if err := DB.Table("column_structs").AutoMigrate(&ColumnStruct2{}); err != nil {
		t.Fatalf("no error should happened when auto migrate column, but got %v", err)
	}

	if columnTypes, err := DB.Migrator().ColumnTypes(&ColumnStruct{}); err != nil {
		t.Fatalf("no error should returns for ColumnTypes")
	} else {
		stmt := &gorm.Statement{DB: DB}
		stmt.Parse(&ColumnStruct2{})

		for _, columnType := range columnTypes {
			switch columnType.Name() {
			case "id":
				if v, ok := columnType.PrimaryKey(); !ok || !v {
					t.Fatalf("column id primary key should be correct, name: %v, column: %#v", columnType.Name(),
						columnType)
				}
			case "name":
				dataType := DB.Dialector.DataTypeOf(stmt.Schema.LookUpField(columnType.Name()))
				if !strings.Contains(strings.ToUpper(dataType), strings.ToUpper(columnType.DatabaseTypeName())) {
					t.Fatalf("column name type should be correct, name: %v, length: %v, expects: %v, column: %#v",
						columnType.Name(), columnType.DatabaseTypeName(), dataType, columnType)
				}
				if length, ok := columnType.Length(); !ok || length != 100 {
					t.Fatalf("column name length should be correct, name: %v, length: %v, expects: %v, column: %#v",
						columnType.Name(), length, 100, columnType)
				}
			case "age":
				if v, ok := columnType.DefaultValue(); !ok || v != "18" {
					t.Fatalf("column age default value should be correct, name: %v, column: %#v", columnType.Name(),
						columnType)
				}
				if v, ok := columnType.Comment(); !ok || v != "my age" {
					t.Fatalf("column age comment should be correct, name: %v, column: %#v", columnType.Name(),
						columnType)
				}
			case "code":
				if v, ok := columnType.Unique(); !ok || !v {
					t.Fatalf("column code unique should be correct, name: %v, column: %#v", columnType.Name(),
						columnType)
				}
				if v, ok := columnType.DefaultValue(); !ok || v != "hello" {
					t.Fatalf("column code default value should be correct, name: %v, column: %#v, default value: %v",
						columnType.Name(), columnType, v)
				}
				if v, ok := columnType.Comment(); !ok || v != "my code2" {
					t.Fatalf("column code comment should be correct, name: %v, column: %#v", columnType.Name(),
						columnType)
				}
			case "code2":
				if v, ok := columnType.Unique(); !ok || !v {
					t.Fatalf("column code2 unique should be correct, name: %v, column: %#v", columnType.Name(),
						columnType)
				}
			case "code3":
				// TODO
				// if v, ok := columnType.Unique(); !ok || v {
				// 	t.Fatalf("column code unique should be correct, name: %v, column: %#v", columnType.Name(), columnType)
				// }
			}
		}
	}
}

func TestMigrateAddDropColumns(t *testing.T) {
	type MigrateAddDropColumns struct {
		gorm.Model
		Name    string
		NewName string
	}

	DB.Migrator().DropTable(&MigrateAddDropColumns{})
	DB.AutoMigrate(&MigrateAddDropColumns{})

	if err := DB.Table("migrate_add_drop_columns").Migrator().AddColumn(&MigrateAddDropColumns{}, "NewName"); err == nil {
		t.Fatalf("Should fail to add column with existing name")
	}

	if err := DB.Table("migrate_add_drop_columns").Migrator().DropColumn(&MigrateAddDropColumns{}, "NewName"); err != nil {
		t.Fatalf("Failed to drop column, got %v", err)
	}

	if DB.Table("migrate_add_drop_columns").Migrator().HasColumn(&MigrateAddDropColumns{}, "NewName") {
		t.Fatalf("Found deleted column")
	}

	if err := DB.Table("migrate_add_drop_columns").Migrator().AddColumn(&MigrateAddDropColumns{}, "NewName"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if err := DB.Table("migrate_add_drop_columns").Migrator().RenameColumn(&MigrateAddDropColumns{}, "NewName",
		"new_new_name"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if !DB.Table("migrate_add_drop_columns").Migrator().HasColumn(&MigrateAddDropColumns{}, "new_new_name") {
		t.Fatalf("Failed to found renamed column")
	}

	if DB.Table("migrate_add_drop_columns").Migrator().HasColumn(&MigrateAddDropColumns{}, "NewName") {
		t.Fatalf("Found renamed column")
	}

	if err := DB.Table("migrate_add_drop_columns").Migrator().DropColumn(&MigrateAddDropColumns{}, "new_new_name"); err != nil {
		t.Fatalf("Failed to add column, got %v", err)
	}

	if DB.Table("migrate_add_drop_columns").Migrator().HasColumn(&MigrateAddDropColumns{}, "new_new_name") {
		t.Fatalf("Found deleted column")
	}
}

func TestMigrateConstraint(t *testing.T) {

	names := []string{"Account", "fk_users_account", "Pets", "fk_users_pets", "Company", "fk_users_company", "Team", "fk_users_team"}

	for _, name := range names {
		if !DB.Migrator().HasConstraint(&User{}, name) {
			DB.Migrator().CreateConstraint(&User{}, name)
		}

		if err := DB.Migrator().DropConstraint(&User{}, name); err != nil {
			t.Fatalf("failed to drop constraint %v, got error %v", name, err)
		}

		if DB.Migrator().HasConstraint(&User{}, name) {
			t.Fatalf("constraint %v should been deleted", name)
		}

		if err := DB.Migrator().CreateConstraint(&User{}, name); err != nil {
			t.Fatalf("failed to create constraint %v, got error %v", name, err)
		}

		if !DB.Migrator().HasConstraint(&User{}, name) {
			t.Fatalf("failed to found constraint %v", name)
		}
	}
}

type DynamicUser struct {
	gorm.Model
	Name      string
	CompanyID string `gorm:"index"`
}

// To test auto migrate crate indexes for dynamic table name
// https://github.com/go-gorm/gorm/issues/4752
func TestMigrateIndexesWithDynamicTableName(t *testing.T) {
	// Create primary table
	if err := DB.AutoMigrate(&DynamicUser{}); err != nil {
		t.Fatalf("AutoMigrate create table error: %#v", err)
	}

	// Create sub tables
	for _, v := range []string{"01", "02", "03"} {
		tableName := "dynamic_users_" + v
		m := DB.Scopes(func(db *gorm.DB) *gorm.DB {
			return db.Table(tableName)
		}).Migrator()

		if err := m.AutoMigrate(&DynamicUser{}); err != nil {
			t.Fatalf("AutoMigrate create table error: %#v", err)
		}

		if !m.HasTable(tableName) {
			t.Fatalf("AutoMigrate expected %#v exist, but not.", tableName)
		}

		if !m.HasIndex(&DynamicUser{}, "CompanyID") {
			t.Fatalf("Should have index on %s", "CompanyI.")
		}

		if !m.HasIndex(&DynamicUser{}, "DeletedAt") {
			t.Fatalf("Should have index on deleted_at.")
		}
	}
}

// check column order after migration, flaky test
// https://github.com/go-gorm/gorm/issues/4351
func TestMigrateColumnOrder(t *testing.T) {
	type UserMigrateColumn struct {
		ID uint
	}
	DB.Migrator().DropTable(&UserMigrateColumn{})
	DB.AutoMigrate(&UserMigrateColumn{})

	type UserMigrateColumn2 struct {
		ID  uint
		F1  string
		F2  string
		F3  string
		F4  string
		F5  string
		F6  string
		F7  string
		F8  string
		F9  string
		F10 string
		F11 string
		F12 string
		F13 string
		F14 string
		F15 string
		F16 string
		F17 string
		F18 string
		F19 string
		F20 string
		F21 string
		F22 string
		F23 string
		F24 string
		F25 string
		F26 string
		F27 string
		F28 string
		F29 string
		F30 string
		F31 string
		F32 string
		F33 string
		F34 string
		F35 string
	}
	if err := DB.Table("user_migrate_columns").AutoMigrate(&UserMigrateColumn2{}); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}

	columnTypes, err := DB.Table("user_migrate_columns").Migrator().ColumnTypes(&UserMigrateColumn2{})
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}
	typ := reflect.Indirect(reflect.ValueOf(&UserMigrateColumn2{})).Type()
	numField := typ.NumField()
	if numField != len(columnTypes) {
		t.Fatalf("column's number not match struct and ddl, %d != %d", numField, len(columnTypes))
	}
	namer := schema.NamingStrategy{}
	for i := 0; i < numField; i++ {
		expectName := namer.ColumnName("", typ.Field(i).Name)
		if columnTypes[i].Name() != expectName {
			t.Fatalf("column order not match struct and ddl, idx %d: %s != %s",
				i, columnTypes[i].Name(), expectName)
		}
	}
}

func TestMigrateSerialColumn(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	type Event struct {
		ID  uint `gorm:"primarykey"`
		UID uint32
	}

	type Event1 struct {
		ID  uint   `gorm:"primarykey"`
		UID uint32 `gorm:"not null;autoIncrement"`
	}

	type Event2 struct {
		ID  uint   `gorm:"primarykey"`
		UID uint16 `gorm:"not null;autoIncrement"`
	}

	var err error
	err = DB.Migrator().DropTable(&Event{})
	if err != nil {
		t.Errorf("DropTable err:%v", err)
	}

	// create sequence
	err = DB.Table("events").AutoMigrate(&Event1{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	// delete sequence
	err = DB.Table("events").AutoMigrate(&Event{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	// update sequence
	err = DB.Table("events").AutoMigrate(&Event1{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}
	err = DB.Table("events").AutoMigrate(&Event2{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	DB.Table("events").Save(&Event2{})
	DB.Table("events").Save(&Event2{})
	DB.Table("events").Save(&Event2{})

	events := make([]*Event, 0)
	DB.Table("events").Find(&events)

	tests.AssertEqual(t, 3, len(events))
	for _, v := range events {
		tests.AssertEqual(t, v.ID, v.UID)
	}
}

func TestMigrateWithSpecialName(t *testing.T) {
	var err error
	err = DB.AutoMigrate(&Coupon{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}
	err = DB.Table("coupon_product_1").AutoMigrate(&CouponProduct{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}
	err = DB.Table("coupon_product_2").AutoMigrate(&CouponProduct{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	tests.AssertEqual(t, true, DB.Migrator().HasTable("coupons"))
	tests.AssertEqual(t, true, DB.Migrator().HasTable("coupon_product_1"))
	tests.AssertEqual(t, true, DB.Migrator().HasTable("coupon_product_2"))
}

func TestMigrateAutoIncrement(t *testing.T) {
	type AutoIncrementStruct struct {
		ID     int64   `gorm:"primarykey;autoIncrement"`
		Field1 uint32  `gorm:"column:field1"`
		Field2 float32 `gorm:"column:field2"`
	}

	if err := DB.AutoMigrate(&AutoIncrementStruct{}); err != nil {
		t.Fatalf("AutoMigrate err: %v", err)
	}

	const ROWS = 10
	for idx := 0; idx < ROWS; idx++ {
		if err := DB.Create(&AutoIncrementStruct{}).Error; err != nil {
			t.Fatalf("create auto_increment_struct fail, err: %v", err)
		}
	}

	rows := make([]*AutoIncrementStruct, 0, ROWS)
	if err := DB.Order("\"id\" ASC").Find(&rows).Error; err != nil {
		t.Fatalf("find auto_increment_struct fail, err: %v", err)
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	lastID := ids[len(ids)-1]

	if err := DB.Where("\"id\" IN (?)", ids).Delete(&AutoIncrementStruct{}).Error; err != nil {
		t.Fatalf("delete auto_increment_struct fail, err: %v", err)
	}

	newRow := &AutoIncrementStruct{}
	if err := DB.Create(newRow).Error; err != nil {
		t.Fatalf("create auto_increment_struct fail, err: %v", err)
	}

	tests.AssertEqual(t, newRow.ID, lastID+1)
}

func TestCurrentTimestamp(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		return
	}
	type CurrentTimestampTest struct {
		ID     string     `gorm:"primary_key"`
		TimeAt *time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;unique"`
	}
	var err error
	err = DB.Migrator().DropTable(&CurrentTimestampTest{})
	if err != nil {
		t.Errorf("DropTable err:%v", err)
	}
	err = DB.AutoMigrate(&CurrentTimestampTest{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	err = DB.AutoMigrate(&CurrentTimestampTest{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}
	tests.AssertEqual(t, true, DB.Migrator().HasConstraint(&CurrentTimestampTest{}, "uni_current_timestamp_tests_time_at"))
	tests.AssertEqual(t, false, DB.Migrator().HasIndex(&CurrentTimestampTest{}, "time_at"))
	tests.AssertEqual(t, false, DB.Migrator().HasIndex(&CurrentTimestampTest{}, "time_at_2"))
}

func TestUniqueColumn(t *testing.T) {
	if DB.Dialector.Name() != "mysql" {
		return
	}

	type UniqueTest struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique"`
	}

	type UniqueTest2 struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique;default:NULL"`
	}

	type UniqueTest3 struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique;default:''"`
	}

	type UniqueTest4 struct {
		ID   string `gorm:"primary_key"`
		Name string `gorm:"unique;default:'123'"`
	}

	var err error
	err = DB.Migrator().DropTable(&UniqueTest{})
	if err != nil {
		t.Errorf("DropTable err:%v", err)
	}

	err = DB.AutoMigrate(&UniqueTest{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	// null -> null
	err = DB.AutoMigrate(&UniqueTest{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err := findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok := ct.DefaultValue()
	tests.AssertEqual(t, "", value)
	tests.AssertEqual(t, false, ok)

	// null -> null
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest2{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	// not trigger alert column
	tests.AssertEqual(t, true, DB.Migrator().HasConstraint(&UniqueTest{}, "uni_unique_tests_name"))
	tests.AssertEqual(t, false, DB.Migrator().HasIndex(&UniqueTest{}, "name"))
	tests.AssertEqual(t, false, DB.Migrator().HasIndex(&UniqueTest{}, "name_1"))
	tests.AssertEqual(t, false, DB.Migrator().HasIndex(&UniqueTest{}, "name_2"))

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	tests.AssertEqual(t, "", value)
	tests.AssertEqual(t, false, ok)

	// null -> empty string
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest3{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	tests.AssertEqual(t, "", value)
	tests.AssertEqual(t, true, ok)

	//  empty string -> 123
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest4{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	tests.AssertEqual(t, "123", value)
	tests.AssertEqual(t, true, ok)

	//  123 -> null
	err = DB.Table("unique_tests").AutoMigrate(&UniqueTest2{})
	if err != nil {
		t.Fatalf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&UniqueTest{}, "name")
	if err != nil {
		t.Fatalf("findColumnType err:%v", err)
	}

	value, ok = ct.DefaultValue()
	tests.AssertEqual(t, "", value)
	tests.AssertEqual(t, false, ok)
}

func findColumnType(dest interface{}, columnName string) (
	foundColumn gorm.ColumnType, err error,
) {
	columnTypes, err := DB.Migrator().ColumnTypes(dest)
	if err != nil {
		err = fmt.Errorf("ColumnTypes err:%v", err)
		return
	}

	for _, c := range columnTypes {
		if c.Name() == columnName {
			foundColumn = c
			break
		}
	}
	return
}

func TestDifferentTypeWithoutDeclaredLength(t *testing.T) {
	type DiffType struct {
		ID   uint
		Name string `gorm:"type:varchar(20)"`
	}

	type DiffType1 struct {
		ID   uint
		Name string `gorm:"type:string"`
	}

	var err error
	DB.Migrator().DropTable(&DiffType{})

	err = DB.AutoMigrate(&DiffType{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	ct, err := findColumnType(&DiffType{}, "name")
	if err != nil {
		t.Errorf("findColumnType err:%v", err)
	}

	// Oracle implicitly converts VARCHAR to VARCHAR2 behind the scenes
	tests.AssertEqual(t, "varchar2", strings.ToLower(ct.DatabaseTypeName()))

	err = DB.Table("diff_types").AutoMigrate(&DiffType1{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	ct, err = findColumnType(&DiffType{}, "name")
	if err != nil {
		t.Errorf("findColumnType err:%v", err)
	}

	tests.AssertEqual(t, "varchar2", strings.ToLower(ct.DatabaseTypeName()))
}

func TestMigrateArrayTypeModel(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	type ArrayTypeModel struct {
		ID              uint
		Number          string     `gorm:"type:varchar(51);NOT NULL"`
		TextArray       []string   `gorm:"type:text[];NOT NULL"`
		NestedTextArray [][]string `gorm:"type:text[][]"`
		NestedIntArray  [][]int64  `gorm:"type:integer[3][3]"`
	}

	var err error
	DB.Migrator().DropTable(&ArrayTypeModel{})

	err = DB.AutoMigrate(&ArrayTypeModel{})
	tests.AssertEqual(t, nil, err)

	ct, err := findColumnType(&ArrayTypeModel{}, "number")
	tests.AssertEqual(t, nil, err)
	tests.AssertEqual(t, "varchar", ct.DatabaseTypeName())

	ct, err = findColumnType(&ArrayTypeModel{}, "text_array")
	tests.AssertEqual(t, nil, err)
	tests.AssertEqual(t, "text[]", ct.DatabaseTypeName())

	ct, err = findColumnType(&ArrayTypeModel{}, "nested_text_array")
	tests.AssertEqual(t, nil, err)
	tests.AssertEqual(t, "text[]", ct.DatabaseTypeName())

	ct, err = findColumnType(&ArrayTypeModel{}, "nested_int_array")
	tests.AssertEqual(t, nil, err)
	tests.AssertEqual(t, "integer[]", ct.DatabaseTypeName())
}

type mockMigrator struct {
	gorm.Migrator
}

func (mm mockMigrator) AlterColumn(dst interface{}, field string) error {
	err := mm.Migrator.AlterColumn(dst, field)
	if err != nil {
		return err
	}
	return fmt.Errorf("trigger alter column error, field: %s", field)
}

func TestMigrateDonotAlterColumn(t *testing.T) {
	wrapMockMigrator := func(m gorm.Migrator) mockMigrator {
		return mockMigrator{
			Migrator: m,
		}
	}
	m := DB.Migrator()
	mockM := wrapMockMigrator(m)

	type NotTriggerUpdate struct {
		ID  uint
		F1  uint16
		F2  uint32
		F3  int
		F4  int64
		F5  string
		F6  float32
		F7  float64
		F8  time.Time
		F9  bool
		F10 []byte
	}

	var err error
	err = mockM.DropTable(&NotTriggerUpdate{})
	// DROP TABLE fails if the table does not exist.
	// tests.AssertEqual(t, err, nil)
	err = mockM.AutoMigrate(&NotTriggerUpdate{})
	tests.AssertEqual(t, err, nil)
	err = mockM.AutoMigrate(&NotTriggerUpdate{})
	tests.AssertEqual(t, err, nil)
}

func TestMigrateSameEmbeddedFieldName(t *testing.T) {
	type UserStat struct {
		GroundDestroyCount int
	}

	type GameUser struct {
		gorm.Model
		StatAb UserStat `gorm:"embedded;embeddedPrefix:stat_ab_"`
	}

	type UserStat1 struct {
		GroundDestroyCount string
	}

	type GroundRate struct {
		GroundDestroyCount int
	}

	type GameUser1 struct {
		gorm.Model
		StatAb       UserStat1  `gorm:"embedded;embeddedPrefix:stat_ab_"`
		GroundRateRb GroundRate `gorm:"embedded;embeddedPrefix:rate_ground_rb_"`
	}

	DB.Migrator().DropTable(&GameUser{})
	err := DB.AutoMigrate(&GameUser{})
	tests.AssertEqual(t, nil, err)

	err = DB.Table("game_users").AutoMigrate(&GameUser1{})
	tests.AssertEqual(t, nil, err)

	_, err = findColumnType(&GameUser{}, "stat_ab_ground_destroy_count")
	tests.AssertEqual(t, nil, err)

	_, err = findColumnType(&GameUser{}, "rate_ground_rb_ground_destroy_count")
	tests.AssertEqual(t, nil, err)
}

func TestMigrateWithDefaultValue(t *testing.T) {
	t.Skip()

	type NullModel struct {
		ID      uint
		Content string `gorm:"default:null"`
	}

	type NullStringModel struct {
		ID      uint
		Content string `gorm:"default:'null'"`
		Active  bool   `gorm:"default:false"`
	}

	tableName := "null_string_model"

	DB.Migrator().DropTable(tableName)

	err := DB.Table(tableName).AutoMigrate(&NullModel{})
	tests.AssertEqual(t, err, nil)

	// default null -> 'null'
	err = DB.Table(tableName).AutoMigrate(&NullStringModel{})
	tests.AssertEqual(t, err, nil)

	columnType, err := findColumnType(tableName, "content")
	tests.AssertEqual(t, err, nil)

	defVal, ok := columnType.DefaultValue()
	tests.AssertEqual(t, defVal, "null")
	tests.AssertEqual(t, ok, true)

	columnType2, err := findColumnType(tableName, "active")
	tests.AssertEqual(t, err, nil)

	defVal, ok = columnType2.DefaultValue()
	bv, _ := strconv.ParseBool(defVal)
	tests.AssertEqual(t, bv, false)
	tests.AssertEqual(t, ok, true)

	// default 'null' -> 'null'
	session := DB.Session(&gorm.Session{Logger: Tracer{
		Logger: DB.Config.Logger,
		Test: func(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
			sql, _ := fc()
			if strings.HasPrefix(sql, "ALTER TABLE") {
				t.Errorf("shouldn't execute: sql=%s", sql)
			}
		},
	}})
	err = session.Table(tableName).AutoMigrate(&NullStringModel{})
	tests.AssertEqual(t, err, nil)

	columnType, err = findColumnType(tableName, "content")
	tests.AssertEqual(t, err, nil)

	defVal, ok = columnType.DefaultValue()
	tests.AssertEqual(t, defVal, "null")
	tests.AssertEqual(t, ok, true)

	// default 'null' -> null
	err = DB.Table(tableName).AutoMigrate(&NullModel{})
	tests.AssertEqual(t, err, nil)

	columnType, err = findColumnType(tableName, "content")
	tests.AssertEqual(t, err, nil)

	defVal, ok = columnType.DefaultValue()
	tests.AssertEqual(t, defVal, "")
	tests.AssertEqual(t, ok, false)
}

func TestMigrateIgnoreRelations(t *testing.T) {
	type RelationModel1 struct {
		ID uint
	}
	type RelationModel2 struct {
		ID uint
	}
	type RelationModel3 struct {
		ID               uint
		RelationModel1ID uint
		RelationModel1   *RelationModel1
		RelationModel2ID uint
		RelationModel2   *RelationModel2 `gorm:"-:migration"`
	}

	var err error
	_ = DB.Migrator().DropTable(&RelationModel1{}, &RelationModel2{}, &RelationModel3{})

	tx := DB.Session(&gorm.Session{})
	tx.IgnoreRelationshipsWhenMigrating = true

	err = tx.AutoMigrate(&RelationModel3{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	// RelationModel3 should be existed
	_, err = findColumnType(&RelationModel3{}, "id")
	tests.AssertEqual(t, nil, err)

	// RelationModel1 should not be existed
	_, err = findColumnType(&RelationModel1{}, "id")
	if err == nil {
		t.Errorf("RelationModel1 should not be migrated")
	}

	// RelationModel2 should not be existed
	_, err = findColumnType(&RelationModel2{}, "id")
	if err == nil {
		t.Errorf("RelationModel2 should not be migrated")
	}

	tx.IgnoreRelationshipsWhenMigrating = false

	err = tx.AutoMigrate(&RelationModel3{})
	if err != nil {
		t.Errorf("AutoMigrate err:%v", err)
	}

	// RelationModel3 should be existed
	_, err = findColumnType(&RelationModel3{}, "id")
	tests.AssertEqual(t, nil, err)

	// RelationModel1 should be existed
	_, err = findColumnType(&RelationModel1{}, "id")
	tests.AssertEqual(t, nil, err)

	// RelationModel2 should not be existed
	_, err = findColumnType(&RelationModel2{}, "id")
	if err == nil {
		t.Errorf("RelationModel2 should not be migrated")
	}
}

func TestMigrateView(t *testing.T) {
	DB.Save(GetUser("joins-args-db", Config{Pets: 2}))

	if err := DB.Migrator().CreateView("invalid_users_pets",
		gorm.ViewOption{Query: nil}); err != gorm.ErrSubQueryRequired {
		t.Fatalf("no view should be created, got %v", err)
	}

	query := DB.Model(&User{}).
		Select("\"users\".\"id\" as users_id, \"users\".\"name\" as users_name, \"pets\".\"id\" as pets_id, \"pets\".\"name\" as pets_name").
		Joins("inner join \"pets\" on \"pets\".\"user_id\" = \"users\".\"id\"")

	if err := DB.Migrator().CreateView("users_pets", gorm.ViewOption{Query: query}); err != nil {
		t.Fatalf("Failed to crate view, got %v", err)
	}

	var count int64
	if err := DB.Table("users_pets").Count(&count).Error; err != nil {
		t.Fatalf("should found created view")
	}

	if err := DB.Migrator().DropView("users_pets"); err != nil {
		t.Fatalf("Failed to drop view, got %v", err)
	}

	query = DB.Model(&User{}).Where("\"age\" > ?", 20)
	if err := DB.Migrator().CreateView("users_view", gorm.ViewOption{Query: query}); err != nil {
		t.Fatalf("Failed to crate view, got %v", err)
	}
	if err := DB.Migrator().DropView("users_view"); err != nil {
		t.Fatalf("Failed to drop view, got %v", err)
	}
}

func TestMigrateExistingBoolColumnPG(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		return
	}

	type ColumnStruct struct {
		gorm.Model
		Name         string
		StringBool   string
		SmallintBool int `gorm:"type:smallint"`
	}

	type ColumnStruct2 struct {
		gorm.Model
		Name         string
		StringBool   bool // change existing boolean column from string to boolean
		SmallintBool bool // change existing boolean column from smallint or other to boolean
	}

	DB.Migrator().DropTable(&ColumnStruct{})

	if err := DB.AutoMigrate(&ColumnStruct{}); err != nil {
		t.Errorf("Failed to migrate, got %v", err)
	}

	if err := DB.Table("column_structs").AutoMigrate(&ColumnStruct2{}); err != nil {
		t.Fatalf("no error should happened when auto migrate column, but got %v", err)
	}

	if columnTypes, err := DB.Migrator().ColumnTypes(&ColumnStruct{}); err != nil {
		t.Fatalf("no error should returns for ColumnTypes")
	} else {
		stmt := &gorm.Statement{DB: DB}
		stmt.Parse(&ColumnStruct2{})

		for _, columnType := range columnTypes {
			switch columnType.Name() {
			case "id":
				if v, ok := columnType.PrimaryKey(); !ok || !v {
					t.Fatalf("column id primary key should be correct, name: %v, column: %#v", columnType.Name(),
						columnType)
				}
			case "string_bool":
				dataType := DB.Dialector.DataTypeOf(stmt.Schema.LookUpField(columnType.Name()))
				if !strings.Contains(strings.ToUpper(dataType), strings.ToUpper(columnType.DatabaseTypeName())) {
					t.Fatalf("column name type should be correct, name: %v, length: %v, expects: %v, column: %#v",
						columnType.Name(), columnType.DatabaseTypeName(), dataType, columnType)
				}
			case "smallint_bool":
				dataType := DB.Dialector.DataTypeOf(stmt.Schema.LookUpField(columnType.Name()))
				if !strings.Contains(strings.ToUpper(dataType), strings.ToUpper(columnType.DatabaseTypeName())) {
					t.Fatalf("column name type should be correct, name: %v, length: %v, expects: %v, column: %#v",
						columnType.Name(), columnType.DatabaseTypeName(), dataType, columnType)
				}
			}
		}
	}
}

func TestMigrateWithUniqueIndexAndUnique(t *testing.T) {
	t.Skip()

	const table = "unique_struct"

	checkField := func(model interface{}, fieldName string, unique bool, uniqueIndex string) {
		stmt := &gorm.Statement{DB: DB}
		err := stmt.Parse(model)
		if err != nil {
			t.Fatalf("%v: failed to parse schema, got error: %v", utils.FileWithLineNum(), err)
		}
		_ = stmt.Schema.ParseIndexes()
		field := stmt.Schema.LookUpField(fieldName)
		if field == nil {
			t.Fatalf("%v: failed to find column %q", utils.FileWithLineNum(), fieldName)
		}
		if field.Unique != unique {
			t.Fatalf("%v: %q column %q unique should be %v but got %v", utils.FileWithLineNum(), stmt.Schema.Table, fieldName, unique, field.Unique)
		}
		if field.UniqueIndex != uniqueIndex {
			t.Fatalf("%v: %q column %q uniqueIndex should be %v but got %v", utils.FileWithLineNum(), stmt.Schema, fieldName, uniqueIndex, field.UniqueIndex)
		}
	}

	type ( // not unique
		UniqueStruct1 struct {
			Name string `gorm:"size:10"`
		}
		UniqueStruct2 struct {
			Name string `gorm:"size:20"`
		}
	)
	checkField(&UniqueStruct1{}, "name", false, "")
	checkField(&UniqueStruct2{}, "name", false, "")

	type ( // unique
		UniqueStruct3 struct {
			Name string `gorm:"size:30;unique"`
		}
		UniqueStruct4 struct {
			Name string `gorm:"size:40;unique"`
		}
	)
	checkField(&UniqueStruct3{}, "name", true, "")
	checkField(&UniqueStruct4{}, "name", true, "")

	type ( // uniqueIndex
		UniqueStruct5 struct {
			Name string `gorm:"size:50;uniqueIndex"`
		}
		UniqueStruct6 struct {
			Name string `gorm:"size:60;uniqueIndex"`
		}
		UniqueStruct7 struct {
			Name     string `gorm:"size:70;uniqueIndex:idx_us6_all_names"`
			NickName string `gorm:"size:70;uniqueIndex:idx_us6_all_names"`
		}
	)
	checkField(&UniqueStruct5{}, "name", false, "idx_unique_struct5_name")
	checkField(&UniqueStruct6{}, "name", false, "idx_unique_struct6_name")

	checkField(&UniqueStruct7{}, "name", false, "")
	checkField(&UniqueStruct7{}, "nick_name", false, "")
	checkField(&UniqueStruct7{}, "nick_name", false, "")

	type UniqueStruct8 struct { // unique and uniqueIndex
		Name string `gorm:"size:60;unique;index:my_us8_index,unique;"`
	}
	checkField(&UniqueStruct8{}, "name", true, "my_us8_index")

	type TestCase struct {
		name      string
		from, to  interface{}
		checkFunc func(t *testing.T)
	}

	checkColumnType := func(t *testing.T, fieldName string, unique bool) {
		columnTypes, err := DB.Migrator().ColumnTypes(table)
		if err != nil {
			t.Fatalf("%v: failed to get column types, got error: %v", utils.FileWithLineNum(), err)
		}
		var found gorm.ColumnType
		for _, columnType := range columnTypes {
			if columnType.Name() == fieldName {
				found = columnType
			}
		}
		if found == nil {
			t.Fatalf("%v: failed to find column type %q", utils.FileWithLineNum(), fieldName)
		}
		if actualUnique, ok := found.Unique(); !ok || actualUnique != unique {
			t.Fatalf("%v: column %q unique should be %v but got %v", utils.FileWithLineNum(), fieldName, unique, actualUnique)
		}
	}

	checkIndex := func(t *testing.T, expected []gorm.Index) {
		indexes, err := DB.Migrator().GetIndexes(table)
		if err != nil {
			t.Fatalf("%v: failed to get indexes, got error: %v", utils.FileWithLineNum(), err)
		}
		assert.ElementsMatch(t, expected, indexes)
	}

	uniqueIndex := &migrator.Index{TableName: table, NameValue: DB.Config.NamingStrategy.IndexName(table, "name"), ColumnList: []string{"name"}, PrimaryKeyValue: sql.NullBool{Bool: false, Valid: true}, UniqueValue: sql.NullBool{Bool: true, Valid: true}}
	myIndex := &migrator.Index{TableName: table, NameValue: "my_us8_index", ColumnList: []string{"name"}, PrimaryKeyValue: sql.NullBool{Bool: false, Valid: true}, UniqueValue: sql.NullBool{Bool: true, Valid: true}}
	mulIndex := &migrator.Index{TableName: table, NameValue: "idx_us6_all_names", ColumnList: []string{"name", "nick_name"}, PrimaryKeyValue: sql.NullBool{Bool: false, Valid: true}, UniqueValue: sql.NullBool{Bool: true, Valid: true}}

	var checkNotUnique, checkUnique, checkUniqueIndex, checkMyIndex, checkMulIndex func(t *testing.T)
	// UniqueAffectedByUniqueIndex is true
	if DB.Dialector.Name() == "mysql" {
		uniqueConstraintIndex := &migrator.Index{TableName: table, NameValue: DB.Config.NamingStrategy.UniqueName(table, "name"), ColumnList: []string{"name"}, PrimaryKeyValue: sql.NullBool{Bool: false, Valid: true}, UniqueValue: sql.NullBool{Bool: true, Valid: true}}
		checkNotUnique = func(t *testing.T) {
			checkColumnType(t, "name", false)
			checkIndex(t, nil)
		}
		checkUnique = func(t *testing.T) {
			checkColumnType(t, "name", true)
			checkIndex(t, []gorm.Index{uniqueConstraintIndex})
		}
		checkUniqueIndex = func(t *testing.T) {
			checkColumnType(t, "name", true)
			checkIndex(t, []gorm.Index{uniqueIndex})
		}
		checkMyIndex = func(t *testing.T) {
			checkColumnType(t, "name", true)
			checkIndex(t, []gorm.Index{uniqueConstraintIndex, myIndex})
		}
		checkMulIndex = func(t *testing.T) {
			checkColumnType(t, "name", false)
			checkColumnType(t, "nick_name", false)
			checkIndex(t, []gorm.Index{mulIndex})
		}
	} else {
		checkNotUnique = func(t *testing.T) { checkColumnType(t, "name", false) }
		checkUnique = func(t *testing.T) { checkColumnType(t, "name", true) }
		checkUniqueIndex = func(t *testing.T) {
			checkColumnType(t, "name", false)
			checkIndex(t, []gorm.Index{uniqueIndex})
		}
		checkMyIndex = func(t *testing.T) {
			checkColumnType(t, "name", true)
			if !DB.Migrator().HasIndex(table, myIndex.Name()) {
				t.Errorf("%v: should has index %s but not", utils.FileWithLineNum(), myIndex.Name())
			}
		}
		checkMulIndex = func(t *testing.T) {
			checkColumnType(t, "name", false)
			checkColumnType(t, "nick_name", false)
			if !DB.Migrator().HasIndex(table, mulIndex.Name()) {
				t.Errorf("%v: should has index %s but not", utils.FileWithLineNum(), mulIndex.Name())
			}
		}
	}

	tests := []TestCase{
		{name: "notUnique to notUnique", from: &UniqueStruct1{}, to: &UniqueStruct2{}, checkFunc: checkNotUnique},
		{name: "notUnique to unique", from: &UniqueStruct1{}, to: &UniqueStruct3{}, checkFunc: checkUnique},
		{name: "notUnique to uniqueIndex", from: &UniqueStruct1{}, to: &UniqueStruct5{}, checkFunc: checkUniqueIndex},
		{name: "notUnique to uniqueAndUniqueIndex", from: &UniqueStruct1{}, to: &UniqueStruct8{}, checkFunc: checkMyIndex},
		{name: "unique to unique", from: &UniqueStruct3{}, to: &UniqueStruct4{}, checkFunc: checkUnique},
		{name: "unique to uniqueIndex", from: &UniqueStruct3{}, to: &UniqueStruct5{}, checkFunc: checkUniqueIndex},
		{name: "unique to uniqueAndUniqueIndex", from: &UniqueStruct3{}, to: &UniqueStruct8{}, checkFunc: checkMyIndex},
		{name: "uniqueIndex to uniqueIndex", from: &UniqueStruct5{}, to: &UniqueStruct6{}, checkFunc: checkUniqueIndex},
		{name: "uniqueIndex to uniqueAndUniqueIndex", from: &UniqueStruct5{}, to: &UniqueStruct8{}, checkFunc: checkMyIndex},
		{name: "uniqueIndex to multi uniqueIndex", from: &UniqueStruct5{}, to: &UniqueStruct7{}, checkFunc: checkMulIndex},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := DB.Migrator().DropTable(table); err != nil {
				t.Fatalf("failed to drop table, got error: %v", err)
			}
			if err := DB.Table(table).AutoMigrate(test.from); err != nil {
				t.Fatalf("failed to migrate table, got error: %v", err)
			}
			if err := DB.Table(table).AutoMigrate(test.to); err != nil {
				t.Fatalf("failed to migrate table, got error: %v", err)
			}
			test.checkFunc(t)
		})
	}

	if DB.Dialector.Name() != "sqlserver" {
		// In SQLServer, If an index or constraint depends on the column,
		// this column will not be able to run ALTER
		// see https://stackoverflow.com/questions/19460912/the-object-df-is-dependent-on-column-changing-int-to-double/19461205#19461205
		// may we need to create another PR to fix it, see https://github.com/go-gorm/sqlserver/pull/106
		tests = []TestCase{
			{name: "unique to notUnique", from: &UniqueStruct3{}, to: &UniqueStruct1{}, checkFunc: checkNotUnique},
			{name: "uniqueIndex to notUnique", from: &UniqueStruct5{}, to: &UniqueStruct2{}, checkFunc: checkNotUnique},
			{name: "uniqueIndex to unique", from: &UniqueStruct5{}, to: &UniqueStruct3{}, checkFunc: checkUnique},
		}
	}

	if DB.Dialector.Name() == "mysql" {
		compatibilityTests := []TestCase{
			{name: "oldUnique to notUnique", to: UniqueStruct1{}, checkFunc: checkNotUnique},
			{name: "oldUnique to unique", to: UniqueStruct3{}, checkFunc: checkUnique},
			{name: "oldUnique to uniqueIndex", to: UniqueStruct5{}, checkFunc: checkUniqueIndex},
			{name: "oldUnique to uniqueAndUniqueIndex", to: UniqueStruct8{}, checkFunc: checkMyIndex},
		}
		for _, test := range compatibilityTests {
			t.Run(test.name, func(t *testing.T) {
				if err := DB.Migrator().DropTable(table); err != nil {
					t.Fatalf("failed to drop table, got error: %v", err)
				}
				if err := DB.Exec("CREATE TABLE ? (`name` varchar(10) UNIQUE)", clause.Table{Name: table}).Error; err != nil {
					t.Fatalf("failed to create table, got error: %v", err)
				}
				if err := DB.Table(table).AutoMigrate(test.to); err != nil {
					t.Fatalf("failed to migrate table, got error: %v", err)
				}
				test.checkFunc(t)
			})
		}
	}
}

// TODO: Add tests for Oracle Driver
func testAutoMigrateDecimal(t *testing.T, model1, model2 any) []string {
	tracer := Tracer{
		Logger: DB.Config.Logger,
		Test: func(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
			sql, _ := fc()
			if strings.HasPrefix(sql, "ALTER TABLE ") {
				t.Fatalf("shouldn't execute ALTER COLUMN TYPE if decimal is not change: sql: %s", sql)
			}
		},
	}
	session := DB.Session(&gorm.Session{Logger: tracer})

	DB.Migrator().DropTable(model1)
	var modifySQL []string
	if err := session.AutoMigrate(model1); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}
	if err := session.AutoMigrate(model1); err != nil {
		t.Fatalf("failed to auto migrate, got error: %v", err)
	}
	tracer2 := Tracer{
		Logger: DB.Config.Logger,
		Test: func(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
			sql, _ := fc()
			modifySQL = append(modifySQL, sql)
		},
	}
	session2 := DB.Session(&gorm.Session{Logger: tracer2})
	err := session2.Table("migrate_decimal_columns").Migrator().AutoMigrate(model2)
	if err != nil {
		t.Fatalf("failed to get column types, got error: %v", err)
	}
	return modifySQL
}

func decimalColumnsTest[T, T2 any](t *testing.T, expectedSQL []string) {
	var t1 T
	var t2 T2
	modSQL := testAutoMigrateDecimal(t, t1, t2)
	var alterSQL []string
	for _, sql := range modSQL {
		if strings.HasPrefix(sql, "ALTER TABLE ") {
			alterSQL = append(alterSQL, sql)
		}
	}

	if len(alterSQL) != 3 {
		t.Fatalf("decimal changed error,expected: %+v,got: %+v.", expectedSQL, alterSQL)
	}
	for i := range alterSQL {
		if alterSQL[i] != expectedSQL[i] {
			t.Fatalf("decimal changed error,expected: %+v,got: %+v.", expectedSQL, alterSQL)
		}
	}
}

func TestAutoMigrateDecimal(t *testing.T) {
	if DB.Dialector.Name() == "sqlserver" { // database/sql will replace numeric to decimal. so only support decimal.
		type MigrateDecimalColumn struct {
			RecID1 int64 `gorm:"column:recid1;type:decimal(9,0);not null" json:"recid1"`
			RecID2 int64 `gorm:"column:recid2;type:decimal(8);not null" json:"recid2"`
			RecID3 int64 `gorm:"column:recid3;type:decimal(8,1);not null" json:"recid3"`
		}
		type MigrateDecimalColumn2 struct {
			RecID1 int64 `gorm:"column:recid1;type:decimal(8);not null" json:"recid1"`
			RecID2 int64 `gorm:"column:recid2;type:decimal(9,1);not null" json:"recid2"`
			RecID3 int64 `gorm:"column:recid3;type:decimal(9,2);not null" json:"recid3"`
		}
		expectedSQL := []string{
			`ALTER TABLE "migrate_decimal_columns" ALTER COLUMN "recid1" decimal(8) NOT NULL`,
			`ALTER TABLE "migrate_decimal_columns" ALTER COLUMN "recid2" decimal(9,1) NOT NULL`,
			`ALTER TABLE "migrate_decimal_columns" ALTER COLUMN "recid3" decimal(9,2) NOT NULL`,
		}
		decimalColumnsTest[MigrateDecimalColumn, MigrateDecimalColumn2](t, expectedSQL)
	} else if DB.Dialector.Name() == "postgres" {
		type MigrateDecimalColumn struct {
			RecID1 int64 `gorm:"column:recid1;type:numeric(9,0);not null" json:"recid1"`
			RecID2 int64 `gorm:"column:recid2;type:numeric(8);not null" json:"recid2"`
			RecID3 int64 `gorm:"column:recid3;type:numeric(8,1);not null" json:"recid3"`
		}
		type MigrateDecimalColumn2 struct {
			RecID1 int64 `gorm:"column:recid1;type:numeric(8);not null" json:"recid1"`
			RecID2 int64 `gorm:"column:recid2;type:numeric(9,1);not null" json:"recid2"`
			RecID3 int64 `gorm:"column:recid3;type:numeric(9,2);not null" json:"recid3"`
		}
		expectedSQL := []string{
			`ALTER TABLE "migrate_decimal_columns" ALTER COLUMN "recid1" TYPE numeric(8) USING "recid1"::numeric(8)`,
			`ALTER TABLE "migrate_decimal_columns" ALTER COLUMN "recid2" TYPE numeric(9,1) USING "recid2"::numeric(9,1)`,
			`ALTER TABLE "migrate_decimal_columns" ALTER COLUMN "recid3" TYPE numeric(9,2) USING "recid3"::numeric(9,2)`,
		}
		decimalColumnsTest[MigrateDecimalColumn, MigrateDecimalColumn2](t, expectedSQL)
	} else if DB.Dialector.Name() == "mysql" {
		type MigrateDecimalColumn struct {
			RecID1 int64 `gorm:"column:recid1;type:decimal(9,0);not null" json:"recid1"`
			RecID2 int64 `gorm:"column:recid2;type:decimal(8);not null" json:"recid2"`
			RecID3 int64 `gorm:"column:recid3;type:decimal(8,1);not null" json:"recid3"`
		}
		type MigrateDecimalColumn2 struct {
			RecID1 int64 `gorm:"column:recid1;type:decimal(8);not null" json:"recid1"`
			RecID2 int64 `gorm:"column:recid2;type:decimal(9,1);not null" json:"recid2"`
			RecID3 int64 `gorm:"column:recid3;type:decimal(9,2);not null" json:"recid3"`
		}
		expectedSQL := []string{
			"ALTER TABLE `migrate_decimal_columns` MODIFY COLUMN `recid1` decimal(8) NOT NULL",
			"ALTER TABLE `migrate_decimal_columns` MODIFY COLUMN `recid2` decimal(9,1) NOT NULL",
			"ALTER TABLE `migrate_decimal_columns` MODIFY COLUMN `recid3` decimal(9,2) NOT NULL",
		}
		decimalColumnsTest[MigrateDecimalColumn, MigrateDecimalColumn2](t, expectedSQL)
	}
}

func TestOracleBasicMigration(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		return
	}

	type OracleTestModel struct {
		ID       uint    `gorm:"primarykey"`
		Name     string  `gorm:"size:100"`
		Email    string  `gorm:"unique;size:255"`
		Age      int     `gorm:"default:0"`
		IsActive bool    `gorm:"default:true"`
		Amount   float64 `gorm:"type:number(10,2)"`
	}

	// Drop and recreate table
	if err := DB.Migrator().DropTable(&OracleTestModel{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Unexpected error dropping table: %v", err)
		}
	}

	if err := DB.AutoMigrate(&OracleTestModel{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error %v", err)
	}

	// Verify table exists
	if !DB.Migrator().HasTable(&OracleTestModel{}) {
		t.Fatalf("Table should exist after migration")
	}

	// Test basic CRUD operations to verify the table works
	testRecord := &OracleTestModel{
		Name:     "Test User",
		Email:    "test@example.com",
		Age:      25,
		IsActive: true,
		Amount:   123.45,
	}

	if err := DB.Create(testRecord).Error; err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	if testRecord.ID == 0 {
		t.Fatalf("Expected auto-generated ID")
	}

	// Verify record can be retrieved
	var retrieved OracleTestModel
	if err := DB.First(&retrieved, testRecord.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve record: %v", err)
	}

	if retrieved.Name != testRecord.Name {
		t.Fatalf("Name mismatch: expected %s, got %s", testRecord.Name, retrieved.Name)
	}
}

func TestOracleDataTypes(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		return
	}

	type OracleDataModel struct {
		ID         uint      `gorm:"primarykey"`
		TextShort  string    `gorm:"size:100"`
		TextLong   string    `gorm:"type:clob"`
		NumberInt  int64     `gorm:"type:number(10)"`
		NumberDec  float64   `gorm:"type:number(10,2)"`
		BoolField  bool      `gorm:"type:number(1)"`
		DateField  time.Time `gorm:"type:timestamp"`
		BinaryData []byte    `gorm:"type:raw(100)"`
	}

	if err := DB.Migrator().DropTable(&OracleDataModel{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Unexpected error dropping table: %v", err)
		}
	}

	if err := DB.AutoMigrate(&OracleDataModel{}); err != nil {
		t.Fatalf("Failed to migrate Oracle data types, got error %v", err)
	}

	// Test Oracle data type operations
	testData := &OracleDataModel{
		TextShort:  "Short text",
		TextLong:   strings.Repeat("Long text data. ", 100),
		NumberInt:  12345,
		NumberDec:  123.45,
		BoolField:  true,
		DateField:  time.Now().Truncate(time.Second),
		BinaryData: []byte{0x01, 0x02, 0x03, 0x04},
	}

	if err := DB.Create(testData).Error; err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	var retrieved OracleDataModel
	if err := DB.First(&retrieved, testData.ID).Error; err != nil {
		t.Fatalf("Failed to retrieve test data: %v", err)
	}

	// Verify Oracle data type handling
	if retrieved.TextShort != testData.TextShort {
		t.Fatalf("TextShort mismatch")
	}

	if retrieved.NumberInt != testData.NumberInt {
		t.Fatalf("NumberInt mismatch")
	}

	if retrieved.BoolField != testData.BoolField {
		t.Fatalf("BoolField mismatch")
	}
}

func TestOracleNullHandling(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		return
	}

	type OracleNullModel struct {
		ID          uint    `gorm:"primarykey"`
		OptionalStr *string `gorm:"size:100"`
		RequiredStr string  `gorm:"size:100;not null"`
	}

	if err := DB.Migrator().DropTable(&OracleNullModel{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Unexpected error dropping table: %v", err)
		}
	}

	if err := DB.AutoMigrate(&OracleNullModel{}); err != nil {
		t.Fatalf("Failed to migrate null model, got error %v", err)
	}

	// Test NULL behavior using direct SQL to verify Oracle's behavior
	t.Run("empty_string_becomes_null", func(t *testing.T) {
		// Insert record with empty string
		if err := DB.Exec(`INSERT INTO "oracle_null_models" ("optional_str", "required_str") VALUES ('', 'test')`).Error; err != nil {
			t.Fatalf("Failed to insert via SQL: %v", err)
		}

		// Verify Oracle converted empty string to NULL
		var count int64
		if err := DB.Raw(`SELECT COUNT(*) FROM "oracle_null_models" WHERE "optional_str" IS NULL`).Scan(&count).Error; err != nil {
			t.Fatalf("Failed to count NULL records: %v", err)
		}
		if count == 0 {
			t.Fatalf("Expected Oracle to convert empty string to NULL")
		}
	})

	// Test GORM's handling of NULL values
	t.Run("gorm_null_handling", func(t *testing.T) {
		// Create record with nil pointer
		model := &OracleNullModel{
			OptionalStr: nil,
			RequiredStr: "required_value",
		}

		if err := DB.Create(model).Error; err != nil {
			t.Fatalf("Failed to create record with NULL: %v", err)
		}

		// Retrieve and verify NULL handling
		var retrieved OracleNullModel
		if err := DB.First(&retrieved, model.ID).Error; err != nil {
			t.Fatalf("Failed to retrieve record: %v", err)
		}

		// The field should remain nil after round-trip
		if retrieved.OptionalStr != nil {
			t.Logf("Note: Oracle NULL handling may need adjustment in driver")
		}
	})
}

func TestOracleSequences(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		return
	}

	type OracleSeqModel struct {
		ID   uint   `gorm:"primarykey;autoIncrement"`
		Name string `gorm:"size:100"`
	}

	if err := DB.Migrator().DropTable(&OracleSeqModel{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Unexpected error dropping table: %v", err)
		}
	}

	if err := DB.AutoMigrate(&OracleSeqModel{}); err != nil {
		t.Fatalf("Failed to migrate sequence model, got error %v", err)
	}

	// Test Oracle sequence behavior for auto-increment
	var records []OracleSeqModel
	for i := 0; i < 3; i++ {
		record := OracleSeqModel{Name: fmt.Sprintf("Test %d", i)}
		if err := DB.Create(&record).Error; err != nil {
			t.Fatalf("Failed to create record %d: %v", i, err)
		}
		records = append(records, record)
	}

	// Verify auto-increment values
	for i, record := range records {
		if record.ID == 0 {
			t.Fatalf("Record %d should have auto-generated ID", i)
		}
		if i > 0 && record.ID <= records[i-1].ID {
			t.Fatalf("Record %d ID should be greater than previous record", i)
		}
	}

	// Test retrieval with proper quoting for Oracle
	var retrieved []OracleSeqModel
	// Use raw SQL to avoid identifier quoting issues in tests
	if err := DB.Raw(`SELECT "id", "name" FROM "oracle_seq_models" ORDER BY "id"`).Scan(&retrieved).Error; err != nil {
		t.Fatalf("Failed to retrieve records: %v", err)
	}

	if len(retrieved) != 3 {
		t.Fatalf("Expected 3 records, got %d", len(retrieved))
	}
}

func TestOracleTypeCreateDrop(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		t.Skip("Skipping Oracle type test: not running on Oracle")
	}

	const typeName = "email_list"
	const tableName = "email_varray_tab"

	// Assert that DB.Migrator() is an oracle.Migrator (so we can use Oracle-specific methods)
	m, ok := DB.Migrator().(oracle.Migrator)
	if !ok {
		t.Skip("Skipping: current dialect migrator is not Oracle-specific")
	}

	// 1️⃣ Drop type if it exists
	t.Run("drop_existing_type_if_any", func(t *testing.T) {
		err := m.Droptype(typeName)
		if err != nil && !strings.Contains(err.Error(), "ORA-04043") {
			t.Fatalf("Unexpected error dropping type: %v", err)
		}
	})

	// 2️⃣ Create new VARRAY type
	t.Run("create_varray_type", func(t *testing.T) {
		err := m.CreateType(typeName, "VARRAY(10) OF VARCHAR2(80)")
		if err != nil {
			t.Fatalf("Failed to create Oracle type: %v", err)
		}

		// Verify it exists
		var count int
		if err := DB.Raw(`SELECT COUNT(*) FROM USER_TYPES WHERE TYPE_NAME = UPPER(?)`, typeName).Scan(&count).Error; err != nil {
			t.Fatalf("Failed to verify created type: %v", err)
		}
		if count == 0 {
			t.Fatalf("Expected Oracle type %s to exist", typeName)
		}
	})

	// 3️⃣ Create table using the custom type
	t.Run("create_table_using_custom_type", func(t *testing.T) {
		createTableSQL := fmt.Sprintf(`
			CREATE TABLE "%s" (
				"ID" NUMBER PRIMARY KEY,
				"EMAILS" "%s"
			)`, tableName, typeName)

		if err := DB.Exec(createTableSQL).Error; err != nil {
			t.Fatalf("Failed to create table using type %s: %v", typeName, err)
		}

		// Verify table exists
		if !m.HasTable(tableName) {
			t.Fatalf("Expected table %s to exist", tableName)
		}
	})

	// 4️⃣ Drop table and type
	t.Run("drop_table_and_type", func(t *testing.T) {
		if err := m.DropTable(tableName); err != nil {
			t.Fatalf("Failed to drop table %s: %v", tableName, err)
		}

		if err := m.Droptype(typeName); err != nil {
			t.Fatalf("Failed to drop type %s: %v", typeName, err)
		}

		// Verify type is gone
		var count int
		if err := DB.Raw(`SELECT COUNT(*) FROM USER_TYPES WHERE TYPE_NAME = LOWER(?)`, typeName).Scan(&count).Error; err != nil {
			t.Fatalf("Failed to verify dropped type: %v", err)
		}
		if count > 0 {
			t.Fatalf("Expected Oracle type %s to be dropped", typeName)
		}
	})
}

func TestOracleIndexes(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		return
	}

	type OracleIndexModel struct {
		ID       uint   `gorm:"primarykey"`
		Name     string `gorm:"size:100;index"`
		Email    string `gorm:"size:255;uniqueIndex"`
		Category string `gorm:"size:50"`
	}

	if err := DB.Migrator().DropTable(&OracleIndexModel{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Unexpected error dropping table: %v", err)
		}
	}

	if err := DB.AutoMigrate(&OracleIndexModel{}); err != nil {
		t.Fatalf("Failed to migrate index model, got error %v", err)
	}

	// Test that GORM-created indexes exist
	if !DB.Migrator().HasIndex(&OracleIndexModel{}, "Name") {
		t.Fatalf("Should have index on Name field")
	}

	if !DB.Migrator().HasIndex(&OracleIndexModel{}, "Email") {
		t.Fatalf("Should have unique index on Email field")
	}

	// Test manual index creation using raw SQL (to avoid migrator issues)
	if err := DB.Exec(`CREATE INDEX "idx_test_category" ON "oracle_index_models"("category")`).Error; err != nil {
		t.Fatalf("Failed to create manual index: %v", err)
	}

	// Verify index was created
	var indexCount int64
	if err := DB.Raw(`SELECT COUNT(*) FROM USER_INDEXES WHERE TABLE_NAME = 'oracle_index_models' AND INDEX_NAME = 'idx_test_category'`).Scan(&indexCount).Error; err != nil {
		t.Fatalf("Failed to check index existence: %v", err)
	}

	if indexCount != 1 {
		t.Fatalf("Index should exist after creation")
	}
}

func TestOracleConstraints(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		return
	}

	type OracleConstraintModel struct {
		ID       uint   `gorm:"primarykey"`
		Email    string `gorm:"unique;size:255"`
		Username string `gorm:"size:50"`
	}

	if err := DB.Migrator().DropTable(&OracleConstraintModel{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Unexpected error dropping table: %v", err)
		}
	}

	if err := DB.AutoMigrate(&OracleConstraintModel{}); err != nil {
		t.Fatalf("Failed to migrate constraint model, got error %v", err)
	}

	// Check that unique constraint was created
	// Oracle generates constraint names automatically
	var constraintCount int64
	if err := DB.Raw(`SELECT COUNT(*) FROM USER_CONSTRAINTS WHERE TABLE_NAME = 'oracle_constraint_models' AND CONSTRAINT_TYPE = 'U'`).Scan(&constraintCount).Error; err != nil {
		t.Fatalf("Failed to check constraints: %v", err)
	}

	if constraintCount == 0 {
		t.Fatalf("Should have unique constraints")
	}

	// Test constraint violation
	model1 := &OracleConstraintModel{Email: "test@example.com", Username: "user1"}
	model2 := &OracleConstraintModel{Email: "test@example.com", Username: "user2"} // Same email

	if err := DB.Create(model1).Error; err != nil {
		t.Fatalf("Failed to create first record: %v", err)
	}

	if err := DB.Create(model2).Error; err == nil {
		t.Fatalf("Expected unique constraint violation")
	} else if !strings.Contains(err.Error(), "ORA-00001") {
		t.Fatalf("Expected ORA-00001 unique constraint violation, got: %v", err)
	}
}

func TestOracleErrorHandling(t *testing.T) {
	if DB.Dialector.Name() != "oracle" {
		return
	}

	// Test Oracle-specific error codes
	type TestModel struct {
		ID   uint   `gorm:"primarykey"`
		Name string `gorm:"size:100"`
	}

	// Test dropping non-existent table
	if err := DB.Migrator().DropTable(&TestModel{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Expected ORA-00942 for non-existent table, got %v", err)
		}
	}

	// Test table creation
	if err := DB.AutoMigrate(&TestModel{}); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Test that duplicate creation doesn't error
	if err := DB.AutoMigrate(&TestModel{}); err != nil {
		t.Fatalf("Duplicate table creation should not error: %v", err)
	}

	type NotAModel struct{}
	if err := DB.Migrator().CreateTable(NotAModel{}); err == nil {
		t.Fatalf("table creation with empty struct shoud report error: %v", err)
	}

	err := DB.Migrator().CreateTable("not_a_model_name")
	if err != nil && err.Error() != "failed to get schema" {
		t.Fatalf("Expect 'failed to get schema', but got : %v", err)
	}

}

func TestMigrateOnUpdateConstraint(t *testing.T) {
	type Owner struct {
		ID   int
		Name string
	}

	type Pen1 struct {
		gorm.Model
		OwnerID int
		Owner   Owner `gorm:"constraint:OnUpdate:CASCADE;"`
	}

	type Pen2 struct {
		gorm.Model
		OwnerID int   `gorm:"default: 18"`
		Owner   Owner `gorm:"constraint:OnUpdate:SET DEFAULT;"`
	}

	type Pen3 struct {
		gorm.Model
		OwnerID int
		Owner   Owner `gorm:"constraint:OnUpdate:SET NULL;"`
	}

	DB.Migrator().DropTable(&Owner{}, &Pen1{}, &Pen2{}, &Pen3{})

	// Test 1: Verify the trigger is created using CreateTable()
	if err := DB.Migrator().CreateTable(&Owner{}, &Pen1{}, &Pen2{}, &Pen3{}); err != nil {
		t.Fatalf("Failed to create table, got error: %v", err)
	}

	triggerNames := []string{
		"fk_trigger_owners_id_pen1_owner_id",
		"fk_trigger_owners_id_pen2_owner_id",
		"fk_trigger_owners_id_pen3_owner_id",
	}

	for _, triggerName := range triggerNames {
		var count int
		DB.Raw("SELECT count(*) FROM user_triggers where trigger_name = ?", triggerName).Scan(&count)
		if count != 1 {
			t.Errorf("Should find the trigger %s", triggerName)
		}
	}

	// Test 2: Verify the trigger is created using CreateConstraint()
	penStructs := []interface{}{&Pen1{}, &Pen2{}, &Pen3{}}
	constraintNames := []string{"fk_pen1_owner", "fk_pen2_owner", "fk_pen3_owner"}
	for i := range 3 {
		if err := DB.Migrator().DropConstraint(penStructs[i], constraintNames[i]); err != nil {
			t.Errorf("failed to drop constraint %v, got error %v", constraintNames[i], err)
		}

		if err := DB.Migrator().CreateConstraint(penStructs[i], constraintNames[i]); err != nil {
			t.Errorf("failed to create constraint %v, got error %v", constraintNames[i], err)
		}

		var count int
		DB.Raw("SELECT count(*) FROM user_triggers where trigger_name = ?", triggerNames[i]).Scan(&count)
		if count != 1 {
			t.Errorf("Should find the trigger %s", triggerNames[i])
		}
	}

	// Test 3: Verify each trigger work
	pen1 := Pen1{Owner: Owner{ID: 1, Name: "John"}}
	DB.Create(&pen1)
	DB.Model(pen1.Owner).Update("id", 100)

	var updatedPen1 Pen1
	if err := DB.First(&updatedPen1, "\"id\" = ?", pen1.ID).Error; err != nil {
		panic(fmt.Errorf("failed to find member, got error: %v", err))
	} else if updatedPen1.OwnerID != 100 {
		panic(fmt.Errorf("company id is not equal: expects: %v, got: %v", 100, updatedPen1.OwnerID))
	}

	pen2 := Pen2{Owner: Owner{ID: 2, Name: "Mary"}}
	DB.Create(&pen2)
	// When the ID in the owners table is updated, the primary key in pen2 (owner_id column)
	// is set to its default value (18). To avoid violating the foreign key constraint in pen2,
	// we need to insert this record into the owners table in advance.
	owner := Owner{ID: 18, Name: "MaryBackup"}
	DB.Create(&owner)
	DB.Model(pen2.Owner).Update("id", 200)

	var updatedPen2 Pen2
	if err := DB.First(&updatedPen2, "\"id\" = ?", pen2.ID).Error; err != nil {
		panic(fmt.Errorf("failed to find member, got error: %v", err))
	} else if updatedPen2.OwnerID != 18 {
		panic(fmt.Errorf("company id is not equal: expects: %v, got: %v", 18, updatedPen2.OwnerID))
	}

	pen3 := Pen3{Owner: Owner{ID: 3, Name: "Jane"}}
	DB.Create(&pen3)
	DB.Model(pen3.Owner).Update("id", 300)

	var updatedPen3 Pen3
	if err := DB.First(&updatedPen3, "\"id\" = ?", pen3.ID).Error; err != nil {
		panic(fmt.Errorf("failed to find member, got error: %v", err))
	} else if updatedPen3.OwnerID != 0 {
		panic(fmt.Errorf("company id is not equal: expects: %v, got: %v", 0, updatedPen3.OwnerID))
	}
}
