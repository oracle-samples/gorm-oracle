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
	"errors"
	"testing"

	"time"

	. "github.com/oracle-samples/gorm-oracle/tests/utils"

	"gorm.io/gorm"
)

func TestTransaction(t *testing.T) {
	tx := DB.Begin()
	user := *GetUser("transaction", Config{})

	if err := tx.Save(&user).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err := tx.First(&User{}, "\"name\" = ?", "transaction").Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	user1 := *GetUser("transaction1-1", Config{})

	if err := tx.Save(&user1).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err := tx.First(&User{}, "\"name\" = ?", user1.Name).Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	if sqlTx, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok || sqlTx == nil {
		t.Fatalf("Should return the underlying sql.Tx")
	}

	tx.Rollback()

	if err := DB.First(&User{}, "\"name\" = ?", "transaction").Error; err == nil {
		t.Fatalf("Should not find record after rollback, but got %v", err)
	}

	txDB := DB.Where("\"fake_name\" = ?", "fake_name")
	tx2 := txDB.Session(&gorm.Session{NewDB: true}).Begin()
	user2 := *GetUser("transaction-2", Config{})
	if err := tx2.Save(&user2).Error; err != nil {
		t.Fatalf("No error should raise, but got %v", err)
	}

	if err := tx2.First(&User{}, "\"name\" = ?", "transaction-2").Error; err != nil {
		t.Fatalf("Should find saved record, but got %v", err)
	}

	tx2.Commit()

	if err := DB.First(&User{}, "\"name\" = ?", "transaction-2").Error; err != nil {
		t.Fatalf("Should be able to find committed record, but got %v", err)
	}

	t.Run("this is test nested transaction and prepareStmt coexist case", func(t *testing.T) {
		// enable prepare statement
		tx3 := DB.Session(&gorm.Session{PrepareStmt: true})
		if err := tx3.Transaction(func(tx4 *gorm.DB) error {
			// nested transaction
			return tx4.Transaction(func(tx5 *gorm.DB) error {
				return tx5.First(&User{}, "\"name\" = ?", "transaction-2").Error
			})
		}); err != nil {
			t.Fatalf("%s", "prepare statement and nested transaction coexist"+err.Error())
		}
	})
}

func TestCancelTransaction(t *testing.T) {
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)
	cancelFunc()

	user := *GetUser("cancel_transaction", Config{})
	DB.Create(&user)

	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var result User
		tx.First(&result, user.ID)
		return nil
	})

	if err == nil {
		t.Fatalf("Transaction should get error when using cancelled context")
	}
}

func TestTransactionWithBlock(t *testing.T) {
	assertPanic := func(f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("The code did not panic")
			}
		}()
		f()
	}

	// rollback
	err := DB.Transaction(func(tx *gorm.DB) error {
		user := *GetUser("transaction-block", Config{})
		if err := tx.Save(&user).Error; err != nil {
			t.Fatalf("No error should raise")
		}

		if err := tx.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}

		return errors.New("the error message")
	})

	if err != nil && err.Error() != "the error message" {
		t.Fatalf("Transaction return error will equal the block returns error")
	}

	if err := DB.First(&User{}, "\"name\" = ?", "transaction-block").Error; err == nil {
		t.Fatalf("Should not find record after rollback")
	}

	// commit
	DB.Transaction(func(tx *gorm.DB) error {
		user := *GetUser("transaction-block-2", Config{})
		if err := tx.Save(&user).Error; err != nil {
			t.Fatalf("No error should raise")
		}

		if err := tx.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}
		return nil
	})

	if err := DB.First(&User{}, "\"name\" = ?", "transaction-block-2").Error; err != nil {
		t.Fatalf("Should be able to find committed record")
	}

	// panic will rollback
	assertPanic(func() {
		DB.Transaction(func(tx *gorm.DB) error {
			user := *GetUser("transaction-block-3", Config{})
			if err := tx.Save(&user).Error; err != nil {
				t.Fatalf("No error should raise")
			}

			if err := tx.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
				t.Fatalf("Should find saved record")
			}

			panic("force panic")
		})
	})

	if err := DB.First(&User{}, "\"name\" = ?", "transaction-block-3").Error; err == nil {
		t.Fatalf("Should not find record after panic rollback")
	}
}

func TestTransactionRaiseErrorOnRollbackAfterCommit(t *testing.T) {
	tx := DB.Begin()
	user := User{Name: "transaction"}
	if err := tx.Save(&user).Error; err != nil {
		t.Fatalf("No error should raise")
	}

	if err := tx.Commit().Error; err != nil {
		t.Fatalf("Commit should not raise error")
	}

	if err := tx.Rollback().Error; err == nil {
		t.Fatalf("Rollback after commit should raise error")
	}
}

func TestTransactionWithSavePoint(t *testing.T) {
	tx := DB.Begin()

	user := *GetUser("transaction-save-point", Config{})
	tx.Create(&user)

	if err := tx.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}

	if err := tx.SavePoint("save_point1").Error; err != nil {
		t.Fatalf("Failed to save point, got error %v", err)
	}

	user1 := *GetUser("transaction-save-point-1", Config{})
	tx.Create(&user1)

	if err := tx.First(&User{}, "\"name\" = ?", user1.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}

	if err := tx.RollbackTo("save_point1").Error; err != nil {
		t.Fatalf("Failed to save point, got error %v", err)
	}

	if err := tx.First(&User{}, "\"name\" = ?", user1.Name).Error; err == nil {
		t.Fatalf("Should not find rollbacked record")
	}

	if err := tx.SavePoint("save_point2").Error; err != nil {
		t.Fatalf("Failed to save point, got error %v", err)
	}

	user2 := *GetUser("transaction-save-point-2", Config{})
	tx.Create(&user2)

	if err := tx.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}

	if err := tx.Commit().Error; err != nil {
		t.Fatalf("Failed to commit, got error %v", err)
	}

	if err := DB.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user1.Name).Error; err == nil {
		t.Fatalf("Should not find rollbacked record")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}
}

func TestNestedTransactionWithBlock(t *testing.T) {
	var (
		user  = *GetUser("transaction-nested", Config{})
		user1 = *GetUser("transaction-nested-1", Config{})
		user2 = *GetUser("transaction-nested-2", Config{})
	)

	if err := DB.Transaction(func(tx *gorm.DB) error {
		tx.Create(&user)

		if err := tx.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}

		if err := tx.Transaction(func(tx1 *gorm.DB) error {
			tx1.Create(&user1)

			if err := tx1.First(&User{}, "\"name\" = ?", user1.Name).Error; err != nil {
				t.Fatalf("Should find saved record")
			}

			return errors.New("rollback")
		}); err == nil {
			t.Fatalf("nested transaction should returns error")
		}

		if err := tx.First(&User{}, "\"name\" = ?", user1.Name).Error; err == nil {
			t.Fatalf("Should not find rollbacked record")
		}

		if err := tx.Transaction(func(tx2 *gorm.DB) error {
			tx2.Create(&user2)

			if err := tx2.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
				t.Fatalf("Should find saved record")
			}

			return nil
		}); err != nil {
			t.Fatalf("nested transaction returns error: %v", err)
		}

		if err := tx.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}
		return nil
	}); err != nil {
		t.Fatalf("no error should return, but got %v", err)
	}

	if err := DB.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user1.Name).Error; err == nil {
		t.Fatalf("Should not find rollbacked record")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}
}

func TestDeeplyNestedTransactionWithBlockAndWrappedCallback(t *testing.T) {
	transaction := func(ctx context.Context, db *gorm.DB, callback func(ctx context.Context, db *gorm.DB) error) error {
		return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return callback(ctx, tx)
		})
	}
	var (
		user  = *GetUser("transaction-nested", Config{})
		user1 = *GetUser("transaction-nested-1", Config{})
		user2 = *GetUser("transaction-nested-2", Config{})
	)

	if err := transaction(context.Background(), DB, func(ctx context.Context, tx *gorm.DB) error {
		tx.Create(&user)

		if err := tx.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}

		if err := transaction(ctx, tx, func(ctx context.Context, tx1 *gorm.DB) error {
			tx1.Create(&user1)

			if err := tx1.First(&User{}, "\"name\" = ?", user1.Name).Error; err != nil {
				t.Fatalf("Should find saved record")
			}

			if err := transaction(ctx, tx1, func(ctx context.Context, tx2 *gorm.DB) error {
				tx2.Create(&user2)

				if err := tx2.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
					t.Fatalf("Should find saved record")
				}

				return errors.New("inner rollback")
			}); err == nil {
				t.Fatalf("nested transaction has no error")
			}

			return errors.New("rollback")
		}); err == nil {
			t.Fatalf("nested transaction should returns error")
		}

		if err := tx.First(&User{}, "\"name\" = ?", user1.Name).Error; err == nil {
			t.Fatalf("Should not find rollbacked record")
		}

		if err := tx.First(&User{}, "\"name\" = ?", user2.Name).Error; err == nil {
			t.Fatalf("Should not find saved record")
		}
		return nil
	}); err != nil {
		t.Fatalf("no error should return, but got %v", err)
	}

	if err := DB.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user1.Name).Error; err == nil {
		t.Fatalf("Should not find rollbacked parent record")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user2.Name).Error; err == nil {
		t.Fatalf("Should not find rollbacked nested record")
	}
}

func TestDisabledNestedTransaction(t *testing.T) {
	var (
		user  = *GetUser("transaction-nested", Config{})
		user1 = *GetUser("transaction-nested-1", Config{})
		user2 = *GetUser("transaction-nested-2", Config{})
	)

	if err := DB.Session(&gorm.Session{DisableNestedTransaction: true}).Transaction(func(tx *gorm.DB) error {
		tx.Create(&user)

		if err := tx.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}

		if err := tx.Transaction(func(tx1 *gorm.DB) error {
			tx1.Create(&user1)

			if err := tx1.First(&User{}, "\"name\" = ?", user1.Name).Error; err != nil {
				t.Fatalf("Should find saved record")
			}

			return errors.New("rollback")
		}); err == nil {
			t.Fatalf("nested transaction should returns error")
		}

		if err := tx.First(&User{}, "\"name\" = ?", user1.Name).Error; err != nil {
			t.Fatalf("Should not rollback record if disabled nested transaction support")
		}

		if err := tx.Transaction(func(tx2 *gorm.DB) error {
			tx2.Create(&user2)

			if err := tx2.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
				t.Fatalf("Should find saved record")
			}

			return nil
		}); err != nil {
			t.Fatalf("nested transaction returns error: %v", err)
		}

		if err := tx.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
			t.Fatalf("Should find saved record")
		}
		return nil
	}); err != nil {
		t.Fatalf("no error should return, but got %v", err)
	}

	if err := DB.First(&User{}, "\"name\" = ?", user.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user1.Name).Error; err != nil {
		t.Fatalf("Should not rollback record if disabled nested transaction support")
	}

	if err := DB.First(&User{}, "\"name\" = ?", user2.Name).Error; err != nil {
		t.Fatalf("Should find saved record")
	}
}

func TestTransactionOnClosedConn(t *testing.T) {
	DB, err := OpenTestConnection(&gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}
	rawDB, _ := DB.DB()
	rawDB.Close()

	if err := DB.Transaction(func(tx *gorm.DB) error {
		return nil
	}); err == nil {
		t.Errorf("should returns error when commit with closed conn, got error %v", err)
	}

	if err := DB.Session(&gorm.Session{PrepareStmt: true}).Transaction(func(tx *gorm.DB) error {
		return nil
	}); err == nil {
		t.Errorf("should returns error when commit with closed conn, got error %v", err)
	}
}

func TestTransactionWithHooks(t *testing.T) {
	user := GetUser("tTestTransactionWithHooks", Config{Account: true})
	DB.Create(&user)

	var err error
	err = DB.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&User{}).Limit(1).Transaction(func(tx2 *gorm.DB) error {
			return tx2.Scan(&User{}).Error
		})
	})
	if err != nil {
		t.Error(err)
	}

	// method with hooks
	err = DB.Transaction(func(tx1 *gorm.DB) error {
		// callMethod do
		tx2 := tx1.Find(&User{}).Session(&gorm.Session{NewDB: true})
		// trx in hooks
		return tx2.Transaction(func(tx3 *gorm.DB) error {
			return tx3.Where("user_id", user.ID).Delete(&Account{}).Error
		})
	})
	if err != nil {
		t.Error(err)
	}
}

func TestTransactionWithDefaultTimeout(t *testing.T) {
	db, err := OpenTestConnection(&gorm.Config{DefaultTransactionTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("failed to connect database, got error %v", err)
	}

	tx := db.Begin()
	time.Sleep(3 * time.Second)
	if err = tx.Find(&User{}).Error; err == nil {
		t.Errorf("should return error when transaction timeout, got error %v", err)
	}
}

// Test nested transactions with different error scenarios
func TestComplexNestedTransactions(t *testing.T) {
	scenarios := []struct {
		name        string
		innerError  error
		middleError error
		outerError  error
		expectUsers []string // Users that should exist after all transactions
	}{
		{
			name:        "All succeed",
			innerError:  nil,
			middleError: nil,
			outerError:  nil,
			expectUsers: []string{"outer", "middle", "inner"},
		},
		{
			name:        "Inner fails",
			innerError:  errors.New("inner failed"),
			middleError: nil,
			outerError:  nil,
			expectUsers: []string{"outer", "middle"},
		},
		{
			name:        "Middle fails",
			innerError:  nil,
			middleError: errors.New("middle failed"),
			outerError:  nil,
			expectUsers: []string{"outer"},
		},
		{
			name:        "Outer fails",
			innerError:  nil,
			middleError: nil,
			outerError:  errors.New("outer failed"),
			expectUsers: []string{},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Clean up
			DB.Where("\"name\" IN ?", []string{"outer", "middle", "inner"}).Delete(&User{})

			err := DB.Transaction(func(tx1 *gorm.DB) error {
				outerUser := *GetUser("outer", Config{})
				if err := tx1.Create(&outerUser).Error; err != nil {
					return err
				}

				if err := tx1.Transaction(func(tx2 *gorm.DB) error {
					middleUser := *GetUser("middle", Config{})
					if err := tx2.Create(&middleUser).Error; err != nil {
						return err
					}

					if err := tx2.Transaction(func(tx3 *gorm.DB) error {
						innerUser := *GetUser("inner", Config{})
						if err := tx3.Create(&innerUser).Error; err != nil {
							return err
						}
						return scenario.innerError
					}); err != nil && scenario.innerError != nil {
						// Expected error
					} else if err != nil {
						return err
					}

					return scenario.middleError
				}); err != nil && scenario.middleError != nil {
					// Expected error
				} else if err != nil {
					return err
				}

				return scenario.outerError
			})

			// Check final error state
			if scenario.outerError != nil && err == nil {
				t.Error("Expected outer error")
			}

			// Verify which users exist
			for _, name := range scenario.expectUsers {
				var count int64
				DB.Model(&User{}).Where("\"name\" = ?", name).Count(&count)
				if count != 1 {
					t.Errorf("Expected user %s to exist", name)
				}
			}

			// Verify which users don't exist
			allUsers := []string{"outer", "middle", "inner"}
			for _, name := range allUsers {
				shouldExist := false
				for _, expectedName := range scenario.expectUsers {
					if name == expectedName {
						shouldExist = true
						break
					}
				}
				if !shouldExist {
					var count int64
					DB.Model(&User{}).Where("\"name\" = ?", name).Count(&count)
					if count != 0 {
						t.Errorf("User %s should not exist", name)
					}
				}
			}
		})
	}
}

// Test transaction with raw SQL
func TestTransactionWithRawSQL(t *testing.T) {
	user := *GetUser("raw-sql-tx", Config{})

	err := DB.Transaction(func(tx *gorm.DB) error {
		// Create user with raw SQL
		if err := tx.Exec(
			"INSERT INTO \"users\" (\"name\", \"age\", \"birthday\", \"created_at\", \"updated_at\") VALUES (?, ?, ?, ?, ?)",
			user.Name, user.Age, user.Birthday, time.Now(), time.Now(),
		).Error; err != nil {
			return err
		}

		// Query with raw SQL
		var count int64
		if err := tx.Raw("SELECT COUNT(*) FROM \"users\" WHERE \"name\" = ?", user.Name).Scan(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return errors.New("user not found after raw insert")
		}

		// Update with raw SQL
		if err := tx.Exec("UPDATE \"users\" SET \"age\" = \"age\" + 1 WHERE \"name\" = ?", user.Name).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		t.Fatalf("Raw SQL transaction failed: %v", err)
	}

	// Verify the user exists and age was updated
	var result User
	if err := DB.Where("\"name\" = ?", user.Name).First(&result).Error; err != nil {
		t.Fatalf("Failed to find user: %v", err)
	}
	if result.Age != user.Age+1 {
		t.Errorf("Expected age %d, got %d", user.Age+1, result.Age)
	}
}
