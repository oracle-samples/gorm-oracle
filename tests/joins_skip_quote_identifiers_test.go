/*
** Copyright (c) 2026 Oracle and/or its affiliates.
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
	"testing"
	"time"

	"github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
)

/**
 * Test 1: Simple Join with SkipQuoteIdentifiers
 **/

// Book domain struct - represents a Book in a library
type Book struct {
	ID          string    `gorm:"column:book_id;primaryKey;size:36"`
	Title       string    `gorm:"column:title;size:256"`
	AuthorID    string    `gorm:"column:author_id;size:36;not null"`
	ISBN        string    `gorm:"column:isbn;size:36;not null"`
	PublishedAt time.Time `gorm:"column:published_at"`
}

func (Book) TableName() string {
	return "books"
}

// BookModel with Author relationship - used for queries with joined data
type BookModel struct {
	Book
	Author *AuthorModel `gorm:"foreignKey:AuthorID;references:ID"` // This is the field that fails to populate on Oracle
}

func (BookModel) TableName() string {
	return "books"
}

// Author domain struct - represents an Author
type Author struct {
	ID   string `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:authorname"`
}

func (Author) TableName() string {
	return "authors"
}

// AuthorModel wrapper - used for relationships
type AuthorModel struct {
	Author
}

func (AuthorModel) TableName() string {
	return "authors"
}

func TestJoinsWithSkipQuoteIdentifiers(t *testing.T) {
	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	// Setup test tables
	if err := setupTest1Tables(db); err != nil {
		t.Fatalf("error setting up test tables: %v", err)
	}

	// Insert test data
	if err := insertTest1Data(db); err != nil {
		t.Fatalf("error inserting test data: %v", err)
	}

	// Test the Join with GORM's ORM mapping
	var dbBooks []BookModel
	result := db.Model(&BookModel{}).
		Joins("Author").
		Find(&dbBooks)

	if result.Error != nil {
		t.Fatalf("error executing GORM query: %v", result.Error)
	}

	if len(dbBooks) != 1 {
		t.Fatalf("expected 1 book, found %d", len(dbBooks))
	}

	// Check if the issue is reproduced
	// The bug: Raw SQL returns Author__* columns, but book.Author is nil
	for i, book := range dbBooks {
		if book.Author == nil {
			t.Fatalf("ISSUE REPRODUCED: Book %d Author is NULL", i)
		}
	}
}

func setupTest1Tables(db *gorm.DB) error {
	// Drop any existing tables
	if db.Migrator().HasTable("BOOKS") {
		db.Migrator().DropTable("BOOKS")
	}
	if db.Migrator().HasTable("AUTHORS") {
		db.Migrator().DropTable("AUTHORS")
	}

	// Create tables using raw SQL
	createAuthorsTable := `
    CREATE TABLE AUTHORS (
        ID VARCHAR2(36) PRIMARY KEY,
        AUTHORNAME VARCHAR2(256)
    )`
	if err := db.Exec(createAuthorsTable).Error; err != nil {
		return fmt.Errorf("error creating AUTHORS table: %w", err)
	}

	createBooksTable := `
    CREATE TABLE BOOKS (
        BOOK_ID VARCHAR2(36) PRIMARY KEY,
        TITLE VARCHAR2(256),
        AUTHOR_ID VARCHAR2(36) NOT NULL,
        ISBN VARCHAR2(36) NOT NULL,
        PUBLISHED_AT TIMESTAMP,
        CONSTRAINT FK_BOOKS_AUTHORS FOREIGN KEY (AUTHOR_ID) REFERENCES AUTHORS (ID)
    )`
	if err := db.Exec(createBooksTable).Error; err != nil {
		return fmt.Errorf("error creating BOOKS table: %w", err)
	}

	return nil
}

func insertTest1Data(db *gorm.DB) error {
	// Insert test author
	author := Author{
		ID:   "JKR",
		Name: "J.K. Rowling",
	}
	if err := db.Create(&author).Error; err != nil {
		return fmt.Errorf("error inserting author: %w", err)
	}

	// Insert test book
	book := Book{
		ID:          "HPATPS",
		Title:       "Harry Potter and the Philosopher's Stone",
		AuthorID:    "JKR",
		ISBN:        "978-0747532699",
		PublishedAt: time.Now(),
	}
	if err := db.Create(&book).Error; err != nil {
		return fmt.Errorf("error inserting book: %w", err)
	}

	return nil
}

/**
 * Test 2: Nested Joins with SkipQuoteIdentifiers
 **/

type company struct {
	ID   string `gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name;size:256"`
}

func (company) TableName() string {
	return "companies"
}

type employee struct {
	ID        string    `gorm:"column:id;primaryKey"`
	Name      string    `gorm:"column:name;size:256"`
	ManagerID string    `gorm:"column:manager_id"`
	Manager   *employee `gorm:"foreignKey:ManagerID;references:ID"`
	CompanyID string    `gorm:"column:company_id"`
	Company   company   `gorm:"foreignKey:CompanyID;references:ID"`
}

func (employee) TableName() string {
	return "employees"
}

func TestJoinsWithSkipQuoteIdentifiers_Nested(t *testing.T) {
	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	// Setup test tables
	if err := setupTest2Tables(db); err != nil {
		t.Fatalf("error setting up test tables: %v", err)
	}

	// Insert test data
	if err := insertTest2Data(db); err != nil {
		t.Fatalf("error inserting test data: %v", err)
	}

	// Test the Join with GORM's ORM mapping
	var employees []employee
	result := db.Model(&employee{}).
		Joins("Manager").Joins("Manager.Company").
		Find(&employees)

	if result.Error != nil {
		t.Fatalf("error executing GORM query: %v", result.Error)
	}

	if len(employees) != 3 {
		t.Fatalf("expected 3 employees, found %d", len(employees))
	}

	// Check if the issue is reproduced
	// The bug: Raw SQL returns Employee__* columns, but employee.Manager is nil
	for i, employee := range employees {
		if employee.Manager == nil {
			t.Fatalf("ISSUE REPRODUCED: Employee %d Manager is NULL", i)
		}
	}
}

func setupTest2Tables(db *gorm.DB) error {
	// Drop any existing tables to ensure a clean slate
	if db.Migrator().HasTable("COMPANIES") {
		db.Migrator().DropTable("COMPANIES")
	}
	if db.Migrator().HasTable("EMPLOYEES") {
		db.Migrator().DropTable("EMPLOYEES")
	}

	// Create tables using raw SQL
	createCompaniesTable := `
    CREATE TABLE COMPANIES (
        ID VARCHAR2(36) PRIMARY KEY,
        NAME VARCHAR2(256)
    )`
	if err := db.Exec(createCompaniesTable).Error; err != nil {
		return fmt.Errorf("error creating COMPANIES table: %w", err)
	}

	createUsersTable := `
    CREATE TABLE EMPLOYEES (
        ID VARCHAR2(36) PRIMARY KEY,
        NAME VARCHAR2(256),
				MANAGER_ID VARCHAR2(36),
				COMPANY_ID VARCHAR2(36) NOT NULL,
				CONSTRAINT FK_EMPLOYEES_MANAGER FOREIGN KEY (MANAGER_ID) REFERENCES EMPLOYEES (ID),
				CONSTRAINT FK_EMPLOYEES_COMPANY FOREIGN KEY (COMPANY_ID) REFERENCES COMPANIES (ID)
    )`
	if err := db.Exec(createUsersTable).Error; err != nil {
		return fmt.Errorf("error creating EMPLOYEES table: %w", err)
	}

	return nil
}

func insertTest2Data(db *gorm.DB) error {
	// Insert test data
	company := company{
		ID:   "COMP1",
		Name: "Test Company",
	}
	if err := db.Create(&company).Error; err != nil {
		return fmt.Errorf("error inserting company: %w", err)
	}

	alice := employee{
		ID:      "EMPLOYEE1",
		Name:    "Alice",
		Manager: nil,
		Company: company,
	}
	if err := db.Create([]employee{alice}).Error; err != nil {
		return fmt.Errorf("error inserting employees: %w", err)
	}

	members := []employee{
		{
			ID:      "EMPLOYEE2",
			Name:    "Bob",
			Manager: &alice,
			Company: company,
		},
		{
			ID:      "EMPLOYEE3",
			Name:    "Candice",
			Manager: &alice,
			Company: company,
		},
	}
	if err := db.Create(members).Error; err != nil {
		return fmt.Errorf("error inserting employees: %w", err)
	}

	return nil
}
