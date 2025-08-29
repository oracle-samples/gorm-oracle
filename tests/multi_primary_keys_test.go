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
	"reflect"
	"sort"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

type Blog struct {
	ID         uint   `gorm:"primary_key"`
	Locale     string `gorm:"primary_key"`
	Subject    string
	Body       string
	Tags       []Tag `gorm:"many2many:blog_tags;"`
	SharedTags []Tag `gorm:"many2many:shared_blog_tags;ForeignKey:id;References:id"`
	LocaleTags []Tag `gorm:"many2many:locale_blog_tags;ForeignKey:id,locale;References:id"`
}

type Tag struct {
	ID     uint   `gorm:"primary_key"`
	Locale string `gorm:"primary_key"`
	Value  string
	Blogs  []*Blog `gorm:"many2many:blog_tags"`
}

func compareTags(tags []Tag, contents []string) bool {
	var tagContents []string
	for _, tag := range tags {
		tagContents = append(tagContents, tag.Value)
	}
	sort.Strings(tagContents)
	sort.Strings(contents)
	return reflect.DeepEqual(tagContents, contents)
}

func TestManyToManyWithMultiPrimaryKeys(t *testing.T) {
	if name := DB.Dialector.Name(); name == "sqlite" || name == "sqlserver" {
		t.Skip("skip sqlite, sqlserver due to it doesn't support multiple primary keys with auto increment")
	}

	if name := DB.Dialector.Name(); name == "postgres" || name == "oracle" {
		stmt := gorm.Statement{DB: DB}
		stmt.Parse(&Blog{})
		stmt.Schema.LookUpField("ID").Unique = true
		stmt.Parse(&Tag{})
		stmt.Schema.LookUpField("ID").Unique = true
		// postgers and oracle only allow unique constraint matching given keys
	}

	DB.Migrator().DropTable(&Blog{}, &Tag{}, "blog_tags", "locale_blog_tags", "shared_blog_tags")
	if err := DB.AutoMigrate(&Blog{}, &Tag{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error: %v", err)
	}

	blog := Blog{
		Locale:  "ZH",
		Subject: "subject",
		Body:    "body",
		Tags: []Tag{
			{Locale: "ZH", Value: "tag1"},
			{Locale: "ZH", Value: "tag2"},
		},
	}

	DB.Save(&blog)
	if !compareTags(blog.Tags, []string{"tag1", "tag2"}) {
		t.Fatalf("Blog should has two tags")
	}

	// Append
	tag3 := &Tag{Locale: "ZH", Value: "tag3"}
	DB.Model(&blog).Association("Tags").Append([]*Tag{tag3})

	if !compareTags(blog.Tags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("Blog should has three tags after Append")
	}

	if count := DB.Model(&blog).Association("Tags").Count(); count != 3 {
		t.Fatalf("Blog should has 3 tags after Append, got %v", count)
	}

	var tags []Tag
	DB.Model(&blog).Association("Tags").Find(&tags)
	if !compareTags(tags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("Should find 3 tags")
	}

	var blog1 Blog
	DB.Preload("Tags").Find(&blog1)
	if !compareTags(blog1.Tags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("Preload many2many relations")
	}

	// Replace
	tag5 := &Tag{Locale: "ZH", Value: "tag5"}
	tag6 := &Tag{Locale: "ZH", Value: "tag6"}
	DB.Model(&blog).Association("Tags").Replace(tag5, tag6)
	var tags2 []Tag
	DB.Model(&blog).Association("Tags").Find(&tags2)
	if !compareTags(tags2, []string{"tag5", "tag6"}) {
		t.Fatalf("Should find 2 tags after Replace")
	}

	if DB.Model(&blog).Association("Tags").Count() != 2 {
		t.Fatalf("Blog should has three tags after Replace")
	}

	// Delete
	DB.Model(&blog).Association("Tags").Delete(tag5)
	var tags3 []Tag
	DB.Model(&blog).Association("Tags").Find(&tags3)
	if !compareTags(tags3, []string{"tag6"}) {
		t.Fatalf("Should find 1 tags after Delete")
	}

	if DB.Model(&blog).Association("Tags").Count() != 1 {
		t.Fatalf("Blog should has three tags after Delete")
	}

	DB.Model(&blog).Association("Tags").Delete(tag3)
	var tags4 []Tag
	DB.Model(&blog).Association("Tags").Find(&tags4)
	if !compareTags(tags4, []string{"tag6"}) {
		t.Fatalf("Tag should not be deleted when Delete with a unrelated tag")
	}

	// Clear
	DB.Model(&blog).Association("Tags").Clear()
	if DB.Model(&blog).Association("Tags").Count() != 0 {
		t.Fatalf("All tags should be cleared")
	}
}

func TestManyToManyWithCustomizedForeignKeys2(t *testing.T) {
	if name := DB.Dialector.Name(); name == "sqlite" || name == "sqlserver" {
		t.Skip("skip sqlite, sqlserver due to it doesn't support multiple primary keys with auto increment")
	}

	if name := DB.Dialector.Name(); name == "postgres" {
		t.Skip("skip postgres due to it only allow unique constraint matching given keys")
	}

	if name := DB.Dialector.Name(); name == "oracle" {
		stmt := gorm.Statement{DB: DB}
		stmt.Parse(&Blog{})
		stmt.Schema.LookUpField("ID").Unique = true
		stmt.Parse(&Tag{})
		stmt.Schema.LookUpField("ID").Unique = true
		// oracle only allow unique constraint matching given keys
	}

	DB.Migrator().DropTable(&Blog{}, &Tag{}, "blog_tags", "locale_blog_tags", "shared_blog_tags")
	if err := DB.AutoMigrate(&Blog{}, &Tag{}); err != nil {
		t.Fatalf("Failed to auto migrate, got error: %v", err)
	}

	blog := Blog{
		Locale:  "ZH",
		Subject: "subject",
		Body:    "body",
		LocaleTags: []Tag{
			{Locale: "ZH", Value: "tag1"},
			{Locale: "ZH", Value: "tag2"},
		},
	}
	DB.Save(&blog)

	blog2 := Blog{
		ID:     2,
		Locale: "EN",
	}
	DB.Create(&blog2)

	// Append
	tag3 := &Tag{Locale: "ZH", Value: "tag3"}
	DB.Model(&blog).Association("LocaleTags").Append([]*Tag{tag3})
	if !compareTags(blog.LocaleTags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("Blog should has three tags after Append")
	}

	if DB.Model(&blog).Association("LocaleTags").Count() != 3 {
		t.Fatalf("Blog should has three tags after Append")
	}

	if DB.Model(&blog2).Association("LocaleTags").Count() != 0 {
		t.Fatalf("EN Blog should has 0 tags after ZH Blog Append")
	}

	var tags []Tag
	DB.Model(&blog).Association("LocaleTags").Find(&tags)
	if !compareTags(tags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("Should find 3 tags")
	}

	DB.Model(&blog2).Association("LocaleTags").Find(&tags)
	if len(tags) != 0 {
		t.Fatalf("Should find 0 tags for EN Blog")
	}

	var blog1 Blog
	DB.Preload("LocaleTags").Find(&blog1, "\"locale\" = ? AND \"id\" = ?", "ZH", blog.ID)
	if !compareTags(blog1.LocaleTags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("Preload many2many relations")
	}

	tag4 := &Tag{Locale: "ZH", Value: "tag4"}
	DB.Model(&blog2).Association("LocaleTags").Append(tag4)

	DB.Model(&blog).Association("LocaleTags").Find(&tags)
	if !compareTags(tags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("Should find 3 tags for EN Blog")
	}

	DB.Model(&blog2).Association("LocaleTags").Find(&tags)
	if !compareTags(tags, []string{"tag4"}) {
		t.Fatalf("Should find 1 tags  for EN Blog")
	}

	// Replace
	tag5 := &Tag{Locale: "ZH", Value: "tag5"}
	tag6 := &Tag{Locale: "ZH", Value: "tag6"}
	DB.Model(&blog2).Association("LocaleTags").Replace(tag5, tag6)

	var tags2 []Tag
	DB.Model(&blog).Association("LocaleTags").Find(&tags2)
	if !compareTags(tags2, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("CN Blog's tags should not be changed after EN Blog Replace")
	}

	var blog11 Blog
	DB.Preload("LocaleTags").First(&blog11, "\"id\" = ? AND \"locale\" = ?", blog.ID, blog.Locale)
	if !compareTags(blog11.LocaleTags, []string{"tag1", "tag2", "tag3"}) {
		t.Fatalf("CN Blog's tags should not be changed after EN Blog Replace")
	}

	DB.Model(&blog2).Association("LocaleTags").Find(&tags2)
	if !compareTags(tags2, []string{"tag5", "tag6"}) {
		t.Fatalf("Should find 2 tags after Replace")
	}

	var blog21 Blog
	DB.Preload("LocaleTags").First(&blog21, "\"id\" = ? AND \"locale\" = ?", blog2.ID, blog2.Locale)
	if !compareTags(blog21.LocaleTags, []string{"tag5", "tag6"}) {
		t.Fatalf("EN Blog's tags should be changed after Replace")
	}

	if DB.Model(&blog).Association("LocaleTags").Count() != 3 {
		t.Fatalf("ZH Blog should has three tags after Replace")
	}

	if DB.Model(&blog2).Association("LocaleTags").Count() != 2 {
		t.Fatalf("EN Blog should has two tags after Replace")
	}

	// Delete
	DB.Model(&blog).Association("LocaleTags").Delete(tag5)

	if DB.Model(&blog).Association("LocaleTags").Count() != 3 {
		t.Fatalf("ZH Blog should has three tags after Delete with EN's tag")
	}

	if DB.Model(&blog2).Association("LocaleTags").Count() != 2 {
		t.Fatalf("EN Blog should has two tags after ZH Blog Delete with EN's tag")
	}

	DB.Model(&blog2).Association("LocaleTags").Delete(tag5)

	if DB.Model(&blog).Association("LocaleTags").Count() != 3 {
		t.Fatalf("ZH Blog should has three tags after EN Blog Delete with EN's tag")
	}

	if DB.Model(&blog2).Association("LocaleTags").Count() != 1 {
		t.Fatalf("EN Blog should has 1 tags after EN Blog Delete with EN's tag")
	}

	// Clear
	DB.Model(&blog2).Association("LocaleTags").Clear()
	if DB.Model(&blog).Association("LocaleTags").Count() != 3 {
		t.Fatalf("ZH Blog's tags should not be cleared when clear EN Blog's tags")
	}

	if DB.Model(&blog2).Association("LocaleTags").Count() != 0 {
		t.Fatalf("EN Blog's tags should be cleared when clear EN Blog's tags")
	}

	DB.Model(&blog).Association("LocaleTags").Clear()
	if DB.Model(&blog).Association("LocaleTags").Count() != 0 {
		t.Fatalf("ZH Blog's tags should be cleared when clear ZH Blog's tags")
	}

	if DB.Model(&blog2).Association("LocaleTags").Count() != 0 {
		t.Fatalf("EN Blog's tags should be cleared")
	}
}

func TestCompositePrimaryKeysAssociations(t *testing.T) {
	type Label struct {
		BookID *uint  `gorm:"primarykey"`
		Name   string `gorm:"primarykey"`
		Value  string
	}

	type Book struct {
		ID     int
		Name   string
		Labels []Label
	}

	DB.Migrator().DropTable(&Label{}, &Book{})
	if err := DB.AutoMigrate(&Label{}, &Book{}); err != nil {
		t.Fatalf("failed to migrate, got %v", err)
	}

	book := Book{
		Name: "my book",
		Labels: []Label{
			{Name: "region", Value: "emea"},
		},
	}

	DB.Create(&book)

	var result Book
	if err := DB.Preload("Labels").First(&result, book.ID).Error; err != nil {
		t.Fatalf("failed to preload, got error %v", err)
	}

	tests.AssertEqual(t, book, result)
}
