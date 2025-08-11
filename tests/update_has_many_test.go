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

	. "github.com/oracle/gorm-oracle/tests/utils"

	"gorm.io/gorm"
)

func TestUpdateHasManyAssociations(t *testing.T) {
	user := *GetUser("update-has-many", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Pets = []*Pet{{Name: "pet1"}, {Name: "pet2"}}
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Pets").Find(&user2, "\"id\" = ?", user.ID)
	CheckUserSkipUpdatedAt(t, user2, user)

	for _, pet := range user.Pets {
		pet.Name += "new"
	}

	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user3 User
	DB.Preload("Pets").Find(&user3, "\"id\" = ?", user.ID)
	CheckUserSkipUpdatedAt(t, user2, user3)

	if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user4 User
	DB.Preload("Pets").Find(&user4, "\"id\" = ?", user.ID)
	CheckUserSkipUpdatedAt(t, user4, user)

	t.Run("Polymorphic", func(t *testing.T) {
		user := *GetUser("update-has-many", Config{})

		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Toys = []Toy{{Name: "toy1"}, {Name: "toy2"}}
		if err := DB.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var user2 User
		DB.Preload("Toys").Find(&user2, "\"id\" = ?", user.ID)
		CheckUserSkipUpdatedAt(t, user2, user)

		for idx := range user.Toys {
			user.Toys[idx].Name += "new"
		}

		if err := DB.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var user3 User
		DB.Preload("Toys").Find(&user3, "\"id\" = ?", user.ID)
		CheckUserSkipUpdatedAt(t, user2, user3)

		if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var user4 User
		DB.Preload("Toys").Find(&user4, "\"id\" = ?", user.ID)
		CheckUserSkipUpdatedAt(t, user4, user)
	})
}
