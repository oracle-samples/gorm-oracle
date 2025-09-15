package tests

import (
	"regexp"
	"testing"

	"github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
)

type Student struct {
	ID   uint
	Name string
}

func (s Student) TableName() string {
	return "STUDENTS"
}

func TestSkipQuoteIdentifiers(t *testing.T) {
	db, err := openTestDBWithOptions(
		&oracle.Config{SkipQuoteIdentifiers: true},
		&gorm.Config{Logger: newLogger})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	db.Migrator().DropTable(&Student{})
	db.Migrator().CreateTable(&Student{})

	if !db.Migrator().HasTable(&Student{}) {
		t.Errorf("Failed to get table: student")
	}

	if !db.Migrator().HasColumn(&Student{}, "ID") {
		t.Errorf("Failed to get column: id")
	}

	if !db.Migrator().HasColumn(&Student{}, "NAME") {
		t.Errorf("Failed to get column: name")
	}

	dryrunDB := db.Session(&gorm.Session{DryRun: true})

	result := dryrunDB.Model(&Student{}).Create(&Student{ID: 1, Name: "John"})
	if !regexp.MustCompile(`^INSERT INTO STUDENTS \(name,id\) VALUES \(:1,:2\)$`).MatchString(result.Statement.SQL.String()) {
		t.Errorf("invalid insert SQL, got %v", result.Statement.SQL.String())
	}

	result = dryrunDB.First(&Student{})
	if !regexp.MustCompile(`^SELECT \* FROM STUDENTS ORDER BY STUDENTS\.id FETCH NEXT 1 ROW ONLY$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	result = dryrunDB.Find(&Student{ID: 1, Name: "John"})
	if !regexp.MustCompile(`^SELECT \* FROM STUDENTS WHERE STUDENTS\.id = :1$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	result = dryrunDB.Save(&Student{ID: 2, Name: "Mary"})
	if !regexp.MustCompile(`^UPDATE STUDENTS SET name=:1 WHERE id = :2$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}

	// Update with conditions
	result = dryrunDB.Model(&Student{}).Where("id = ?", 1).Update("name", "hello")
	if !regexp.MustCompile(`^UPDATE STUDENTS SET name=:1 WHERE id = :2$`).MatchString(result.Statement.SQL.String()) {
		t.Fatalf("SQL should include selected names, but got %v", result.Statement.SQL.String())
	}
}
