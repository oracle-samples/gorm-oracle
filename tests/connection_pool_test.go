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
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"
)

type TestProduct struct {
	ID          uint    `gorm:"primaryKey;autoIncrement"`
	Code        string  `gorm:"column:CODE;size:100;uniqueIndex"`
	Name        string  `gorm:"column:NAME;size:200"`
	Price       uint    `gorm:"column:PRICE"`
	Description *string `gorm:"column:DESCRIPTION;size:500"`
	CategoryID  *uint   `gorm:"column:CATEGORY_ID"`
}

type TestCategory struct {
	ID   uint   `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"column:NAME;size:100"`
}

func TestConnectionPooling(t *testing.T) {
	t.Skip()
	setupConnectionPoolTestTables(t)

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	t.Log("Setting up connection pool configuration...")
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	t.Log("Connection pool configured")

	// Test 1: Check initial pool stats
	t.Run("InitialPoolStatistics", func(t *testing.T) {
		stats := sqlDB.Stats()
		t.Logf("Max Open Connections: %d", stats.MaxOpenConnections)
		t.Logf("Open Connections: %d", stats.OpenConnections)
		t.Logf("In Use: %d", stats.InUse)
		t.Logf("Idle: %d", stats.Idle)

		if stats.MaxOpenConnections != 10 {
			t.Errorf("Expected max open connections: 10, got: %d", stats.MaxOpenConnections)
		}
	})

	// Test 2: Concurrent database operations
	t.Run("ConcurrentOperations", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 15

		results := make(chan string, numGoroutines)

		startTime := time.Now()

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				switch id % 3 {
				case 0:
					// SELECT operation
					var count int64
					err := DB.Model(&TestProduct{}).Count(&count).Error
					if err != nil {
						results <- fmt.Sprintf("Goroutine %d SELECT failed: %v", id, err)
					} else {
						results <- fmt.Sprintf("Goroutine %d SELECT success: %d products", id, count)
					}

				case 1:
					// INSERT operation
					product := &TestProduct{
						Code:  fmt.Sprintf("POOL_%d_%d", id, time.Now().UnixNano()),
						Name:  fmt.Sprintf("Pool Test Product %d", id),
						Price: uint(100 + id),
					}
					err := DB.Create(product).Error
					if err != nil {
						results <- fmt.Sprintf("Goroutine %d INSERT failed: %v", id, err)
					} else {
						results <- fmt.Sprintf("Goroutine %d INSERT success: ID %d", id, product.ID)
					}

				case 2:
					// Long-running query (simulate connection hold)
					var products []TestProduct
					err := DB.Raw("SELECT * FROM test_products WHERE ROWNUM <= 10").Scan(&products).Error
					time.Sleep(100 * time.Millisecond)
					if err != nil {
						results <- fmt.Sprintf("Goroutine %d LONG-QUERY failed: %v", id, err)
					} else {
						results <- fmt.Sprintf("Goroutine %d LONG-QUERY success: %d products", id, len(products))
					}
				}
			}(i)
		}

		// Monitor pool stats during concurrent operations
		go func() {
			for i := 0; i < 5; i++ {
				time.Sleep(200 * time.Millisecond)
				stats := sqlDB.Stats()
				t.Logf("[Monitor] Open: %d, InUse: %d, Idle: %d, WaitCount: %d",
					stats.OpenConnections, stats.InUse, stats.Idle, stats.WaitCount)
			}
		}()

		wg.Wait()
		close(results)

		duration := time.Since(startTime)
		t.Logf("Concurrent operations completed in %v", duration)

		// Collect and display results
		successCount := 0
		errorCount := 0
		for result := range results {
			if strings.Contains(result, "success") {
				successCount++
			} else {
				errorCount++
			}
			t.Logf("%s", result)
		}

		t.Logf("Summary: %d successful, %d failed operations", successCount, errorCount)

		if errorCount > 0 {
			t.Errorf("Expected no errors, but got %d failed operations", errorCount)
		}
	})

	// Test 3: Final pool statistics
	t.Run("FinalPoolStatistics", func(t *testing.T) {
		finalStats := sqlDB.Stats()
		t.Logf("Max Open Connections: %d", finalStats.MaxOpenConnections)
		t.Logf("Open Connections: %d", finalStats.OpenConnections)
		t.Logf("In Use: %d", finalStats.InUse)
		t.Logf("Idle: %d", finalStats.Idle)
		t.Logf("Wait Count: %d", finalStats.WaitCount)
		t.Logf("Wait Duration: %v", finalStats.WaitDuration)
		t.Logf("Max Idle Closed: %d", finalStats.MaxIdleClosed)
		t.Logf("Max Idle Time Closed: %d", finalStats.MaxIdleTimeClosed)
		t.Logf("Max Lifetime Closed: %d", finalStats.MaxLifetimeClosed)
	})

	// Test 4: Connection timeout test
	t.Run("ConnectionTimeout", func(t *testing.T) {
		var timeoutWg sync.WaitGroup
		numTimeoutTests := 20

		for i := 0; i < numTimeoutTests; i++ {
			timeoutWg.Add(1)
			go func(id int) {
				defer timeoutWg.Done()

				// Create a context with timeout
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				var count int64
				err := DB.WithContext(ctx).Model(&TestProduct{}).Count(&count).Error
				if err != nil {
					t.Logf("Timeout test %d failed: %v", id, err)
				} else {
					t.Logf("Timeout test %d success", id)
				}
			}(i)
		}

		timeoutWg.Wait()
		t.Log("Connection timeout tests completed")
	})

	// Test 5: Connection cleanup verification
	t.Run("ConnectionCleanup", func(t *testing.T) {
		t.Log("Waiting for idle connection cleanup...")
		time.Sleep(2 * time.Second)

		cleanupStats := sqlDB.Stats()
		t.Logf("After cleanup - Open: %d, InUse: %d, Idle: %d",
			cleanupStats.OpenConnections, cleanupStats.InUse, cleanupStats.Idle)
	})

	// Test 6: Connection health check
	t.Run("ConnectionHealth", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := sqlDB.PingContext(ctx)
		if err != nil {
			t.Errorf("Connection health check failed: %v", err)
		} else {
			t.Log("Connection health check passed")
		}
	})
	t.Run("PoolExhaustion", func(t *testing.T) {
		numConnections := 15 // More than maxOpenConns (10)
		var wg sync.WaitGroup

		t.Logf("Starting %d long-running connections (max pool size: 10)", numConnections)

		// Start operations that hold connections for a longer period
		for i := 0; i < numConnections; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				t.Logf("Goroutine %d: Starting long transaction", id)
				tx := DB.Begin()
				if tx.Error != nil {
					t.Logf("Goroutine %d: Failed to begin transaction: %v", id, tx.Error)
					return
				}

				// Perform some work to actually use the connection
				var count int64
				tx.Model(&TestProduct{}).Count(&count)

				// Hold connection for a significant time
				time.Sleep(2 * time.Second)

				tx.Rollback()
				t.Logf("Goroutine %d: Transaction completed", id)
			}(i)
		}

		// Give goroutines time to start and acquire connections
		time.Sleep(500 * time.Millisecond)

		// Check pool stats during the exhaustion period
		stats := sqlDB.Stats()
		t.Logf("Pool stats during exhaustion:")
		t.Logf("  Max Open: %d", stats.MaxOpenConnections)
		t.Logf("  Open: %d", stats.OpenConnections)
		t.Logf("  InUse: %d", stats.InUse)
		t.Logf("  Idle: %d", stats.Idle)
		t.Logf("  WaitCount: %d", stats.WaitCount)
		t.Logf("  WaitDuration: %v", stats.WaitDuration)

		// Wait a bit more to ensure some connections are waiting
		time.Sleep(500 * time.Millisecond)

		// Check again for wait statistics
		finalStats := sqlDB.Stats()
		t.Logf("Final pool stats:")
		t.Logf("  WaitCount: %d", finalStats.WaitCount)
		t.Logf("  WaitDuration: %v", finalStats.WaitDuration)

		// Either we should have some waits, OR all connections should be in use
		if finalStats.WaitCount == 0 {
			t.Errorf("Expected either some waits (WaitCount > 0) or high connection usage (InUse >= 8), got WaitCount=%d, InUse=%d",
				finalStats.WaitCount, finalStats.InUse)
		}

		// Wait for all goroutines to complete
		wg.Wait()

		t.Log("Pool exhaustion test completed")
	})
}

func setupConnectionPoolTestTables(t *testing.T) {
	t.Log("Setting up test tables using GORM migrator")

	// Drop existing tables
	DB.Migrator().DropTable(&TestProduct{}, &TestCategory{})

	// Create tables using GORM
	err := DB.AutoMigrate(&TestCategory{}, &TestProduct{})
	if err != nil {
		t.Fatalf("Failed to migrate tables: %v", err)
	}

	t.Log("Test tables created successfully")
}

func TestPoolConfiguration(t *testing.T) {
	setupConnectionPoolTestTables(t)

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	t.Run("ValidPoolSettings", func(t *testing.T) {
		// Test valid pool configuration
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
		sqlDB.SetConnMaxIdleTime(5 * time.Minute)

		// Verify settings took effect
		stats := sqlDB.Stats()
		if stats.MaxOpenConnections != 10 {
			t.Errorf("Expected MaxOpenConnections: 10, got: %d", stats.MaxOpenConnections)
		}

		// Test basic database operation works with pool
		var result string
		err := DB.Raw("SELECT 'pool_test' FROM dual").Scan(&result).Error
		if err != nil {
			t.Errorf("Basic query failed with pool configuration: %v", err)
		}
		if result != "pool_test" {
			t.Errorf("Expected 'pool_test', got '%s'", result)
		}
	})

	t.Run("InvalidPoolSettings", func(t *testing.T) {
		// Test that invalid settings are handled gracefully
		// Note: Go's sql.DB doesn't validate these at set time, but we can test edge cases

		// Test with zero values
		sqlDB.SetMaxOpenConns(0) // 0 means unlimited
		sqlDB.SetMaxIdleConns(0) // 0 means use default

		// Verify database still works
		var count int64
		err := DB.Model(&TestProduct{}).Count(&count).Error
		if err != nil {
			t.Errorf("Database operation failed with zero pool settings: %v", err)
		}

		// Reset to reasonable values
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
	})
}

// TestBasicPoolOperations tests fundamental pool operations
func TestBasicPoolOperations(t *testing.T) {
	setupConnectionPoolTestTables(t)

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	// Configure pool for testing
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)
	sqlDB.SetConnMaxLifetime(10 * time.Minute)

	t.Run("BasicCRUDOperations", func(t *testing.T) {
		// Test CREATE
		product := &TestProduct{
			Code:  "POOL_CRUD_001",
			Name:  "Pool Test Product",
			Price: 100,
		}
		err := DB.Create(product).Error
		if err != nil {
			t.Errorf("CREATE operation failed: %v", err)
		}
		if product.ID == 0 {
			t.Error("Product ID should be set after creation")
		}

		// Test READ
		var foundProduct TestProduct
		err = DB.Where("\"CODE\" = ?", "POOL_CRUD_001").First(&foundProduct).Error
		if err != nil {
			t.Errorf("READ operation failed: %v", err)
		}
		if foundProduct.Name != "Pool Test Product" {
			t.Errorf("Expected 'Pool Test Product', got '%s'", foundProduct.Name)
		}

		// Test UPDATE
		err = DB.Model(&foundProduct).Update("\"PRICE\"", 150).Error
		if err != nil {
			t.Errorf("UPDATE operation failed: %v", err)
		}

		// Verify update
		var updatedProduct TestProduct
		err = DB.Where("\"CODE\" = ?", "POOL_CRUD_001").First(&updatedProduct).Error
		if err != nil {
			t.Errorf("Failed to verify update: %v", err)
		}
		if updatedProduct.Price != 150 {
			t.Errorf("Expected price 150, got %d", updatedProduct.Price)
		}

		// Test DELETE
		err = DB.Delete(&updatedProduct).Error
		if err != nil {
			t.Errorf("DELETE operation failed: %v", err)
		}

		// Verify deletion
		var deletedProduct TestProduct
		err = DB.Where("\"CODE\" = ?", "POOL_CRUD_001").First(&deletedProduct).Error
		if err != gorm.ErrRecordNotFound {
			t.Errorf("Expected record not found, got: %v", err)
		}
	})

	t.Run("PoolStatistics", func(t *testing.T) {
		initialStats := sqlDB.Stats()
		t.Logf("Initial Pool Stats - Open: %d, InUse: %d, Idle: %d",
			initialStats.OpenConnections, initialStats.InUse, initialStats.Idle)

		// Perform some operations to exercise the pool
		for i := 0; i < 3; i++ {
			var result string
			err := DB.Raw("SELECT ? FROM dual", fmt.Sprintf("test_%d", i)).Scan(&result).Error
			if err != nil {
				t.Errorf("Query %d failed: %v", i, err)
			}
		}

		finalStats := sqlDB.Stats()
		t.Logf("Final Pool Stats - Open: %d, InUse: %d, Idle: %d",
			finalStats.OpenConnections, finalStats.InUse, finalStats.Idle)

		// Basic sanity checks
		if finalStats.OpenConnections < 0 {
			t.Error("OpenConnections should not be negative")
		}
		if finalStats.InUse < 0 {
			t.Error("InUse connections should not be negative")
		}
	})
}

// TestPoolExhaustion tests behavior when pool is exhausted
func TestPoolExhaustion(t *testing.T) {
	setupConnectionPoolTestTables(t)

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	// Configure a very small pool for testing exhaustion
	sqlDB.SetMaxOpenConns(2)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxIdleTime(1 * time.Second)

	t.Run("PoolExhaustionBehavior", func(t *testing.T) {
		var wg sync.WaitGroup
		const numGoroutines = 5 // More than maxOpenConns
		errors := make(chan error, numGoroutines)

		t.Logf("Starting %d goroutines with pool size 2", numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Use context with timeout to prevent infinite waiting
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Start a transaction to hold the connection longer
				tx := DB.WithContext(ctx).Begin()
				if tx.Error != nil {
					errors <- fmt.Errorf("goroutine %d: failed to begin transaction: %v", id, tx.Error)
					return
				}

				// Do some work
				var count int64
				err := tx.Model(&TestProduct{}).Count(&count).Error
				if err != nil {
					tx.Rollback()
					errors <- fmt.Errorf("goroutine %d: count failed: %v", id, err)
					return
				}

				// Hold the connection for a bit
				time.Sleep(1 * time.Second)

				// Commit and release
				err = tx.Commit().Error
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: commit failed: %v", id, err)
					return
				}

				t.Logf("Goroutine %d completed successfully", id)
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check if all operations completed (some might timeout, which is expected)
		errorCount := 0
		for err := range errors {
			t.Logf("Error: %v", err)
			errorCount++
		}

		// In a properly configured pool, some operations might timeout but shouldn't panic
		if errorCount == numGoroutines {
			t.Error("All operations failed - pool might not be working correctly")
		}

		// Check final pool stats
		finalStats := sqlDB.Stats()
		t.Logf("Pool exhaustion stats - WaitCount: %d, WaitDuration: %v",
			finalStats.WaitCount, finalStats.WaitDuration)
	})
}

// TestConcurrentDatabaseOperations tests concurrent GORM operations
func TestConcurrentDatabaseOperations(t *testing.T) {
	setupConnectionPoolTestTables(t)

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	t.Run("ConcurrentCRUDMix", func(t *testing.T) {
		var wg sync.WaitGroup
		const numWorkers = 8
		const opsPerWorker = 10

		results := make(chan string, numWorkers*opsPerWorker)

		for worker := 0; worker < numWorkers; worker++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for op := 0; op < opsPerWorker; op++ {
					switch op % 4 {
					case 0: // CREATE
						product := &TestProduct{
							Code:  fmt.Sprintf("WORKER_%d_OP_%d", workerID, op),
							Name:  fmt.Sprintf("Worker %d Product %d", workerID, op),
							Price: uint(100 + workerID + op),
						}
						if err := DB.Create(product).Error; err != nil {
							results <- fmt.Sprintf("Worker %d CREATE failed: %v", workerID, err)
						} else {
							results <- fmt.Sprintf("Worker %d CREATE success", workerID)
						}

					case 1: // READ
						var count int64
						if err := DB.Model(&TestProduct{}).Count(&count).Error; err != nil {
							results <- fmt.Sprintf("Worker %d READ failed: %v", workerID, err)
						} else {
							results <- fmt.Sprintf("Worker %d READ success: %d products", workerID, count)
						}

					case 2: // UPDATE
						affected := DB.Model(&TestProduct{}).
							Where("\"CODE\" LIKE ?", fmt.Sprintf("WORKER_%d_%%", workerID)).
							Update("\"PRICE\"", uint(200+workerID+op)).RowsAffected
						results <- fmt.Sprintf("Worker %d UPDATE: %d rows affected", workerID, affected)

					case 3: // RAW QUERY
						var result string
						if err := DB.Raw("SELECT ? || '_' || ? FROM dual",
							fmt.Sprintf("worker_%d", workerID),
							fmt.Sprintf("op_%d", op)).Scan(&result).Error; err != nil {
							results <- fmt.Sprintf("Worker %d RAW failed: %v", workerID, err)
						} else {
							results <- fmt.Sprintf("Worker %d RAW success: %s", workerID, result)
						}
					}

					// Small delay to simulate real work
					time.Sleep(10 * time.Millisecond)
				}
			}(worker)
		}

		wg.Wait()
		close(results)

		// Collect and analyze results
		successCount := 0
		errorCount := 0
		for result := range results {
			if strings.Contains(result, "failed") {
				t.Logf("Error: %s", result)
				errorCount++
			} else {
				successCount++
			}
		}

		t.Logf("Concurrent operations completed - Success: %d, Errors: %d", successCount, errorCount)

		// We expect most operations to succeed
		totalOps := numWorkers * opsPerWorker
		if successCount < totalOps/2 {
			t.Errorf("Too many failures: %d errors out of %d operations", errorCount, totalOps)
		}
	})
}

// TestPoolConnectionRecovery tests pool behavior after connection errors
func TestPoolConnectionRecovery(t *testing.T) {
	setupConnectionPoolTestTables(t)

	sqlDB, err := DB.DB()
	if err != nil {
		t.Fatalf("Failed to get underlying DB: %v", err)
	}

	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)

	t.Run("RecoveryAfterInvalidQuery", func(t *testing.T) {
		// First, verify pool is working
		var result string
		err := DB.Raw("SELECT 'before_error' FROM dual").Scan(&result).Error
		if err != nil {
			t.Fatalf("Initial query failed: %v", err)
		}

		// Execute an invalid query that should cause an error but not break the pool
		err = DB.Raw("SELECT * FROM non_existent_table_12345").Scan(&result).Error
		if err == nil {
			t.Error("Expected error for invalid query")
		}

		// Verify pool still works after the error
		err = DB.Raw("SELECT 'after_error' FROM dual").Scan(&result).Error
		if err != nil {
			t.Errorf("Pool should work after error, but got: %v", err)
		}
		if result != "after_error" {
			t.Errorf("Expected 'after_error', got '%s'", result)
		}

		// Test that GORM operations still work
		var count int64
		err = DB.Model(&TestProduct{}).Count(&count).Error
		if err != nil {
			t.Errorf("GORM operations should work after error: %v", err)
		}
	})

	t.Run("ContextCancellationHandling", func(t *testing.T) {
		// Test context cancellation doesn't break the pool
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// This should timeout/cancel
		var result string
		err := DB.WithContext(ctx).Raw("SELECT 'timeout_test' FROM dual").Scan(&result).Error
		// Error is expected (timeout or cancellation)

		// Verify pool still works after context cancellation
		err = DB.Raw("SELECT 'post_cancel' FROM dual").Scan(&result).Error
		if err != nil {
			t.Errorf("Pool should work after context cancellation: %v", err)
		}
		if result != "post_cancel" {
			t.Errorf("Expected 'post_cancel', got '%s'", result)
		}
	})
}

// TestPoolLifecycleManagement tests proper pool lifecycle and cleanup
func TestPoolLifecycleManagement(t *testing.T) {
	setupConnectionPoolTestTables(t)

	t.Run("ConnectionPingAndHealth", func(t *testing.T) {
		sqlDB, err := DB.DB()
		if err != nil {
			t.Fatalf("Failed to get underlying DB: %v", err)
		}

		// Test basic ping
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
	})

	t.Run("PoolStatisticsOverTime", func(t *testing.T) {
		sqlDB, err := DB.DB()
		if err != nil {
			t.Fatalf("Failed to get underlying DB: %v", err)
		}

		// Configure pool with short idle timeout for testing
		sqlDB.SetMaxOpenConns(5)
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetConnMaxIdleTime(2 * time.Second)

		// Create some connections
		for i := 0; i < 3; i++ {
			var result string
			err := DB.Raw("SELECT ? FROM dual", fmt.Sprintf("conn_test_%d", i)).Scan(&result).Error
			if err != nil {
				t.Errorf("Query %d failed: %v", i, err)
			}
		}

		initialStats := sqlDB.Stats()
		t.Logf("Initial stats - Open: %d, InUse: %d, Idle: %d",
			initialStats.OpenConnections, initialStats.InUse, initialStats.Idle)

		// Wait for idle timeout
		time.Sleep(3 * time.Second)

		finalStats := sqlDB.Stats()
		t.Logf("Final stats - Open: %d, InUse: %d, Idle: %d",
			finalStats.OpenConnections, finalStats.InUse, finalStats.Idle)
		t.Logf("Lifetime stats - MaxIdleTimeClosed: %d, MaxLifetimeClosed: %d",
			finalStats.MaxIdleTimeClosed, finalStats.MaxLifetimeClosed)

		// Verify pool is still functional
		var result string
		err = DB.Raw("SELECT 'final_test' FROM dual").Scan(&result).Error
		if err != nil {
			t.Errorf("Pool should still work after idle timeout: %v", err)
		}
	})
}
