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
	"gorm.io/gorm/clause"
)

type Person struct {
	ID        int
	Name      string
	Addresses []Address `gorm:"many2many:person_addresses;"`
	DeletedAt gorm.DeletedAt
}

type Address struct {
	ID   uint
	Name string
}

type PersonAddress struct {
	PersonID  int
	AddressID int
	CreatedAt time.Time
	DeletedAt gorm.DeletedAt
}

func TestOverrideJoinTable(t *testing.T) {
	DB.Migrator().DropTable(&Person{}, &Address{}, &PersonAddress{})

	if err := DB.SetupJoinTable(&Person{}, "Addresses", &PersonAddress{}); err != nil {
		t.Fatalf("Failed to setup join table for person, got error %v", err)
	}

	if err := DB.AutoMigrate(&Person{}, &Address{}); err != nil {
		t.Fatalf("Failed to migrate, got %v", err)
	}

	address1 := Address{Name: "address 1"}
	address2 := Address{Name: "address 2"}
	person := Person{Name: "person", Addresses: []Address{address1, address2}}
	DB.Create(&person)

	var addresses1 []Address
	if err := DB.Model(&person).Association("Addresses").Find(&addresses1); err != nil || len(addresses1) != 2 {
		t.Fatalf("Failed to find address, got error %v, length: %v", err, len(addresses1))
	}

	if err := DB.Model(&person).Association("Addresses").Delete(&person.Addresses[0]); err != nil {
		t.Fatalf("Failed to delete address, got error %v", err)
	}

	if len(person.Addresses) != 1 {
		t.Fatalf("Should have one address left")
	}

	if DB.Find(&[]PersonAddress{}, "\"person_id\" = ?", person.ID).RowsAffected != 1 {
		t.Fatalf("Should found one address")
	}

	var addresses2 []Address
	if err := DB.Model(&person).Association("Addresses").Find(&addresses2); err != nil || len(addresses2) != 1 {
		t.Fatalf("Failed to find address, got error %v, length: %v", err, len(addresses2))
	}

	if DB.Model(&person).Association("Addresses").Count() != 1 {
		t.Fatalf("Should found one address")
	}

	var addresses3 []Address
	if err := DB.Unscoped().Model(&person).Association("Addresses").Find(&addresses3); err != nil || len(addresses3) != 2 {
		t.Fatalf("Failed to find address, got error %v, length: %v", err, len(addresses3))
	}

	if DB.Unscoped().Find(&[]PersonAddress{}, "\"person_id\" = ?", person.ID).RowsAffected != 2 {
		t.Fatalf("Should found soft deleted addresses with unscoped")
	}

	if DB.Unscoped().Model(&person).Association("Addresses").Count() != 2 {
		t.Fatalf("Should found soft deleted addresses with unscoped")
	}

	DB.Model(&person).Association("Addresses").Clear()

	if DB.Model(&person).Association("Addresses").Count() != 0 {
		t.Fatalf("Should deleted all addresses")
	}

	if DB.Unscoped().Model(&person).Association("Addresses").Count() != 2 {
		t.Fatalf("Should found soft deleted addresses with unscoped")
	}

	DB.Unscoped().Model(&person).Association("Addresses").Clear()

	if DB.Unscoped().Model(&person).Association("Addresses").Count() != 0 {
		t.Fatalf("address should be deleted when clear with unscoped")
	}

	address2_1 := Address{Name: "address 2-1"}
	address2_2 := Address{Name: "address 2-2"}
	person2 := Person{Name: "person_2", Addresses: []Address{address2_1, address2_2}}
	DB.Create(&person2)
	if err := DB.Select(clause.Associations).Delete(&person2).Error; err != nil {
		t.Fatalf("failed to delete person, got error: %v", err)
	}

	if count := DB.Unscoped().Model(&person2).Association("Addresses").Count(); count != 2 {
		t.Errorf("person's addresses expects 2, got %v", count)
	}

	if count := DB.Model(&person2).Association("Addresses").Count(); count != 0 {
		t.Errorf("person's addresses expects 2, got %v", count)
	}
}

func TestOverrideJoinTableInvalidAssociation(t *testing.T) {
	DB.Migrator().DropTable(&Person{}, &Address{}, &PersonAddress{})
	if err := DB.SetupJoinTable(&Person{}, "Addresses", &PersonAddress{}); err != nil {
		t.Fatalf("Failed to setup join table for person, got error %v", err)
	}
	if err := DB.AutoMigrate(&Person{}, &Address{}); err != nil {
		t.Fatalf("Failed to migrate, got %v", err)
	}

	person := Person{Name: "invalid-assoc"}
	DB.Create(&person)

	err := DB.Model(&person).Association("NonExistent").Find(&[]Address{})
	if err == nil {
		t.Fatalf("Expected error when accessing non-existent association, got nil")
	}
}

func TestOverrideJoinTableClearWithoutAssociations(t *testing.T) {
	DB.Migrator().DropTable(&Person{}, &Address{}, &PersonAddress{})
	if err := DB.SetupJoinTable(&Person{}, "Addresses", &PersonAddress{}); err != nil {
		t.Fatalf("Failed to setup join table for person, got error %v", err)
	}
	if err := DB.AutoMigrate(&Person{}, &Address{}); err != nil {
		t.Fatalf("Failed to migrate, got %v", err)
	}

	person := Person{Name: "no-assoc"}
	DB.Create(&person)

	if err := DB.Model(&person).Association("Addresses").Clear(); err != nil {
		t.Fatalf("Expected no error clearing empty associations, got %v", err)
	}

	if count := DB.Model(&person).Association("Addresses").Count(); count != 0 {
		t.Fatalf("Expected 0 associations, got %v", count)
	}
}

func TestOverrideJoinTableDeleteNonExistentAssociation(t *testing.T) {
	DB.Migrator().DropTable(&Person{}, &Address{}, &PersonAddress{})
	if err := DB.SetupJoinTable(&Person{}, "Addresses", &PersonAddress{}); err != nil {
		t.Fatalf("Failed to setup join table for person, got error %v", err)
	}
	if err := DB.AutoMigrate(&Person{}, &Address{}); err != nil {
		t.Fatalf("Failed to migrate, got %v", err)
	}

	address := Address{Name: "non-existent"}
	person := Person{Name: "test-delete"}
	DB.Create(&person)

	if err := DB.Model(&person).Association("Addresses").Delete(&address); err != nil {
		t.Fatalf("Expected no error when deleting non-existent association, got %v", err)
	}
}
