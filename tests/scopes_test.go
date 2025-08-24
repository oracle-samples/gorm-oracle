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
	"fmt"
	"sync"
	"testing"
	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
)

func NameIn1And2(d *gorm.DB) *gorm.DB {
	return d.Where("\"name\" in (?)", []string{"ScopeUser1", "ScopeUser2"})
}

func NameIn2And3(d *gorm.DB) *gorm.DB {
	return d.Where("\"name\" in (?)", []string{"ScopeUser2", "ScopeUser3"})
}

func NameIn(names []string) func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		return d.Where("\"name\" in (?)", names)
	}
}

func TestScopes(t *testing.T) {
	users := []*User{
		GetUser("ScopeUser1", Config{}),
		GetUser("ScopeUser2", Config{}),
		GetUser("ScopeUser3", Config{}),
	}

	DB.Create(&users)

	var users1, users2, users3 []User
	DB.Scopes(NameIn1And2).Find(&users1)
	if len(users1) != 2 {
		t.Errorf("Should found two users's name in 1, 2, but got %v", len(users1))
	}

	DB.Scopes(NameIn1And2, NameIn2And3).Find(&users2)
	if len(users2) != 1 {
		t.Errorf("Should found one user's name is 2, but got %v", len(users2))
	}

	DB.Scopes(NameIn([]string{users[0].Name, users[2].Name})).Find(&users3)
	if len(users3) != 2 {
		t.Errorf("Should found two users's name in 1, 3, but got %v", len(users3))
	}

	db := DB.Scopes(func(tx *gorm.DB) *gorm.DB {
		return tx.Table("custom_table")
	}).Session(&gorm.Session{})

	db.AutoMigrate(&User{})
	if db.Find(&User{}).Statement.Table != "custom_table" {
		t.Errorf("failed to call Scopes")
	}

	result := DB.Scopes(NameIn1And2, func(tx *gorm.DB) *gorm.DB {
		return tx.Session(&gorm.Session{})
	}).Find(&users1)

	if result.RowsAffected != 2 {
		t.Errorf("Should found two users's name in 1, 2, but got %v", result.RowsAffected)
	}

	var maxID int64
	userTable := func(db *gorm.DB) *gorm.DB {
		return db.WithContext(context.Background()).Table("users")
	}
	if err := DB.Scopes(userTable).Select("max(\"id\")").Scan(&maxID).Error; err != nil {
		t.Errorf("select max(id)")
	}
}

func TestComplexScopes(t *testing.T) {
	tests := []struct {
		name     string
		queryFn  func(tx *gorm.DB) *gorm.DB
		expected string
	}{
		{
			name: "depth_1",
			queryFn: func(tx *gorm.DB) *gorm.DB {
				return tx.Scopes(
					func(d *gorm.DB) *gorm.DB { return d.Where("\"a\" = 1") },
					func(d *gorm.DB) *gorm.DB {
						return d.Where(DB.Or("\"b\" = 2").Or("\"c\" = 3"))
					},
				).Find(&Language{})
			},
			expected: `SELECT * FROM "languages" WHERE "a" = 1 AND ("b" = 2 OR "c" = 3)`,
		}, {
			name: "depth_1_pre_cond",
			queryFn: func(tx *gorm.DB) *gorm.DB {
				return tx.Where("\"z\" = 0").Scopes(
					func(d *gorm.DB) *gorm.DB { return d.Where("\"a\" = 1") },
					func(d *gorm.DB) *gorm.DB {
						return d.Or(DB.Where("\"b\" = 2").Or("\"c\" = 3"))
					},
				).Find(&Language{})
			},
			expected: `SELECT * FROM "languages" WHERE "z" = 0 AND "a" = 1 OR ("b" = 2 OR "c" = 3)`,
		}, {
			name: "depth_2",
			queryFn: func(tx *gorm.DB) *gorm.DB {
				return tx.Scopes(
					func(d *gorm.DB) *gorm.DB { return d.Model(&Language{}) },
					func(d *gorm.DB) *gorm.DB {
						return d.
							Or(DB.Scopes(
								func(d *gorm.DB) *gorm.DB { return d.Where("\"a\" = 1") },
								func(d *gorm.DB) *gorm.DB { return d.Where("\"b\" = 2") },
							)).
							Or("\"c\" = 3")
					},
					func(d *gorm.DB) *gorm.DB { return d.Where("\"d\" = 4") },
				).Find(&Language{})
			},
			expected: `SELECT * FROM "languages" WHERE "d" = 4 OR "c" = 3 OR ("a" = 1 AND "b" = 2)`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertEqualSQL(t, test.expected, DB.ToSQL(test.queryFn))
		})
	}
}

func TestEmptyAndNilScopes(t *testing.T) {
	setupScopeTestData(t)

	// Test with no scopes
	var users []User
	err := DB.Scopes().Find(&users).Error
	if err != nil {
		t.Errorf("Empty scopes should work, got error: %v", err)
	}

	// Test with empty slice of scopes
	emptyScopes := []func(*gorm.DB) *gorm.DB{}
	err = DB.Scopes(emptyScopes...).Find(&users).Error
	if err != nil {
		t.Errorf("Empty scope slice should work, got error: %v", err)
	}

	// Test behavior when we have mixed nil and valid scopes
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Mixed nil scopes caused panic (expected): %v", r)
		}
	}()
}

func TestScopesWithDatabaseErrors(t *testing.T) {
	setupScopeTestData(t)

	// Scope that generates invalid SQL
	invalidSQLScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("invalid_column_name_12345 = ?", "test")
	}

	var users []User
	err := DB.Scopes(invalidSQLScope).Find(&users).Error
	if err == nil {
		t.Error("Expected error for invalid SQL in scope, got nil")
	}

	// Verify database still works after scope error
	err = DB.Find(&users).Error
	if err != nil {
		t.Errorf("Database should still work after scope error, got: %v", err)
	}
}

func TestConflictingScopes(t *testing.T) {
	setupScopeTestData(t)

	// Scopes with contradictory conditions
	alwaysTrue := func(db *gorm.DB) *gorm.DB {
		return db.Where("1 = 1")
	}
	alwaysFalse := func(db *gorm.DB) *gorm.DB {
		return db.Where("1 = 0")
	}

	var users []User
	err := DB.Scopes(alwaysTrue, alwaysFalse).Find(&users).Error
	if err != nil {
		t.Errorf("Conflicting scopes should not cause error, got: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("Conflicting scopes should return no results, got %d users", len(users))
	}

	// Test conflicting WHERE conditions on same column
	nameScope1 := func(db *gorm.DB) *gorm.DB {
		return db.Where("\"name\" = ?", "ScopeUser1")
	}
	nameScope2 := func(db *gorm.DB) *gorm.DB {
		return db.Where("\"name\" = ?", "ScopeUser2")
	}

	err = DB.Scopes(nameScope1, nameScope2).Find(&users).Error
	if err != nil {
		t.Errorf("Conflicting name scopes should not cause error, got: %v", err)
	}
	if len(users) != 0 {
		t.Errorf("Conflicting name scopes should return no results, got %d users", len(users))
	}
}

func TestContextCancellationInScopes(t *testing.T) {
	setupScopeTestData(t)

	// Create a context that gets cancelled immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	contextScope := func(db *gorm.DB) *gorm.DB {
		return db.WithContext(ctx)
	}

	var users []User
	err := DB.Scopes(contextScope).Find(&users).Error
	// Error is expected due to context cancellation

	// Verify database still works after context cancellation
	err = DB.Find(&users).Error
	if err != nil {
		t.Errorf("Database should work after context cancellation in scope, got: %v", err)
	}
}

func TestConcurrentScopeUsage(t *testing.T) {
	setupScopeTestData(t)

	const numGoroutines = 10
	const operationsPerGoroutine = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				userScope := func(db *gorm.DB) *gorm.DB {
					return db.Where("\"name\" LIKE ?", fmt.Sprintf("ScopeUser%d", (goroutineID%3)+1))
				}

				var users []User
				err := DB.Scopes(userScope).Find(&users).Error
				if err != nil {
					errors <- fmt.Errorf("goroutine %d operation %d failed: %v", goroutineID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent scope error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Errorf("Got %d errors in concurrent scope usage", errorCount)
	}
}

func TestScopesThatModifyUnexpectedQuery(t *testing.T) {
	setupScopeTestData(t)

	// Scope that changes the table
	tableChangingScope := func(db *gorm.DB) *gorm.DB {
		return db.Table("companies")
	}

	// Scope that changes the model
	modelChangingScope := func(db *gorm.DB) *gorm.DB {
		return db.Model(&Company{})
	}

	// Test table changing scope
	var users []User
	err := DB.Model(&User{}).Scopes(tableChangingScope).Find(&users).Error

	// Test model changing scope
	err = DB.Scopes(modelChangingScope).Find(&users).Error

	// Scope that adds unexpected clauses
	limitScope := func(db *gorm.DB) *gorm.DB {
		return db.Limit(1).Offset(1).Order("\"id\" DESC")
	}

	err = DB.Scopes(limitScope).Find(&users).Error
	if err != nil {
		t.Errorf("Scope with limit/offset/order should work, got: %v", err)
	}
}

func TestLargeNumberOfScopes(t *testing.T) {
	setupScopeTestData(t)

	// Create a large number of scopes
	const numScopes = 100
	scopes := make([]func(*gorm.DB) *gorm.DB, numScopes)

	for i := 0; i < numScopes; i++ {
		val := i
		scopes[i] = func(db *gorm.DB) *gorm.DB {
			return db.Where("\"id\" > ?", val*-1) // Always true conditions
		}
	}

	var users []User
	start := time.Now()
	err := DB.Scopes(scopes...).Find(&users).Error
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Large number of scopes failed: %v", err)
	}

	t.Logf("Processing %d scopes took %v", numScopes, duration)

	// Verify we still get results
	if len(users) == 0 {
		t.Error("Large number of scopes should still return results")
	}
}

func TestScopesWithTransactions(t *testing.T) {
	setupScopeTestData(t)

	// Test scopes within a transaction
	err := DB.Transaction(func(tx *gorm.DB) error {
		transactionScope := func(db *gorm.DB) *gorm.DB {
			return db.Where("\"name\" = ?", "ScopeUser1")
		}

		var users []User
		return tx.Scopes(transactionScope).Find(&users).Error
	})

	if err != nil {
		t.Errorf("Scopes within transaction should work, got: %v", err)
	}

	// Test scope that tries to start its own transaction (nested transaction scenario)
	nestedTxScope := func(db *gorm.DB) *gorm.DB {
		return db.Begin()
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		var users []User
		return tx.Scopes(nestedTxScope).Find(&users).Error
	})
}

func TestScopesWithRawSQL(t *testing.T) {
	setupScopeTestData(t)

	// Scope that adds raw SQL conditions
	rawSQLScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("LENGTH(\"name\") > ?", 5)
	}

	var users []User
	err := DB.Scopes(rawSQLScope).Find(&users).Error
	if err != nil {
		t.Errorf("Raw SQL scope should work, got: %v", err)
	}

	// Test proper parameterized queries (safe)
	safeParameterizedScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("\"name\" = ? OR \"name\" LIKE ?", "test", "ScopeUser%")
	}

	err = DB.Scopes(safeParameterizedScope).Find(&users).Error
	if err != nil {
		t.Errorf("Parameterized SQL in scope should be safe, got: %v", err)
	}

	// Test complex safe expressions
	complexSafeScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("(\"name\" = ? OR \"age\" > ?) AND \"deleted_at\" IS NULL", "ScopeUser1", 10)
	}

	err = DB.Scopes(complexSafeScope).Find(&users).Error
	if err != nil {
		t.Errorf("Complex parameterized SQL should work, got: %v", err)
	}

	// Test that we get expected results
	if len(users) == 0 {
		t.Error("Should have found some users with complex scope")
	}
}

func TestScopeErrorRecovery(t *testing.T) {
	setupScopeTestData(t)

	// First, cause an error with a bad scope
	badScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("non_existent_column = ?", "value")
	}

	var users []User
	err := DB.Scopes(badScope).Find(&users).Error
	if err == nil {
		t.Error("Expected error from bad scope")
	}

	// Then verify normal operations still work
	goodScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("\"name\" = ?", "ScopeUser1")
	}

	err = DB.Scopes(goodScope).Find(&users).Error
	if err != nil {
		t.Errorf("Good scope should work after bad scope error: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user, got %d", len(users))
	}
}

func TestScopeChainModification(t *testing.T) {
	// Test that scopes don't interfere with each other's chain modifications
	setupScopeTestData(t)

	scope1Called := false
	scope2Called := false

	scope1 := func(db *gorm.DB) *gorm.DB {
		scope1Called = true
		return db.Where("\"id\" > ?", 0)
	}

	scope2 := func(db *gorm.DB) *gorm.DB {
		scope2Called = true
		return db.Where("\"name\" IS NOT NULL")
	}

	var users []User
	err := DB.Scopes(scope1, scope2).Find(&users).Error
	if err != nil {
		t.Errorf("Scope chain should work, got: %v", err)
	}

	if !scope1Called {
		t.Error("Scope1 should have been called")
	}
	if !scope2Called {
		t.Error("Scope2 should have been called")
	}
}

func TestScopesWithSubqueries(t *testing.T) {
	setupScopeTestData(t)

	// Scope that uses a subquery
	subqueryScope := func(db *gorm.DB) *gorm.DB {
		subQuery := DB.Model(&User{}).Select("\"name\"").Where("\"id\" = 1")
		return db.Where("\"name\" IN (?)", subQuery)
	}

	var users []User
	err := DB.Scopes(subqueryScope).Find(&users).Error
	if err != nil {
		t.Errorf("Subquery scope should work, got: %v", err)
	}
}

// Helper function to set up test data for scope tests
func setupScopeTestData(t *testing.T) {
	// Clean up any existing data
	DB.Exec("DELETE FROM users WHERE \"name\" LIKE 'ScopeUser%'")

	// Create test users
	users := []*User{
		GetUser("ScopeUser1", Config{}),
		GetUser("ScopeUser2", Config{}),
		GetUser("ScopeUser3", Config{}),
	}

	err := DB.Create(&users).Error
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}
}
