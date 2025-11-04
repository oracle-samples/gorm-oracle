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
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

type UUIDModel struct {
	ID   uint       `gorm:"primaryKey"`
	UUID *uuid.UUID `gorm:"type:VARCHAR2(36)"`
	// Data string     `gorm:"type:clob"`
}

func setupUUIDTestTables(t *testing.T) {
	t.Log("Setting up UUID test tables")

	DB.Migrator().DropTable(&UUIDModel{})

	err := DB.AutoMigrate(&UUIDModel{})
	if err != nil {
		t.Fatalf("Failed to migrate UUID test tables: %v", err)
	}

	t.Log("UUID test tables created successfully")
}

func TestUUIDPLSQL(t *testing.T) {
	myUUID := uuid.New()
	type test struct {
		model any
		fn    func(model any) error
	}
	tests := map[string]test{
		"InsertWithReturning": {
			model: []UUIDModel{
				{
					UUID: &myUUID,
				},
			},
			fn: func(model any) error {
				return DB.Create(model).Error
			},
		},
		"InsertWithReturningNil": {
			model: []UUIDModel{
				{
					UUID: nil,
				},
			},
			fn: func(model any) error {
				return DB.Create(model).Error
			},
		},
		"BatchInsert": {
			model: []UUIDModel{
				{
					UUID: &myUUID,
				},
				{
					UUID: nil,
				},
			},
			fn: func(model any) error {
				return DB.Clauses(clause.OnConflict{
					UpdateAll: true,
				}).CreateInBatches(model, 1000).Error
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			setupUUIDTestTables(t)
			err := tc.fn(tc.model)
			if err != nil {
				t.Fatalf("Failed to create UUID record with PLSQL: %v", err)
			}
		})
	}
}
