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
	"strings"
	"testing"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

func AssertAssociationCount(t *testing.T, data interface{}, name string, result int64, reason string) {
	if count := DB.Model(data).Association(name).Count(); count != result {
		t.Fatalf("invalid %v count %v, expects: %v got %v", name, reason, result, count)
	}

	var newUser User
	if user, ok := data.(User); ok {
		DB.Find(&newUser, "\"id\" = ?", user.ID)
	} else if user, ok := data.(*User); ok {
		DB.Find(&newUser, "\"id\" = ?", user.ID)
	}

	if newUser.ID != 0 {
		if count := DB.Model(&newUser).Association(name).Count(); count != result {
			t.Fatalf("invalid %v count %v, expects: %v got %v", name, reason, result, count)
		}
	}
}

func TestInvalidAssociation(t *testing.T) {
	user := *GetUser("invalid", Config{Company: true, Manager: true})
	if err := DB.Model(&user).Association("Invalid").Find(&user.Company).Error; err == nil {
		t.Fatalf("should return errors for invalid association, but got nil")
	}
}

func TestAssociationNotNullClear(t *testing.T) {
	type Profile struct {
		gorm.Model
		Number   string
		MemberID uint `gorm:"not null"`
	}

	type Member struct {
		gorm.Model
		Profiles []Profile
	}

	DB.Migrator().DropTable(&Member{}, &Profile{})

	if err := DB.AutoMigrate(&Member{}, &Profile{}); err != nil {
		t.Fatalf("Failed to migrate, got error: %v", err)
	}

	member := &Member{
		Profiles: []Profile{{
			Number: "1",
		}, {
			Number: "2",
		}},
	}

	if err := DB.Create(&member).Error; err != nil {
		t.Fatalf("Failed to create test data, got error: %v", err)
	}

	if err := DB.Model(member).Association("Profiles").Clear(); err == nil {
		t.Fatalf("No error occurred during clearind not null association")
	}
}

func TestForeignKeyConstraints(t *testing.T) {
	type Profile struct {
		ID       uint
		Name     string
		MemberID uint
	}

	type Member struct {
		ID      uint
		Refer   uint `gorm:"unique"`
		Name    string
		Profile Profile `gorm:"Constraint:OnUpdate:CASCADE,OnDelete:CASCADE;FOREIGNKEY:MemberID;References:Refer"`
	}

	DB.Migrator().DropTable(&Profile{}, &Member{})

	if err := DB.AutoMigrate(&Profile{}, &Member{}); err != nil {
		t.Fatalf("Failed to migrate, got error: %v", err)
	}

	member := Member{Refer: 1, Name: "foreign_key_constraints", Profile: Profile{Name: "my_profile"}}

	DB.Create(&member)

	var profile Profile
	if err := DB.First(&profile, "\"id\" = ?", member.Profile.ID).Error; err != nil {
		t.Fatalf("failed to find profile, got error: %v", err)
	} else if profile.MemberID != member.ID {
		t.Fatalf("member id is not equal: expects: %v, got: %v", member.ID, profile.MemberID)
	}

	member.Profile = Profile{}
	DB.Model(&member).Update("Refer", 100)

	var profile2 Profile
	if err := DB.First(&profile2, "\"id\" = ?", profile.ID).Error; err != nil {
		t.Fatalf("failed to find profile, got error: %v", err)
	} else if profile2.MemberID != 100 {
		t.Fatalf("member id is not equal: expects: %v, got: %v", 100, profile2.MemberID)
	}

	if r := DB.Delete(&member); r.Error != nil || r.RowsAffected != 1 {
		t.Fatalf("Should delete member, got error: %v, affected: %v", r.Error, r.RowsAffected)
	}

	var result Member
	if err := DB.First(&result, member.ID).Error; err == nil {
		t.Fatalf("Should not find deleted member")
	}

	if err := DB.First(&profile2, profile.ID).Error; err == nil {
		t.Fatalf("Should not find deleted profile")
	}
}

func TestForeignKeyConstraintsBelongsTo(t *testing.T) {
	type Profile struct {
		ID    uint
		Name  string
		Refer uint `gorm:"unique"`
	}

	type Member struct {
		ID        uint
		Name      string
		ProfileID uint
		Profile   Profile `gorm:"Constraint:OnUpdate:CASCADE,OnDelete:CASCADE;FOREIGNKEY:ProfileID;References:Refer"`
	}

	DB.Migrator().DropTable(&Profile{}, &Member{})

	if err := DB.AutoMigrate(&Profile{}, &Member{}); err != nil {
		t.Fatalf("Failed to migrate, got error: %v", err)
	}

	member := Member{Name: "foreign_key_constraints_belongs_to", Profile: Profile{Name: "my_profile_belongs_to", Refer: 1}}

	DB.Create(&member)

	var profile Profile
	if err := DB.First(&profile, "\"id\" = ?", member.Profile.ID).Error; err != nil {
		t.Fatalf("failed to find profile, got error: %v", err)
	} else if profile.Refer != member.ProfileID {
		t.Fatalf("member id is not equal: expects: %v, got: %v", profile.Refer, member.ProfileID)
	}

	DB.Model(&profile).Update("Refer", 100)

	var member2 Member
	if err := DB.First(&member2, "\"id\" = ?", member.ID).Error; err != nil {
		t.Fatalf("failed to find member, got error: %v", err)
	} else if member2.ProfileID != 100 {
		t.Fatalf("member id is not equal: expects: %v, got: %v", 100, member2.ProfileID)
	}

	if r := DB.Delete(&profile); r.Error != nil || r.RowsAffected != 1 {
		t.Fatalf("Should delete member, got error: %v, affected: %v", r.Error, r.RowsAffected)
	}

	var result Member
	if err := DB.First(&result, member.ID).Error; err == nil {
		t.Fatalf("Should not find deleted member")
	}

	if err := DB.First(&profile, profile.ID).Error; err == nil {
		t.Fatalf("Should not find deleted profile")
	}
}

func TestFullSaveAssociations(t *testing.T) {
	t.Skip()
	coupon := &Coupon{
		AppliesToProduct: []*CouponProduct{
			{ProductId: "full-save-association-product1"},
		},
		AmountOff:  10,
		PercentOff: 0.0,
	}

	err := DB.
		Session(&gorm.Session{FullSaveAssociations: true}).
		Create(coupon).Error
	if err != nil {
		t.Errorf("Failed, got error: %v", err)
	}

	if DB.First(&Coupon{}, "\"id\" = ?", coupon.ID).Error != nil {
		t.Errorf("Failed to query saved coupon")
	}

	if DB.First(&CouponProduct{}, "\"coupon_id\" = ? AND \"product_id\" = ?", coupon.ID, "full-save-association-product1").Error != nil {
		t.Errorf("Failed to query saved association")
	}

	orders := []Order{{Num: "order1", Coupon: coupon}, {Num: "order2", Coupon: coupon}}
	if err := DB.Create(&orders).Error; err != nil {
		t.Errorf("failed to create orders, got %v", err)
	}

	coupon2 := Coupon{
		AppliesToProduct: []*CouponProduct{{Description: "coupon-description"}},
	}

	DB.Session(&gorm.Session{FullSaveAssociations: true}).Create(&coupon2)
	var result Coupon
	if err := DB.Preload("AppliesToProduct").First(&result, "\"id\" = ?", coupon2.ID).Error; err != nil {
		t.Errorf("Failed to create coupon w/o name, got error: %v", err)
	}

	if len(result.AppliesToProduct) != 1 {
		t.Errorf("Failed to preload AppliesToProduct")
	}
}

func TestSaveBelongsCircularReference(t *testing.T) {
	parent := Parent{}
	DB.Create(&parent)

	child := Child{ParentID: &parent.ID, Parent: &parent}
	DB.Create(&child)

	parent.FavChildID = child.ID
	parent.FavChild = &child
	DB.Save(&parent)

	var parent1 Parent
	DB.First(&parent1, parent.ID)
	tests.AssertObjEqual(t, parent, parent1, "ID", "FavChildID")

	// Save and Updates is the same
	DB.Updates(&parent)
	DB.First(&parent1, parent.ID)
	tests.AssertObjEqual(t, parent, parent1, "ID", "FavChildID")
}

func TestSaveHasManyCircularReference(t *testing.T) {
	parent := Parent{}
	DB.Create(&parent)

	child := Child{ParentID: &parent.ID, Parent: &parent, Name: "HasManyCircularReference"}
	child1 := Child{ParentID: &parent.ID, Parent: &parent, Name: "HasManyCircularReference1"}

	parent.Children = []*Child{&child, &child1}
	DB.Save(&parent)

	var children []*Child
	DB.Where("\"parent_id\" = ?", parent.ID).Find(&children)
	if len(children) != len(parent.Children) ||
		children[0].ID != parent.Children[0].ID ||
		children[1].ID != parent.Children[1].ID {
		t.Errorf("circular reference children save not equal children:%v parent.Children:%v",
			children, parent.Children)
	}
}

func TestAssociationError(t *testing.T) {
	user := *GetUser("TestAssociationError", Config{Pets: 2, Company: true, Account: true, Languages: 2})
	DB.Create(&user)

	var user1 User
	DB.Preload("Company").Preload("Pets").Preload("Account").Preload("Languages").First(&user1)

	var emptyUser User
	var err error
	// belongs to
	err = DB.Model(&emptyUser).Association("Company").Delete(&user1.Company)
	tests.AssertEqual(t, err, gorm.ErrPrimaryKeyRequired)
	// has many
	err = DB.Model(&emptyUser).Association("Pets").Delete(&user1.Pets)
	tests.AssertEqual(t, err, gorm.ErrPrimaryKeyRequired)
	// has one
	err = DB.Model(&emptyUser).Association("Account").Delete(&user1.Account)
	tests.AssertEqual(t, err, gorm.ErrPrimaryKeyRequired)
	// many to many
	err = DB.Model(&emptyUser).Association("Languages").Delete(&user1.Languages)
	tests.AssertEqual(t, err, gorm.ErrPrimaryKeyRequired)
}

type (
	myType           string
	emptyQueryClause struct {
		Field *schema.Field
	}
)

func (myType) QueryClauses(f *schema.Field) []clause.Interface {
	return []clause.Interface{emptyQueryClause{Field: f}}
}

func (sd emptyQueryClause) Name() string {
	return "empty"
}

func (sd emptyQueryClause) Build(clause.Builder) {
}

func (sd emptyQueryClause) MergeClause(*clause.Clause) {
}

func (sd emptyQueryClause) ModifyStatement(stmt *gorm.Statement) {
	// do nothing
}

func TestAssociationEmptyQueryClause(t *testing.T) {
	type Organization struct {
		gorm.Model
		Name string
	}
	type Region struct {
		gorm.Model
		Name          string
		Organizations []Organization `gorm:"many2many:region_orgs;"`
	}
	type RegionOrg struct {
		RegionID       uint
		OrganizationID uint
		Empty          myType
	}
	if err := DB.SetupJoinTable(&Region{}, "Organizations", &RegionOrg{}); err != nil {
		t.Fatalf("Failed to set up join table, got error: %s", err)
	}
	if err := DB.Migrator().DropTable(&Organization{}, &Region{}); err != nil {
		if !strings.Contains(err.Error(), "ORA-00942") {
			t.Fatalf("Failed to migrate, got error: %s", err)
		}
	}
	if err := DB.AutoMigrate(&Organization{}, &Region{}); err != nil {
		t.Fatalf("Failed to migrate, got error: %v", err)
	}
	region := &Region{Name: "Region1"}
	if err := DB.Create(region).Error; err != nil {
		t.Fatalf("fail to create region %v", err)
	}
	var orgs []Organization

	if err := DB.Model(&Region{}).Association("Organizations").Find(&orgs); err != nil {
		t.Fatalf("fail to find region organizations %v", err)
	} else {
		tests.AssertEqual(t, len(orgs), 0)
	}
}

func TestBasicBelongsToAssociation(t *testing.T) {
	// Test basic BelongsTo association operations
	user := GetUser("TestBelongsTo", Config{Company: true})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user with company: %v", err)
	}

	// finding the association
	var foundUser User
	if err := DB.Preload("Company").First(&foundUser, "\"id\" = ?", user.ID).Error; err != nil {
		t.Fatalf("Failed to find user with company: %v", err)
	}

	if foundUser.Company.ID != user.Company.ID {
		t.Fatalf("Company ID mismatch: expected %d, got %d", user.Company.ID, foundUser.Company.ID)
	}

	// association count
	AssertAssociationCount(t, user, "Company", 1, "after creation")

	// replacing association
	newCompany := Company{Name: "New Test Company"}
	if err := DB.Create(&newCompany).Error; err != nil {
		t.Fatalf("Failed to create new company: %v", err)
	}

	if err := DB.Model(&user).Association("Company").Replace(&newCompany); err != nil {
		t.Fatalf("Failed to replace company association: %v", err)
	}

	var updatedUser User
	if err := DB.Preload("Company").First(&updatedUser, "\"id\" = ?", user.ID).Error; err != nil {
		t.Fatalf("Failed to find updated user: %v", err)
	}

	if updatedUser.Company.ID != newCompany.ID {
		t.Fatalf("Company was not replaced: expected %d, got %d", newCompany.ID, updatedUser.Company.ID)
	}
}

func TestBasicHasManyAssociation(t *testing.T) {
	// Test basic HasMany association operations
	user := GetUser("TestHasMany", Config{Pets: 3})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user with pets: %v", err)
	}

	// association count
	AssertAssociationCount(t, user, "Pets", 3, "after creation")

	// finding pets
	var pets []Pet
	if err := DB.Model(&user).Association("Pets").Find(&pets); err != nil {
		t.Fatalf("Failed to find pets: %v", err)
	}

	if len(pets) != 3 {
		t.Fatalf("Expected 3 pets, got %d", len(pets))
	}

	// appending new pet
	newPet := Pet{Name: "Additional Pet", UserID: &user.ID}
	if err := DB.Model(&user).Association("Pets").Append(&newPet); err != nil {
		t.Fatalf("Failed to append pet: %v", err)
	}

	AssertAssociationCount(t, user, "Pets", 4, "after append")

	// deleting one pet from association
	if err := DB.Model(&user).Association("Pets").Delete(&pets[0]); err != nil {
		t.Fatalf("Failed to delete pet from association: %v", err)
	}

	AssertAssociationCount(t, user, "Pets", 3, "after delete")
}

func TestBasicManyToManyAssociation(t *testing.T) {
	// Test basic ManyToMany association operations
	user := GetUser("TestManyToMany", Config{Languages: 2})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("Failed to create user with languages: %v", err)
	}

	// association count
	AssertAssociationCount(t, user, "Languages", 2, "after creation")

	// finding languages
	var languages []Language
	if err := DB.Model(&user).Association("Languages").Find(&languages); err != nil {
		t.Fatalf("Failed to find languages: %v", err)
	}

	if len(languages) != 2 {
		t.Fatalf("Expected 2 languages, got %d", len(languages))
	}

	// appending new language
	newLanguage := Language{Code: "FR", Name: "French"}
	if err := DB.Create(&newLanguage).Error; err != nil {
		t.Fatalf("Failed to create new language: %v", err)
	}

	if err := DB.Model(&user).Association("Languages").Append(&newLanguage); err != nil {
		t.Fatalf("Failed to append language: %v", err)
	}

	AssertAssociationCount(t, user, "Languages", 3, "after append")

	// replacing all languages
	replaceLanguages := []Language{
		{Code: "DE", Name: "German"},
		{Code: "IT", Name: "Italian"},
	}

	for i := range replaceLanguages {
		if err := DB.Create(&replaceLanguages[i]).Error; err != nil {
			t.Fatalf("Failed to create replacement language: %v", err)
		}
	}

	if err := DB.Model(&user).Association("Languages").Replace(replaceLanguages); err != nil {
		t.Fatalf("Failed to replace languages: %v", err)
	}

	AssertAssociationCount(t, user, "Languages", 2, "after replace")

	var finalLanguages []Language
	if err := DB.Model(&user).Association("Languages").Find(&finalLanguages); err != nil {
		t.Fatalf("Failed to find final languages: %v", err)
	}

	languageCodes := make(map[string]bool)
	for _, lang := range finalLanguages {
		languageCodes[lang.Code] = true
	}

	if !languageCodes["DE"] || !languageCodes["IT"] {
		t.Fatal("Languages were not replaced correctly")
	}

	// clearing all associations
	if err := DB.Model(&user).Association("Languages").Clear(); err != nil {
		t.Fatalf("Failed to clear languages: %v", err)
	}

	AssertAssociationCount(t, user, "Languages", 0, "after clear")
}
