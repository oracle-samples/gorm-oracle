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
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/oracle-samples/gorm-oracle/oracle"
)

func TestWithSingleConnection(t *testing.T) {
	expectedString := "hello, db"
	var actualString string

	sqlString := fmt.Sprintf("select '%s' from dual", expectedString)

	err := DB.Connection(func(tx *gorm.DB) error {
		if err := tx.Raw(sqlString).Scan(&actualString).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Errorf("WithSingleConnection should work, but got err %v", err)
	}

	if actualString != expectedString {
		t.Errorf("WithSingleConnection() method should get correct value, expect: %v, got %v", expectedString, actualString)
	}
}

func TestConnectionAfterError(t *testing.T) {
	// Execute an invalid query
	err := DB.Exec("SELECT invalid_column FROM dual").Error
	if err == nil {
		t.Error("Expected error for invalid query, but got nil")
	}

	// Verify connection still works after error
	var result string
	err = DB.Raw("SELECT 'connection_works' FROM dual").Scan(&result).Error
	if err != nil {
		t.Errorf("Connection should work after error, but got: %v", err)
	}
	if result != "connection_works" {
		t.Errorf("Expected 'connection_works', got '%s'", result)
	}
}

func TestConnectionWithInvalidQuery(t *testing.T) {
	err := DB.Connection(func(tx *gorm.DB) error {
		return tx.Exec("SELECT * FROM non_existent_table").Error
	})
	if err == nil {
		t.Fatalf("Expected error for invalid query in Connection, got nil")
	}
}

func TestMultipleSequentialConnections(t *testing.T) {
	for i := 0; i < 20; i++ {
		var val int
		err := DB.Connection(func(tx *gorm.DB) error {
			return tx.Raw("SELECT 1 FROM dual").Scan(&val).Error
		})
		if err != nil {
			t.Fatalf("Sequential Connection #%d failed: %v", i+1, err)
		}
		if val != 1 {
			t.Fatalf("Sequential Connection #%d got wrong result: %v", i+1, val)
		}
	}
}

func TestConnectionAfterDBClose(t *testing.T) {
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("DB.DB() should not fail, got: %v", err)
	}
	err = sqlDB.Close()
	if err != nil {
		t.Fatalf("sqlDB.Close() failed: %v", err)
	}
	cerr := DB.Connection(func(tx *gorm.DB) error {
		var v int
		return tx.Raw("SELECT 1 FROM dual").Scan(&v).Error
	})
	if cerr == nil {
		t.Fatalf("Expected error when calling Connection after DB closed, got nil")
	}
	if DB, err = OpenTestConnection(&gorm.Config{Logger: newLogger}); err != nil {
		log.Printf("failed to connect database, got error %v", err)
		os.Exit(1)
	}
}

func TestConnectionHandlesPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Expected panic inside Connection, but none occurred")
		}
	}()
	DB.Connection(func(tx *gorm.DB) error {
		panic("panic in connection callback")
	})
	t.Fatalf("Should have panicked inside connection callback")
}

func TestConcurrentConnections(t *testing.T) {
	const numGoroutines = 10
	const operationsPerGoroutine = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				var result string
				query := fmt.Sprintf("SELECT 'goroutine_%d_op_%d' FROM dual", goroutineID, j)
				if err := DB.Raw(query).Scan(&result).Error; err != nil {
					errors <- fmt.Errorf("goroutine %d operation %d failed: %v", goroutineID, j, err)
					return
				}
				expected := fmt.Sprintf("goroutine_%d_op_%d", goroutineID, j)
				if result != expected {
					errors <- fmt.Errorf("goroutine %d operation %d: expected '%s', got '%s'", goroutineID, j, expected, result)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Error(err)
	}
}

func TestContextTimeout(t *testing.T) {
	// Test with very short timeout that should trigger
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This should timeout for most operations
	err := DB.WithContext(ctx).Raw("SELECT 1 FROM dual").Error
	if err == nil {
		t.Log("Operation completed before timeout (this is possible on fast systems)")
	}

	// Test with reasonable timeout that should succeed
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	var result int
	err = DB.WithContext(ctx2).Raw("SELECT 42 FROM dual").Scan(&result).Error
	if err != nil {
		t.Errorf("Operation with reasonable timeout failed: %v", err)
	}
	if result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}
}

func TestSessionInfo(t *testing.T) {
	// Test USER function first
	var username string
	err := DB.Raw("SELECT USER FROM dual").Scan(&username).Error
	if err != nil {
		t.Errorf("Failed to get username: %v", err)
		return
	}

	if username == "" {
		t.Skip("USER function returned empty - unusual Oracle configuration")
	}

	// Test SYS_CONTEXT functions
	var sessionInfo struct {
		InstanceName string `gorm:"column:instance_name"`
		DatabaseName string `gorm:"column:database_name"`
	}

	query := `
		SELECT 
			SYS_CONTEXT('USERENV', 'INSTANCE_NAME') as instance_name,
			SYS_CONTEXT('USERENV', 'DB_NAME') as database_name
		FROM dual
	`

	err = DB.Raw(query).Scan(&sessionInfo).Error
	if err != nil {
		t.Errorf("Failed to get session context info: %v", err)
		return
	}

	t.Logf("Session Info - User: %s", username)
	if sessionInfo.InstanceName != "" {
		t.Logf("Instance: %s", sessionInfo.InstanceName)
	}
	if sessionInfo.DatabaseName != "" {
		t.Logf("Database: %s", sessionInfo.DatabaseName)
	}

	// Only require username
	if sessionInfo.InstanceName == "" || sessionInfo.DatabaseName == "" {
		t.Skip("SYS_CONTEXT functions unavailable - likely permissions or configuration issue")
	}
}

func TestConnectionPing(t *testing.T) {
	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get sql.DB: %v", err)
	}

	err = sqlDB.Ping()
	if err != nil {
		t.Errorf("Database ping failed: %v", err)
	}

	// Test ping with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = sqlDB.PingContext(ctx)
	if err != nil {
		t.Errorf("Database ping with context failed: %v", err)
	}
}

func TestServerVersion(t *testing.T) {
	version, err := oracle.GetServerVersion(DB)
	if err != nil {
		t.Errorf("Failed to get server version: %v", err)
	}
	if version <= 0 {
		t.Errorf("Invalid server version: %d", version)
	}
	t.Logf("Database server version: %d", version)
}
