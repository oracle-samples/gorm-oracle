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
	"sort"
	"strconv"
	"strings"
	"testing"

	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

type Config struct {
	Account   bool
	Pets      int
	Toys      int
	Company   bool
	Manager   bool
	Team      int
	Languages int
	Friends   int
	NamedPet  bool
	Tools     int
}

func GetUser(name string, config Config) *User {
	var (
		birthday = time.Now().Round(time.Second)
		user     = User{
			Name:     name,
			Age:      18,
			Birthday: &birthday,
		}
	)

	if config.Account {
		user.Account = Account{AccountNumber: name + "_account"}
	}

	for i := 0; i < config.Pets; i++ {
		user.Pets = append(user.Pets, &Pet{Name: name + "_pet_" + strconv.Itoa(i+1)})
	}

	for i := 0; i < config.Toys; i++ {
		user.Toys = append(user.Toys, Toy{Name: name + "_toy_" + strconv.Itoa(i+1)})
	}

	for i := 0; i < config.Tools; i++ {
		user.Tools = append(user.Tools, Tools{Name: name + "_tool_" + strconv.Itoa(i+1)})
	}

	if config.Company {
		user.Company = Company{Name: "company-" + name}
	}

	if config.Manager {
		user.Manager = GetUser(name+"_manager", Config{})
	}

	for i := 0; i < config.Team; i++ {
		user.Team = append(user.Team, *GetUser(name+"_team_"+strconv.Itoa(i+1), Config{}))
	}

	for i := 0; i < config.Languages; i++ {
		name := name + "_locale_" + strconv.Itoa(i+1)
		language := Language{Code: name, Name: name}
		user.Languages = append(user.Languages, language)
	}

	for i := 0; i < config.Friends; i++ {
		user.Friends = append(user.Friends, GetUser(name+"_friend_"+strconv.Itoa(i+1), Config{}))
	}

	if config.NamedPet {
		user.NamedPet = &Pet{Name: name + "_namepet"}
	}

	return &user
}

func CheckPetUnscoped(t *testing.T, pet Pet, expect Pet) {
	doCheckPet(t, pet, expect, true, false)
}

func CheckPet(t *testing.T, pet Pet, expect Pet) {
	doCheckPet(t, pet, expect, false, false)
}

func CheckPetSkipUpdatedAt(t *testing.T, pet Pet, expect Pet) {
	doCheckPet(t, pet, expect, false, true)
}

func doCheckPet(t *testing.T, pet Pet, expect Pet, unscoped bool, skipUpdatedAt bool) {
	var (
		petFields = []string{"ID", "CreatedAt", "DeletedAt", "UserID", "Name"}
		toyFields = []string{"ID", "CreatedAt", "DeletedAt", "Name", "OwnerID", "OwnerType"}
	)

	if !skipUpdatedAt {
		for _, fields := range []*[]string{&petFields, &toyFields} {
			*fields = append(*fields, "UpdatedAt")
		}
	}

	if pet.ID != 0 {
		var newPet Pet
		if err := db(unscoped).Where("\"id\" = ?", pet.ID).First(&newPet).Error; err != nil {
			t.Fatalf("errors happened when query: %v", err)
		} else {
			tests.AssertObjEqual(t, newPet, pet, petFields...)
			tests.AssertObjEqual(t, newPet, expect, petFields...)
		}

		tests.AssertObjEqual(t, pet, expect, "ID", "CreatedAt", "DeletedAt", "UserID", "Name")

		tests.AssertObjEqual(t, pet.Toy, expect.Toy, "ID", "CreatedAt", "DeletedAt", "Name", "OwnerID", "OwnerType")
	} else {
		if pet.ID != 0 {
			var newPet Pet
			if err := db(unscoped).Where("\"id\" = ?", pet.ID).First(&newPet).Error; err != nil {
				t.Fatalf("errors happened when query: %v", err)
			} else {
				tests.AssertObjEqual(t, newPet, pet, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "UserID", "Name")
				tests.AssertObjEqual(t, newPet, expect, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "UserID", "Name")
			}
		}

		tests.AssertObjEqual(t, pet, expect, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "UserID", "Name")

		tests.AssertObjEqual(t, pet.Toy, expect.Toy, "ID", "CreatedAt", "UpdatedAt", "DeletedAt", "Name", "OwnerID", "OwnerType")
	}

	tests.AssertObjEqual(t, pet, expect, petFields...)

	tests.AssertObjEqual(t, pet.Toy, expect.Toy, toyFields...)

	expectedOwnerType := "pets"
	if expect.Toy.Name != "" && expect.Toy.OwnerType != expectedOwnerType {
		t.Errorf("toys's OwnerType, expect: %v, got %v", expectedOwnerType, expect.Toy.OwnerType)
	}
}

func CheckUserUnscoped(t *testing.T, user User, expect User) {
	doCheckUser(t, user, expect, true, false)
}

func CheckUser(t *testing.T, user User, expect User) {
	doCheckUser(t, user, expect, false, false)
}

func CheckUserSkipUpdatedAt(t *testing.T, user User, expect User) {
	doCheckUser(t, user, expect, false, true)
}

func doCheckUser(t *testing.T, user User, expect User, unscoped bool, skipUpdatedAt bool) {
	var (
		userFields = []string{"ID", "CreatedAt", "DeletedAt", "Name", "Age", "Birthday", "CompanyID",
			"ManagerID", "Active"}
		accountFields = []string{"ID", "CreatedAt", "DeletedAt", "UserID", "AccountNumber"}
		toyFields     = []string{"ID", "CreatedAt", "Name", "OwnerID", "OwnerType"}
	)

	if !skipUpdatedAt {
		for _, fields := range []*[]string{&userFields, &accountFields, &toyFields} {
			*fields = append(*fields, "UpdatedAt")
		}
	}

	if user.ID != 0 {
		var newUser User
		if err := db(unscoped).Where("\"id\" = ?", user.ID).First(&newUser).Error; err != nil {
			t.Fatalf("errors happened when query: %v", err)
		} else {
			tests.AssertObjEqual(t, newUser, user, userFields...)
		}
	}

	tests.AssertObjEqual(t, user, expect, userFields...)

	t.Run("Account", func(t *testing.T) {
		tests.AssertObjEqual(t, user.Account, expect.Account, accountFields...)

		if user.Account.AccountNumber != "" {
			if !user.Account.UserID.Valid {
				t.Errorf("Account's foreign key should be saved")
			} else {
				var account Account
				db(unscoped).First(&account, "\"user_id\" = ?", user.ID)
				tests.AssertObjEqual(t, account, user.Account, accountFields...)
			}
		}
	})

	t.Run("Pets", func(t *testing.T) {
		if len(user.Pets) != len(expect.Pets) {
			t.Fatalf("pets should equal, expect: %v, got %v", len(expect.Pets), len(user.Pets))
		}

		sort.Slice(user.Pets, func(i, j int) bool {
			return user.Pets[i].ID > user.Pets[j].ID
		})

		sort.Slice(expect.Pets, func(i, j int) bool {
			return expect.Pets[i].ID > expect.Pets[j].ID
		})

		for idx, pet := range user.Pets {
			if pet == nil || expect.Pets[idx] == nil {
				t.Errorf("pets#%v should equal, expect: %v, got %v", idx, expect.Pets[idx], pet)
			} else {
				doCheckPet(t, *pet, *expect.Pets[idx], unscoped, skipUpdatedAt)
			}
		}
	})

	t.Run("Toys", func(t *testing.T) {
		if len(user.Toys) != len(expect.Toys) {
			t.Fatalf("toys should equal, expect: %v, got %v", len(expect.Toys), len(user.Toys))
		}

		sort.Slice(user.Toys, func(i, j int) bool {
			return user.Toys[i].ID > user.Toys[j].ID
		})

		sort.Slice(expect.Toys, func(i, j int) bool {
			return expect.Toys[i].ID > expect.Toys[j].ID
		})

		for idx, toy := range user.Toys {
			if toy.OwnerType != "users" {
				t.Errorf("toys's OwnerType, expect: %v, got %v", "users", toy.OwnerType)
			}

			tests.AssertObjEqual(t, toy, expect.Toys[idx], toyFields...)
		}
	})

	t.Run("Company", func(t *testing.T) {
		tests.AssertObjEqual(t, user.Company, expect.Company, "ID", "Name")
	})

	t.Run("Manager", func(t *testing.T) {
		if user.Manager != nil {
			if user.ManagerID == nil {
				t.Errorf("Manager's foreign key should be saved")
			} else {
				var manager User
				db(unscoped).First(&manager, "\"id\" = ?", *user.ManagerID)
				tests.AssertObjEqual(t, manager, user.Manager, userFields...)
				tests.AssertObjEqual(t, manager, expect.Manager, userFields...)
			}
		} else if user.ManagerID != nil {
			t.Errorf("Manager should not be created for zero value, got: %+v", user.ManagerID)
		}
	})

	t.Run("Team", func(t *testing.T) {
		if len(user.Team) != len(expect.Team) {
			t.Fatalf("Team should equal, expect: %v, got %v", len(expect.Team), len(user.Team))
		}

		sort.Slice(user.Team, func(i, j int) bool {
			return user.Team[i].ID > user.Team[j].ID
		})

		sort.Slice(expect.Team, func(i, j int) bool {
			return expect.Team[i].ID > expect.Team[j].ID
		})

		for idx, team := range user.Team {
			tests.AssertObjEqual(t, team, expect.Team[idx], userFields...)
		}
	})

	t.Run("Languages", func(t *testing.T) {
		if len(user.Languages) != len(expect.Languages) {
			t.Fatalf("Languages should equal, expect: %v, got %v", len(expect.Languages), len(user.Languages))
		}

		sort.Slice(user.Languages, func(i, j int) bool {
			return strings.Compare(user.Languages[i].Code, user.Languages[j].Code) > 0
		})

		sort.Slice(expect.Languages, func(i, j int) bool {
			return strings.Compare(expect.Languages[i].Code, expect.Languages[j].Code) > 0
		})
		for idx, language := range user.Languages {
			tests.AssertObjEqual(t, language, expect.Languages[idx], "Code", "Name")
		}
	})

	t.Run("Friends", func(t *testing.T) {
		if len(user.Friends) != len(expect.Friends) {
			t.Fatalf("Friends should equal, expect: %v, got %v", len(expect.Friends), len(user.Friends))
		}

		sort.Slice(user.Friends, func(i, j int) bool {
			return user.Friends[i].ID > user.Friends[j].ID
		})

		sort.Slice(expect.Friends, func(i, j int) bool {
			return expect.Friends[i].ID > expect.Friends[j].ID
		})

		for idx, friend := range user.Friends {
			tests.AssertObjEqual(t, friend, expect.Friends[idx], userFields...)
		}
	})
}

func db(unscoped bool) *gorm.DB {
	if unscoped {
		return DB.Unscoped()
	} else {
		return DB
	}
}
