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
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"testing"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestGenericsCreate(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "TestGenericsCreate", Age: 18}
	err := gorm.G[User](DB).Create(ctx, &user)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("no primary key found for %v", user)
	}

	if u, err := gorm.G[User](DB).Where("\"name\" = ?", user.Name).First(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != user.Name || u.ID != user.ID {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	if u, err := gorm.G[User](DB).Where("\"name\" = ?", user.Name).Take(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != user.Name || u.ID != user.ID {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	if u, err := gorm.G[User](DB).Select("name").Where("\"name\" = ?", user.Name).First(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != user.Name || u.Age != 0 {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	if u, err := gorm.G[User](DB).Omit("name").Where("\"name\" = ?", user.Name).First(ctx); err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != "" || u.Age != user.Age {
		t.Errorf("found invalid user, got %v, expect %v", u, user)
	}

	result := struct {
		ID   int
		Name string
	}{}
	if err := gorm.G[User](DB).Where("\"name\" = ?", user.Name).Scan(ctx, &result); err != nil {
		t.Fatalf("failed to scan user, got error: %v", err)
	} else if result.Name != user.Name || uint(result.ID) != user.ID {
		t.Errorf("found invalid user, got %v, expect %v", result, user)
	}

	mapResult, err := gorm.G[map[string]interface{}](DB).Table("users").Where("\"name\" = ?", user.Name).MapColumns(map[string]string{"name": "user_name"}).Take(ctx)
	if v := mapResult["user_name"]; fmt.Sprint(v) != user.Name {
		t.Errorf("failed to find map results, got %v, err %v", mapResult, err)
	}
}

func TestGenericsCreateInBatches(t *testing.T) {
	batch := []User{
		{Name: "GenericsCreateInBatches1"},
		{Name: "GenericsCreateInBatches2"},
		{Name: "GenericsCreateInBatches3"},
	}
	ctx := context.Background()

	if err := gorm.G[User](DB).CreateInBatches(ctx, &batch, 2); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	for _, u := range batch {
		if u.ID == 0 {
			t.Fatalf("no primary key found for %v", u)
		}
	}

	count, err := gorm.G[User](DB).Where("\"name\" like ?", "GenericsCreateInBatches%").Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}

	found, err := gorm.G[User](DB).Raw("SELECT * FROM \"users\" WHERE \"name\" LIKE ?", "GenericsCreateInBatches%").Find(ctx)
	if err != nil {
		t.Fatalf("Raw Find failed: %v", err)
	}
	if len(found) != len(batch) {
		t.Errorf("expected %d from Raw Find, got %d", len(batch), len(found))
	}

	found, err = gorm.G[User](DB).Where("\"name\" like ?", "GenericsCreateInBatches%").Limit(2).Find(ctx)
	if err != nil {
		t.Fatalf("Raw Find failed: %v", err)
	}
	if len(found) != 2 {
		t.Errorf("expected %d from Raw Find, got %d", 2, len(found))
	}

	found, err = gorm.G[User](DB).Where("\"name\" like ?", "GenericsCreateInBatches%").Offset(2).Limit(2).Find(ctx)
	if err != nil {
		t.Fatalf("Raw Find failed: %v", err)
	}
	if len(found) != 1 {
		t.Errorf("expected %d from Raw Find, got %d", 1, len(found))
	}
}

func TestGenericsExecAndUpdate(t *testing.T) {
	ctx := context.Background()

	name := "GenericsExec"
	if err := gorm.G[User](DB).Exec(ctx, "INSERT INTO \"users\"(\"name\") VALUES(?)", name); err != nil {
		t.Fatalf("Exec insert failed: %v", err)
	}
	// todo: uncomment the below line, once the alias quoting issue is resolved.
	// Gorm issue track: https://github.com/oracle-samples/gorm-oracle/issues/36
	// u, err := gorm.G[User](DB).Table("\"users\" u").Where("u.\"name\" = ?", name).First(ctx)
	u, err := gorm.G[User](DB).Table("users").Where("\"name\" = ?", name).First(ctx)
	if err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if u.Name != name || u.ID == 0 {
		t.Errorf("found invalid user, got %v", u)
	}

	name += "Update"
	rows, err := gorm.G[User](DB).Where("\"id\" = ?", u.ID).Update(ctx, "name", name)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	} else if rows != 1 {
		t.Fatalf("failed to get affected rows, got %d, should be %d", rows, 1)
	}

	nu, err := gorm.G[User](DB).Where("\"name\" = ?", name).First(ctx)
	if err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if nu.Name != name || u.ID != nu.ID {
		t.Fatalf("found invalid user, got %v, expect %v", nu.ID, u.ID)
	}

	rows, err = gorm.G[User](DB).Where("\"id\" = ?", u.ID).Updates(ctx, User{Name: "GenericsExecUpdates", Age: 18})
	if err != nil {
		t.Fatalf("Updates failed: %v", err)
	} else if rows != 1 {
		t.Fatalf("failed to get affected rows, got %d, should be %d", rows, 1)
	}

	nu, err = gorm.G[User](DB).Where("\"id\" = ?", u.ID).Last(ctx)
	if err != nil {
		t.Fatalf("failed to find user, got error: %v", err)
	} else if nu.Name != "GenericsExecUpdates" || nu.Age != 18 || u.ID != nu.ID {
		t.Fatalf("found invalid user, got %v, expect %v", nu.ID, u.ID)
	}
}

func TestGenericsRow(t *testing.T) {
	ctx := context.Background()

	user := User{Name: "GenericsRow"}
	if err := gorm.G[User](DB).Create(ctx, &user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	row := gorm.G[User](DB).Raw("SELECT \"name\" FROM \"users\" WHERE \"id\" = ?", user.ID).Row(ctx)
	var name string
	if err := row.Scan(&name); err != nil {
		t.Fatalf("Row scan failed: %v", err)
	}
	if name != user.Name {
		t.Errorf("expected %s, got %s", user.Name, name)
	}

	user2 := User{Name: "GenericsRow2"}
	if err := gorm.G[User](DB).Create(ctx, &user2); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	rows, err := gorm.G[User](DB).Raw("SELECT \"name\" FROM \"users\" WHERE \"id\" IN ?", []uint{user.ID, user2.ID}).Rows(ctx)
	if err != nil {
		t.Fatalf("Rows failed: %v", err)
	}

	count := 0
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("rows.Scan failed: %v", err)
		}
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 rows, got %d", count)
	}
}

func TestGenericsDelete(t *testing.T) {
	ctx := context.Background()

	u := User{Name: "GenericsDelete"}
	if err := gorm.G[User](DB).Create(ctx, &u); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	rows, err := gorm.G[User](DB).Where("\"id\" = ?", u.ID).Delete(ctx)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if rows != 1 {
		t.Errorf("expected 1 row deleted, got %d", rows)
	}

	_, err = gorm.G[User](DB).Where("\"id\" = ?", u.ID).First(ctx)
	if err != gorm.ErrRecordNotFound {
		t.Fatalf("User after delete failed: %v", err)
	}

	u2 := User{Name: "GenericsDeleteCond"}
	if err := gorm.G[User](DB).Create(ctx, &u2); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	rows, err = gorm.G[User](DB).Where("\"name\" = ?", "GenericsDeleteCond").Delete(ctx)
	if err != nil {
		t.Fatalf("Conditional delete failed: %v", err)
	}
	if rows != 1 {
		t.Fatalf("Conditional delete failed, err=%v, rows=%d", err, rows)
	}
	_, err = gorm.G[User](DB).Where("\"id\" = ?", u2.ID).First(ctx)
	if err != gorm.ErrRecordNotFound {
		t.Errorf("Expected deleted record to be gone, got: %v", err)
	}

	users := []User{
		{Name: "GenericsBatchDel1"},
		{Name: "GenericsBatchDel2"},
	}
	if err := gorm.G[User](DB).CreateInBatches(ctx, &users, 2); err != nil {
		t.Fatalf("Batch create for delete failed: %v", err)
	}
	rows, err = gorm.G[User](DB).Where("\"name\" LIKE ?", "GenericsBatchDel%").Delete(ctx)
	if err != nil {
		t.Fatalf("Batch delete failed: %v", err)
	}
	if rows != 2 {
		t.Errorf("batch delete expected 2 rows, got %d", rows)
	}
}

func TestGenericsFindInBatches(t *testing.T) {
	ctx := context.Background()

	users := []User{
		{Name: "GenericsFindBatchA"},
		{Name: "GenericsFindBatchB"},
		{Name: "GenericsFindBatchC"},
		{Name: "GenericsFindBatchD"},
		{Name: "GenericsFindBatchE"},
	}
	if err := gorm.G[User](DB).CreateInBatches(ctx, &users, len(users)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	total := 0
	err := gorm.G[User](DB).Where("\"name\" like ?", "GenericsFindBatch%").FindInBatches(ctx, 2, func(chunk []User, batch int) error {
		if len(chunk) > 2 {
			t.Errorf("batch size exceed 2: got %d", len(chunk))
		}

		total += len(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("FindInBatches failed: %v", err)
	}

	if total != len(users) {
		t.Errorf("expected total %d, got %d", len(users), total)
	}
}

func TestGenericsScopes(t *testing.T) {
	ctx := context.Background()

	users := []User{{Name: "GenericsScopes1"}, {Name: "GenericsScopes2"}, {Name: "GenericsScopes3"}}
	err := gorm.G[User](DB).CreateInBatches(ctx, &users, len(users))
	if err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	filterName1 := func(stmt *gorm.Statement) {
		stmt.Where("\"name\" = ?", "GenericsScopes1")
	}

	results, err := gorm.G[User](DB).Scopes(filterName1).Find(ctx)
	if err != nil {
		t.Fatalf("Scopes failed: %v", err)
	} else if len(results) != 1 || results[0].Name != "GenericsScopes1" {
		t.Fatalf("Scopes expected 1, got %d", len(results))
	}

	notResult, err := gorm.G[User](DB).Where("\"name\" like ?", "GenericsScopes%").Not("\"name\" = ?", "GenericsScopes1").Order("\"name\"").Find(ctx)
	if err != nil {
		t.Fatalf("Not failed: %v", err)
	} else if len(notResult) != 2 {
		t.Fatalf("expected 2 results, got %d", len(notResult))
	} else if notResult[0].Name != "GenericsScopes2" || notResult[1].Name != "GenericsScopes3" {
		t.Fatalf("expected names 'GenericsScopes2' and 'GenericsScopes3', got %s and %s", notResult[0].Name, notResult[1].Name)
	}

	orResult, err := gorm.G[User](DB).Or("\"name\" = ?", "GenericsScopes1").Or("\"name\" = ?", "GenericsScopes2").Order("\"name\"").Find(ctx)
	if err != nil {
		t.Fatalf("Or failed: %v", err)
	} else if len(orResult) != 2 {
		t.Fatalf("expected 2 results, got %d", len(notResult))
	} else if orResult[0].Name != "GenericsScopes1" || orResult[1].Name != "GenericsScopes2" {
		t.Fatalf("expected names 'GenericsScopes2' and 'GenericsScopes3', got %s and %s", orResult[0].Name, orResult[1].Name)
	}
}

func TestGenericsJoins(t *testing.T) {
	ctx := context.Background()
	db := gorm.G[User](DB)

	u := User{Name: "GenericsJoins", Company: Company{Name: "GenericsCompany"}}
	u2 := User{Name: "GenericsJoins_2", Company: Company{Name: "GenericsCompany_2"}}
	u3 := User{Name: "GenericsJoins_3", Company: Company{Name: "GenericsCompany_3"}}
	db.CreateInBatches(ctx, &[]User{u3, u, u2}, 10)

	// Inner JOIN + WHERE
	result, err := db.Joins(clause.Has("Company"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		db.Where("?.\"name\" = ?", joinTable, u.Company.Name)
		return nil
	}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Inner JOIN + WHERE with map
	result, err = db.Joins(clause.Has("Company"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		db.Where(map[string]any{"name": u.Company.Name})
		return nil
	}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Left JOIN w/o WHERE
	result, err = db.Joins(clause.LeftJoin.Association("Company"), nil).Where(map[string]any{"name": u.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// Left JOIN + Alias WHERE
	result, err = db.Joins(clause.LeftJoin.Association("Company").As("t"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		if joinTable.Name != "t" {
			t.Fatalf("Join table should be t, but got %v", joinTable.Name)
		}
		db.Where("?.\"name\" = ?", joinTable, u.Company.Name)
		return nil
	}).Where(map[string]any{"name": u.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Joins failed: %v", err)
	}
	if result.Name != u.Name || result.Company.Name != u.Company.Name {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}

	// TODO: Temporarily disabled due to issue with As("t")
	// Raw Subquery JOIN + WHERE
	/*result, err = db.Joins(clause.LeftJoin.AssociationFrom("Company", gorm.G[Company](DB)).As("t"),
		func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
			if joinTable.Name != "t" {
				t.Fatalf("Join table should be t, but got %v", joinTable.Name)
			}
			db.Where("?.\"name\" = ?", joinTable, u.Company.Name)
			return nil
		},
	).Where(map[string]any{"name": u2.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Raw subquery join failed: %v", err)
	}
	if result.Name != u2.Name || result.Company.Name != u.Company.Name || result.Company.ID == 0 {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}*/

	// Raw Subquery JOIN + WHERE + Select
	/*result, err = db.Joins(clause.LeftJoin.AssociationFrom("Company", gorm.G[Company](DB).Select("Name")).As("t"),
		func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
			if joinTable.Name != "t" {
				t.Fatalf("Join table should be t, but got %v", joinTable.Name)
			}
			db.Where("?.\"name\" = ?", joinTable, u.Company.Name)
			return nil
		},
	).Where(map[string]any{"name": u2.Name}).First(ctx)
	if err != nil {
		t.Fatalf("Raw subquery join failed: %v", err)
	}
	if result.Name != u2.Name || result.Company.Name != u.Company.Name || result.Company.ID != 0 {
		t.Fatalf("Joins expected %s, got %+v", u.Name, result)
	}*/

	_, err = db.Joins(clause.Has("Company"), func(db gorm.JoinBuilder, joinTable clause.Table, curTable clause.Table) error {
		return errors.New("join error")
	}).First(ctx)
	if err == nil {
		t.Fatalf("Joins should got error, but got nil")
	}
}

func TestGenericsNestedJoins(t *testing.T) {
	users := []User{
		{
			Name: "generics-nested-joins-1",
			Manager: &User{
				Name: "generics-nested-joins-manager-1",
				Company: Company{
					Name: "generics-nested-joins-manager-company-1",
				},
				NamedPet: &Pet{
					Name: "generics-nested-joins-manager-namepet-1",
					Toy: Toy{
						Name: "generics-nested-joins-manager-namepet-toy-1",
					},
				},
			},
			NamedPet: &Pet{Name: "generics-nested-joins-namepet-1", Toy: Toy{Name: "generics-nested-joins-namepet-toy-1"}},
		},
		{
			Name:     "generics-nested-joins-2",
			Manager:  GetUser("generics-nested-joins-manager-2", Config{Company: true, NamedPet: true}),
			NamedPet: &Pet{Name: "generics-nested-joins-namepet-2", Toy: Toy{Name: "generics-nested-joins-namepet-toy-2"}},
		},
	}

	ctx := context.Background()
	db := gorm.G[User](DB)
	db.CreateInBatches(ctx, &users, 100)

	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	users2, err := db.Joins(clause.LeftJoin.Association("Manager"), nil).
		Joins(clause.LeftJoin.Association("Manager.Company"), nil).
		Joins(clause.LeftJoin.Association("Manager.NamedPet.Toy"), nil).
		Joins(clause.LeftJoin.Association("NamedPet.Toy"), nil).
		Joins(clause.LeftJoin.Association("NamedPet").As("t"), nil).
		Where(map[string]any{"id": userIDs}).Find(ctx)

	if err != nil {
		t.Fatalf("Failed to load with joins, got error: %v", err)
	} else if len(users2) != len(users) {
		t.Fatalf("Failed to load join users, got: %v, expect: %v", len(users2), len(users))
	}

	sort.Slice(users2, func(i, j int) bool {
		return users2[i].ID > users2[j].ID
	})

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID > users[j].ID
	})

	for idx, user := range users {
		// user
		CheckUser(t, user, users2[idx])
		if users2[idx].Manager == nil {
			t.Fatalf("Failed to load Manager")
		}
		// manager
		CheckUser(t, *user.Manager, *users2[idx].Manager)
		// user pet
		if users2[idx].NamedPet == nil {
			t.Fatalf("Failed to load NamedPet")
		}
		CheckPet(t, *user.NamedPet, *users2[idx].NamedPet)
		// manager pet
		if users2[idx].Manager.NamedPet == nil {
			t.Fatalf("Failed to load NamedPet")
		}
		CheckPet(t, *user.Manager.NamedPet, *users2[idx].Manager.NamedPet)
	}
}

func TestGenericsPreloads(t *testing.T) {
	ctx := context.Background()
	db := gorm.G[User](DB)

	u := *GetUser("GenericsPreloads_1", Config{Company: true, Pets: 3, Friends: 7})
	u2 := *GetUser("GenericsPreloads_2", Config{Company: true, Pets: 5, Friends: 5})
	u3 := *GetUser("GenericsPreloads_3", Config{Company: true, Pets: 7, Friends: 3})
	names := []string{u.Name, u2.Name, u3.Name}

	db.CreateInBatches(ctx, &[]User{u3, u, u2}, 10)

	result, err := db.Preload("Company", nil).Preload("Pets", nil).Where("\"name\" = ?", u.Name).First(ctx)
	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}

	if result.Name != u.Name || result.Company.Name != u.Company.Name || len(result.Pets) != len(u.Pets) {
		t.Fatalf("Preload expected %s, got %+v", u.Name, result)
	}

	results, err := db.Preload("Company", func(db gorm.PreloadBuilder) error {
		db.Where("\"name\" = ?", u.Company.Name)
		return nil
	}).Where("\"name\" in ?", names).Find(ctx)
	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}
	for _, result := range results {
		if result.Name == u.Name {
			if result.Company.Name != u.Company.Name {
				t.Fatalf("Preload user %v company should be %v, but got %+v", u.Name, u.Company.Name, result.Company.Name)
			}
		} else if result.Company.Name != "" {
			t.Fatalf("Preload other company should not loaded, user %v company expect %v but got %+v", u.Name, u.Company.Name, result.Company.Name)
		}
	}

	_, err = db.Preload("Company", func(db gorm.PreloadBuilder) error {
		return errors.New("preload error")
	}).Where("\"name\" in ?", names).Find(ctx)
	if err == nil {
		t.Fatalf("Preload should failed, but got nil")
	}

	results, err = db.Preload("Pets", func(db gorm.PreloadBuilder) error {
		db.Select(
			"pets.id",
			"pets.created_at",
			"pets.updated_at",
			"pets.deleted_at",
			"pets.user_id",
			"pets.name",
		)
		db.LimitPerRecord(5)
		return nil
	}).Where("\"name\" in ?", names).Find(ctx)

	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}
	for _, result := range results {
		if result.Name == u.Name {
			if len(result.Pets) != len(u.Pets) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
		} else if len(result.Pets) != 5 {
			t.Fatalf("Preload user %v pets should be 5, but got %+v", result.Name, result.Pets)
		}
	}

	results, err = db.Preload("Pets", func(db gorm.PreloadBuilder) error {
		db.Select(
			"pets.id",
			"pets.created_at",
			"pets.updated_at",
			"pets.deleted_at",
			"pets.user_id",
			"pets.name",
		)
		db.Order("\"name\" desc").LimitPerRecord(5)
		return nil
	}).Where("\"name\" in ?", names).Find(ctx)

	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}
	for _, result := range results {
		if result.Name == u.Name {
			if len(result.Pets) != len(u.Pets) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
		} else if len(result.Pets) != 5 {
			t.Fatalf("Preload user %v pets should be 5, but got %+v", result.Name, result.Pets)
		}
		for i := 1; i < len(result.Pets); i++ {
			if result.Pets[i-1].Name < result.Pets[i].Name {
				t.Fatalf("Preload user %v pets not ordered correctly, last %v, cur %v", result.Name, result.Pets[i-1], result.Pets[i])
			}
		}
	}

	results, err = db.Preload("Pets", func(db gorm.PreloadBuilder) error {
		db.Select(
			"pets.id",
			"pets.created_at",
			"pets.updated_at",
			"pets.deleted_at",
			"pets.user_id",
			"pets.name",
		)
		db.Order("\"name\"").LimitPerRecord(5)
		return nil
	}).Preload("Friends", func(db gorm.PreloadBuilder) error {
		db.Order("\"name\"")
		return nil
	}).Where("\"name\" in ?", names).Find(ctx)

	if err != nil {
		t.Fatalf("Preload failed: %v", err)
	}
	for _, result := range results {
		if result.Name == u.Name {
			if len(result.Pets) != len(u.Pets) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
			if len(result.Friends) != len(u.Friends) {
				t.Fatalf("Preload user %v pets should be %v, but got %+v", u.Name, u.Pets, result.Pets)
			}
		} else if len(result.Pets) != 5 || len(result.Friends) == 0 {
			t.Fatalf("Preload user %v pets should be 5, but got %+v", result.Name, result.Pets)
		}
		for i := 1; i < len(result.Pets); i++ {
			if result.Pets[i-1].Name > result.Pets[i].Name {
				t.Fatalf("Preload user %v pets not ordered correctly, last %v, cur %v", result.Name, result.Pets[i-1], result.Pets[i])
			}
		}
		for i := 1; i < len(result.Pets); i++ {
			if result.Pets[i-1].Name > result.Pets[i].Name {
				t.Fatalf("Preload user %v friends not ordered correctly, last %v, cur %v", result.Name, result.Pets[i-1], result.Pets[i])
			}
		}
	}
}

func TestGenericsNestedPreloads(t *testing.T) {
	user := *GetUser("generics_nested_preload", Config{Pets: 2})
	user.Friends = []*User{GetUser("generics_nested_preload", Config{Pets: 5})}

	ctx := context.Background()
	db := gorm.G[User](DB)

	for idx, pet := range user.Pets {
		pet.Toy = Toy{Name: "toy_nested_preload_" + strconv.Itoa(idx+1)}
	}

	if err := db.Create(ctx, &user); err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user2, err := db.Preload("Pets.Toy", nil).Preload("Friends.Pets", func(db gorm.PreloadBuilder) error {
		return nil
	}).Where(user.ID).Take(ctx)
	if err != nil {
		t.Errorf("failed to nested preload user")
	}
	CheckUser(t, user2, user)
	if len(user.Pets) == 0 || len(user.Friends) == 0 || len(user.Friends[0].Pets) == 0 {
		t.Fatalf("failed to nested preload")
	}

	user3, err := db.Preload("Pets.Toy", nil).Preload("Friends.Pets", func(db gorm.PreloadBuilder) error {
		db.Select(
			"pets.id",
			"pets.created_at",
			"pets.updated_at",
			"pets.deleted_at",
			"pets.user_id",
			"pets.name",
		)
		db.LimitPerRecord(3)
		return nil
	}).Where(user.ID).Take(ctx)
	if err != nil {
		t.Errorf("failed to nested preload user")
	}
	CheckUser(t, user3, user)

	if len(user3.Friends) != 1 || len(user3.Friends[0].Pets) != 3 {
		t.Errorf("failed to nested preload with limit per record")
	}
}

func TestGenericsDistinct(t *testing.T) {
	ctx := context.Background()

	batch := []User{
		{Name: "GenericsDistinctDup"},
		{Name: "GenericsDistinctDup"},
		{Name: "GenericsDistinctUnique"},
	}
	if err := gorm.G[User](DB).CreateInBatches(ctx, &batch, len(batch)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	results, err := gorm.G[User](DB).Where("\"name\" like ?", "GenericsDistinct%").Distinct("name").Find(ctx)
	if err != nil {
		t.Fatalf("Distinct Find failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 distinct names, got %d", len(results))
	}

	var names []string
	for _, u := range results {
		names = append(names, u.Name)
	}
	sort.Strings(names)
	expected := []string{"GenericsDistinctDup", "GenericsDistinctUnique"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected names %v, got %v", expected, names)
	}
}

func TestGenericsGroupHaving(t *testing.T) {
	ctx := context.Background()

	batch := []User{
		{Name: "GenericsGroupHavingMulti"},
		{Name: "GenericsGroupHavingMulti"},
		{Name: "GenericsGroupHavingSingle"},
	}
	if err := gorm.G[User](DB).CreateInBatches(ctx, &batch, len(batch)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	grouped, err := gorm.G[User](DB).Select("name").Where("\"name\" like ?", "GenericsGroupHaving%").Group("name").Having("COUNT(\"id\") > ?", 1).Find(ctx)
	if err != nil {
		t.Fatalf("Group+Having Find failed: %v", err)
	}

	if len(grouped) != 1 {
		t.Errorf("expected 1 group with count>1, got %d", len(grouped))
	} else if grouped[0].Name != "GenericsGroupHavingMulti" {
		t.Errorf("expected group name 'GenericsGroupHavingMulti', got '%s'", grouped[0].Name)
	}
}

func TestGenericsSubQuery(t *testing.T) {
	ctx := context.Background()
	users := []User{
		{Name: "GenericsSubquery_1", Age: 10},
		{Name: "GenericsSubquery_2", Age: 20},
		{Name: "GenericsSubquery_3", Age: 30},
		{Name: "GenericsSubquery_4", Age: 40},
	}

	if err := gorm.G[User](DB).CreateInBatches(ctx, &users, len(users)); err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	results, err := gorm.G[User](DB).Where("\"name\" IN (?)", gorm.G[User](DB).Select("name").Where("\"name\" LIKE ?", "GenericsSubquery%")).Find(ctx)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Four users should be found, instead found %d", len(results))
	}

	results, err = gorm.G[User](DB).Where("\"name\" IN (?)", gorm.G[User](DB).Select("name").Where("\"name\" IN ?", []string{"GenericsSubquery_1", "GenericsSubquery_2"}).Or("\"name\" = ?", "GenericsSubquery_3")).Find(ctx)
	if err != nil {
		t.Fatalf("got error: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Three users should be found, instead found %d", len(results))
	}
}

func TestGenericsUpsert(t *testing.T) {
	ctx := context.Background()
	lang := Language{Code: "upsert", Name: "Upsert"}

	if err := gorm.G[Language](DB, clause.OnConflict{DoNothing: true}).Create(ctx, &lang); err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	lang2 := Language{Code: "upsert", Name: "Upsert"}
	if err := gorm.G[Language](DB, clause.OnConflict{DoNothing: true}).Create(ctx, &lang2); err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	langs, err := gorm.G[Language](DB).Where("\"code\" = ?", lang.Code).Find(ctx)
	if err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs) != 1 {
		t.Errorf("should only find only 1 languages, but got %+v", langs)
	}

	lang3 := Language{Code: "upsert", Name: "Upsert"}
	if err := gorm.G[Language](DB, clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.Assignments(map[string]interface{}{"name": "upsert-new"}),
	}).Create(ctx, &lang3); err != nil {
		t.Fatalf("failed to upsert, got %v", err)
	}

	if langs, err := gorm.G[Language](DB).Where("\"code\" = ?", lang.Code).Find(ctx); err != nil {
		t.Errorf("no error should happen when find languages with code, but got %v", err)
	} else if len(langs) != 1 {
		t.Errorf("should only find only 1 languages, but got %+v", langs)
	} else if langs[0].Name != "upsert-new" {
		t.Errorf("should update name on conflict, but got name %+v", langs[0].Name)
	}
}

func TestGenericsWithResult(t *testing.T) {
	ctx := context.Background()
	users := []User{{Name: "TestGenericsWithResult", Age: 18}, {Name: "TestGenericsWithResult2", Age: 18}}

	result := gorm.WithResult()
	err := gorm.G[User](DB, result).CreateInBatches(ctx, &users, 2)
	if err != nil {
		t.Errorf("failed to create users WithResult")
	}

	if result.RowsAffected != 2 {
		t.Errorf("failed to get affected rows, got %d, should be %d", result.RowsAffected, 2)
	}
}

func TestGenericsReuse(t *testing.T) {
	ctx := context.Background()
	users := []User{{Name: "TestGenericsReuse1", Age: 18}, {Name: "TestGenericsReuse2", Age: 18}}

	err := gorm.G[User](DB).CreateInBatches(ctx, &users, 2)
	if err != nil {
		t.Errorf("failed to create users")
	}

	reusedb := gorm.G[User](DB).Where("\"name\" like ?", "TestGenericsReuse%")

	sg := sync.WaitGroup{}
	for i := 0; i < 5; i++ {
		sg.Add(1)

		go func() {
			if u1, err := reusedb.Where("\"id\" = ?", users[0].ID).First(ctx); err != nil {
				t.Errorf("failed to find user, got error: %v", err)
			} else if u1.Name != users[0].Name || u1.ID != users[0].ID {
				t.Errorf("found invalid user, got %v, expect %v", u1, users[0])
			}

			if u2, err := reusedb.Where("\"id\" = ?", users[1].ID).First(ctx); err != nil {
				t.Errorf("failed to find user, got error: %v", err)
			} else if u2.Name != users[1].Name || u2.ID != users[1].ID {
				t.Errorf("found invalid user, got %v, expect %v", u2, users[1])
			}

			if users, err := reusedb.Where("\"id\" IN ?", []uint{users[0].ID, users[1].ID}).Find(ctx); err != nil {
				t.Errorf("failed to find user, got error: %v", err)
			} else if len(users) != 2 {
				t.Errorf("should find 2 users, but got %d", len(users))
			}
			sg.Done()
		}()
	}
	sg.Wait()
}

func TestGenericsWithTransaction(t *testing.T) {
	ctx := context.Background()
	tx := DB.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin transaction: %v", tx.Error)
	}

	users := []User{{Name: "TestGenericsTransaction", Age: 18}, {Name: "TestGenericsTransaction2", Age: 18}}
	err := gorm.G[User](tx).CreateInBatches(ctx, &users, 2)
	if err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}

	count, err := gorm.G[User](tx).Where("\"name\" like ?", "TestGenericsTransaction%").Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 records, got %d", count)
	}

	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("failed to rollback transaction: %v", err)
	}

	count2, err := gorm.G[User](DB).Where("\"name\" like ?", "TestGenericsTransaction%").Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count2 != 0 {
		t.Errorf("expected 0 records after rollback, got %d", count2)
	}
}

func TestGenericsToSQL(t *testing.T) {
	ctx := context.Background()
	sql := DB.ToSQL(func(tx *gorm.DB) *gorm.DB {
		gorm.G[User](tx).Limit(10).Find(ctx)
		return tx
	})

	if !regexp.MustCompile(`SELECT \* FROM .users..* 10`).MatchString(sql) {
		t.Errorf("ToSQL: got wrong sql with Generics API %v", sql)
	}
}

func TestGenericsCountOmitSelect(t *testing.T) {
	ctx := context.Background()
	users := []User{{Name: "GenericsCount1", Age: 5}, {Name: "GenericsCount2", Age: 7}}
	err := gorm.G[User](DB).CreateInBatches(ctx, &users, 2)
	if err != nil {
		t.Fatalf("CreateInBatches failed: %v", err)
	}
	count, err := gorm.G[User](DB).Omit("age").Where("\"name\" LIKE ?", "GenericsCount%").Count(ctx, "*")
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Count with Omit: expected 2, got %d", count)
	}
}

func TestGenericsSelectAndOmitFind(t *testing.T) {
	ctx := context.Background()
	u := User{Name: "GenericsSelectOmit", Age: 30}
	err := gorm.G[User](DB).Create(ctx, &u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result, err := gorm.G[User](DB).Select("id").Omit("age").Where("\"name\" = ?", u.Name).First(ctx)
	if err != nil {
		t.Fatalf("Select and Omit Find failed: %v", err)
	}
	if result.ID != u.ID || result.Name != "" || result.Age != 0 {
		t.Errorf("SelectAndOmitFind expects partial zero values, got: %+v", result)
	}
}

func TestGenericsSelectWithPreloadAssociations(t *testing.T) {
	ctx := context.Background()
	user := User{Name: "SelectPreloadCombo", Age: 40, Company: Company{Name: "ComboCompany"}}
	for i := 1; i <= 2; i++ {
		user.Pets = append(user.Pets, &Pet{Name: fmt.Sprintf("Pet-%d", i)})
	}
	if err := gorm.G[User](DB).Create(ctx, &user); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	result, err := gorm.G[User](DB).Select("id", "name", "company_id").Preload("Company", nil).Preload("Pets", nil).Where("\"name\" = ?", user.Name).First(ctx)
	if err != nil {
		t.Fatalf("Select+Preload First failed: %v", err)
	}
	if result.ID == 0 || result.Name == "" {
		t.Errorf("Expected user id/name; got: %+v", result)
	}
	if result.Age != 0 {
		t.Errorf("Expected omitted Age=0, got %d", result.Age)
	}
	if result.Company.Name != user.Company.Name {
		t.Errorf("Expected company %+v, got %+v", user.Company, result.Company)
	}
	if len(result.Pets) != len(user.Pets) {
		t.Errorf("Expected %d pets, got %d", len(user.Pets), len(result.Pets))
	}
}

func TestGenericsTransactionRollbackOnPreloadError(t *testing.T) {
	ctx := context.Background()
	tx := DB.Begin()
	if tx.Error != nil {
		t.Fatalf("Failed to begin transaction: %v", tx.Error)
	}
	user := User{Name: "TxRollbackPreload", Age: 25, Company: Company{Name: "TxCompany"}}
	if err := gorm.G[User](tx).Create(ctx, &user); err != nil {
		_ = tx.Rollback()
		t.Fatalf("Failed to create user in tx: %v", err)
	}
	var gotErr error
	_, gotErr = gorm.G[User](tx).
		Preload("Company", func(db gorm.PreloadBuilder) error { return fmt.Errorf("bad preload") }).
		Where("\"name\" = ?", user.Name).
		First(ctx)
	errRollback := tx.Rollback().Error
	if gotErr == nil {
		t.Errorf("Expected preload error, got nil")
	}
	if errRollback != nil {
		t.Fatalf("Failed to rollback on Preload error: %v", errRollback)
	}
	_, err := gorm.G[User](DB).Where("\"name\" = ?", user.Name).First(ctx)
	if err == nil {
		t.Errorf("Expected no user after rollback, but found one")
	}
}
