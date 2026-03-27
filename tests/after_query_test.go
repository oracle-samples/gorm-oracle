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
	"sync"
	"testing"

	"github.com/google/uuid"
	oracle "github.com/oracle-samples/gorm-oracle/oracle"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// uuid as pointer should return nil if Oracle holds NULL
type afterQueryUUIDPtrModel struct {
	ID   uint       `gorm:"primaryKey"`
	UUID *uuid.UUID `gorm:"type:uuid"`
}

// uuid as value should return zero UUID if Oracle holds NULL
type afterQueryUUIDValueModel struct {
	ID   uint      `gorm:"primaryKey"`
	UUID uuid.UUID `gorm:"type:uuid"`
}

// uuid as string should not be modified by AfterQuery
type afterQueryNonUUIDModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"type:varchar(100)"`
}

// construct minimal *gorm.DB with dest struct bound on statement
func buildDBForAfterQuery(t *testing.T, dest interface{}) *gorm.DB {
	t.Helper()
	s, err := schema.Parse(dest, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("schema.Parse failed: %v", err)
	}
	return &gorm.DB{
		Statement: &gorm.Statement{
			Context: context.Background(),
			Dest:    dest,
			Schema:  s,
		},
	}
}

// ensure nil *gorm.DB does not cause a panic
func TestAfterQuery_NilDB(t *testing.T) {
	oracle.AfterQuery(nil) // must not panic
}

// ensure DB with a nil Statement does not panic
func TestAfterQuery_NilStatement(t *testing.T) {
	oracle.AfterQuery(&gorm.DB{}) // must not panic
}

// UUID pointer - ensure the 00000000-0000-0000-0000-000000000000 UUID is mapped to nil
func TestAfterQuery_UUIDPtr_ZeroUUID_SetToNil(t *testing.T) {
	zeroUUID := uuid.UUID{} // 00000000-0000-0000-0000-000000000000 is what gorm reflection gives back when Oracle holds NULL
	dest := &afterQueryUUIDPtrModel{UUID: &zeroUUID}
	oracle.AfterQuery(buildDBForAfterQuery(t, dest))

	if dest.UUID != nil {
		t.Errorf("expected UUID to be nil for a zero UUID value, got %v", dest.UUID)
	}
}

// UUID pointer - ensure non-nil pointer to a non-zero UUID is left unchanged
func TestAfterQuery_UUIDPtr_NonZeroUUID_Unchanged(t *testing.T) {
	id := uuid.New()
	dest := &afterQueryUUIDPtrModel{UUID: &id}
	oracle.AfterQuery(buildDBForAfterQuery(t, dest))

	if dest.UUID == nil {
		t.Fatal("expected UUID to remain non-nil for a non-zero UUID value")
	}
	if *dest.UUID != id {
		t.Errorf("expected UUID %v, got %v", id, *dest.UUID)
	}
}

// UUID pointer - ensure nil (already representing NULL) is left unchanged and does not panic
func TestAfterQuery_UUIDPtr_NilUUID_RemainsNil(t *testing.T) {
	dest := &afterQueryUUIDPtrModel{UUID: nil}
	oracle.AfterQuery(buildDBForAfterQuery(t, dest))

	if dest.UUID != nil {
		t.Errorf("expected UUID to remain nil, got %v", dest.UUID)
	}
}

// UUID value - ensure the 00000000-0000-0000-0000-000000000000 UUID is left unchanged as zero UUID
func TestAfterQuery_UUIDValue_ZeroUUID_Unchanged(t *testing.T) {
	zeroUUID := uuid.UUID{} // 00000000-0000-0000-0000-000000000000 is what gorm reflection gives back when Oracle holds NULL
	dest := &afterQueryUUIDValueModel{UUID: zeroUUID}
	oracle.AfterQuery(buildDBForAfterQuery(t, dest))

	if dest.UUID != zeroUUID {
		t.Errorf("expected UUID to remain unchanged as zero UUID, got %v", dest.UUID)
	}
}

// UUID value - ensure non-zero UUID is left unchanged
func TestAfterQuery_UUIDValue_NonZeroUUID_Unchanged(t *testing.T) {
	id := uuid.New()
	dest := &afterQueryUUIDValueModel{UUID: id}
	oracle.AfterQuery(buildDBForAfterQuery(t, dest))

	if dest.UUID != id {
		t.Errorf("expected UUID %v, got %v", id, dest.UUID)
	}
}

// ensure model without any uuid-typed fields does not panic
func TestAfterQuery_NoUUIDFields(t *testing.T) {
	dest := &afterQueryNonUUIDModel{ID: 1, Name: "hello"}
	oracle.AfterQuery(buildDBForAfterQuery(t, dest))

	if dest.Name != "hello" {
		t.Errorf("expected Name to be 'hello', got %q", dest.Name)
	}
}
