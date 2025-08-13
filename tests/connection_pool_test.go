package tests

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
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
