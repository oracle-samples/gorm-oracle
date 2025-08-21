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
	"fmt"
	"log"
	"os"
	"sync"
	"testing"

	"gorm.io/gorm"
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
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errChan := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			var val int
			err := DB.Connection(func(tx *gorm.DB) error {
				return tx.Raw("SELECT ? FROM dual", i).Scan(&val).Error
			})
			if err != nil {
				errChan <- fmt.Errorf("goroutine #%d: connection err: %v", i, err)
				return
			}
			if val != i {
				errChan <- fmt.Errorf("goroutine #%d: got wrong result: %v", i, val)
			}
		}(i)
	}

	wg.Wait()
	close(errChan)
	for err := range errChan {
		t.Error(err)
	}
}
