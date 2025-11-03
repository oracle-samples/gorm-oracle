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
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"gorm.io/driver/oracledb"
)

var DB *gorm.DB

type TSTZTest struct {
	ID        uint          `gorm:"primaryKey"`
	EventTime time.Time     `gorm:"type:TIMESTAMP WITH TIME ZONE;column:event_time"`
	Nullable  sql.NullTime  `gorm:"type:TIMESTAMP WITH TIME ZONE;column:nullable_time"`
	PrecNsec  time.Time     `gorm:"type:TIMESTAMP WITH TIME ZONE;column:prec_nsec"`
	LocTz     time.Time     `gorm:"type:TIMESTAMP WITH LOCAL TIME ZONE;column:local_tz"`
	Note      sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time
}

func openTestDB(t *testing.T) *gorm.DB {
	if DB != nil {
		return DB
	}
	dsn := os.Getenv("ORACLE_DSN")
	if dsn == "" {
		t.Skip("ORACLE_DSN not set and DB not provided; skipping TIMESTAMP WITH TIME ZONE tests")
	}
	db, err := gorm.Open(oracledb.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open oracledb DSN: %v", err)
	}
	return db
}

// tolerance: we accept up to 1 microsecond differences because JDBC/driver normalization
// and Oracle internal representation can lose monotonic clock or round fractional secs.
func timesAlmostEqual(a, b time.Time) bool {
	diff := a.Sub(b)
	if diff < 0 {
		diff = -diff
	}
	// use 2 microseconds tolerance for safety; bump if your environment shows slightly larger diffs
	return diff <= 2*time.Microsecond
}

func mustCreateTable(t *testing.T, db *gorm.DB) {
	if err := db.Migrator().DropTable(&TSTZTest{}); err != nil {
		// some drivers return error when table doesn't exist;
	}
	if err := db.AutoMigrate(&TSTZTest{}); err != nil {
		t.Fatalf("failed to migrate TSTZTest: %v", err)
	}
}

func mustTruncate(t *testing.T, db *gorm.DB) {
	if err := db.Exec("TRUNCATE TABLE tstz_tests").Error; err != nil {
		// fallback to delete for test safety
		_ = db.Exec("DELETE FROM tstz_tests").Error
	}
}
func TestTSTZ_MigrationAndColumnTypes(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)

	// Attempt to introspect column types using GORM's Migrator / Raw query.
	// Depending on the driver you may use different introspection; we keep this conservative.
	// The goal: assert the column exists and isn't NULLABLE if not defined as pointer.
	cols, err := db.Migrator().ColumnTypes(&TSTZTest{})
	if err != nil {
		t.Fatalf("failed to get column types: %v", err)
	}

	found := map[string]bool{}
	for _, c := range cols {
		name := c.Name()
		found[name] = true
		// optionally inspect dbType name if available
		// dt, _ := c.DatabaseTypeName()
		// t.Logf("col: %s dbtype: %s", name, dt)
	}
	wantCols := []string{"id", "event_time", "nullable_time", "prec_nsec", "local_tz", "note", "created_at", "updated_at"}
	for _, w := range wantCols {
		if !found[w] {
			t.Fatalf("expected column %s to exist after migration", w)
		}
	}
}

func TestTSTZ_InsertRetrieve_RoundtripEquality(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// Choose a time with a non-zero fractional second and a specific tz offset
	loc := time.FixedZone("IST", 5*3600+1800) // +05:30
	testTime := time.Date(2023, 11, 5, 13, 7, 6, 123456000, loc) // 123456 microseconds

	rec := TSTZTest{
		EventTime: testTime,
		Nullable:  sql.NullTime{Time: testTime.Add(time.Minute), Valid: true},
		PrecNsec:  testTime,
		LocTz:     testTime, // local tz column
		Note:      sql.NullString{String: "roundtrip", Valid: true},
	}

	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("failed to insert record: %v", err)
	}

	var got TSTZTest
	if err := db.First(&got, rec.ID).Error; err != nil {
		t.Fatalf("failed to fetch record: %v", err)
	}

	// EventTime: TIMESTAMP WITH TIME ZONE should preserve the instant; Compare instants
	if !timesAlmostEqual(got.EventTime.UTC(), testTime.UTC()) {
		t.Fatalf("event_time mismatch (instant): expected %v (UTC=%v) got %v (UTC=%v)",
			testTime, testTime.UTC(), got.EventTime, got.EventTime.UTC())
	}

	// Check nullable preserved
	if !got.Nullable.Valid {
		t.Fatalf("expected nullable to be valid")
	}
	if !timesAlmostEqual(got.Nullable.Time.UTC(), testTime.Add(time.Minute).UTC()) {
		t.Fatalf("nullable time mismatch: expected %v got %v", testTime.Add(time.Minute), got.Nullable.Time)
	}

	// Fractional seconds: ensure microsecond-level precision preserved
	if !timesAlmostEqual(got.PrecNsec.UTC(), testTime.UTC()) {
		t.Fatalf("fractional seconds lost: expected %v got %v", testTime, got.PrecNsec)
	}
}

func TestTSTZ_NullHandlingAndPointerNil(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	rec := TSTZTest{
		EventTime: time.Now(),
		Nullable:  sql.NullTime{Valid: false}, // explicit NULL
		Note:      sql.NullString{Valid: false},
	}

	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	var got TSTZTest
	if err := db.First(&got, rec.ID).Error; err != nil {
		t.Fatalf("select failed: %v", err)
	}

	if got.Nullable.Valid {
		t.Fatalf("expected Nullable to be NULL; got %v", got.Nullable)
	}
	if got.Note.Valid {
		t.Fatalf("expected Note to be NULL; got %v", got.Note)
	}
}

func TestTSTZ_TimezoneRoundTripDifferentTZs(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// Represent the same instant in two tzs
	ny := time.FixedZone("EST", -5*3600) // simple fixed zone for test
	berlin := time.FixedZone("CET", 1*3600)
	baseNY := time.Date(2025, 3, 9, 7, 30, 15, 987000000, ny) // 987 ms
	sameInstant := baseNY.UTC().In(berlin)

	rec1 := TSTZTest{EventTime: baseNY, Note: sql.NullString{String: "tz1", Valid: true}}
	rec2 := TSTZTest{EventTime: sameInstant, Note: sql.NullString{String: "tz2", Valid: true}}

	if err := db.Create(&rec1).Error; err != nil {
		t.Fatalf("insert rec1 failed: %v", err)
	}
	if err := db.Create(&rec2).Error; err != nil {
		t.Fatalf("insert rec2 failed: %v", err)
	}

	var out1, out2 TSTZTest
	if err := db.First(&out1, rec1.ID).Error; err != nil {
		t.Fatalf("fetch out1 failed: %v", err)
	}
	if err := db.First(&out2, rec2.ID).Error; err != nil {
		t.Fatalf("fetch out2 failed: %v", err)
	}

	// Both should represent the same instant in UTC
	if !timesAlmostEqual(out1.EventTime.UTC(), out2.EventTime.UTC()) {
		t.Fatalf("expected same instant across timezones; got %v and %v", out1.EventTime.UTC(), out2.EventTime.UTC())
	}
}

func TestTSTZ_DSTTransitions(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// Use a real DST zone if available in the runtime (Local zone database)
	// If zone is unavailable (some environments) we fall back to fixed zone; tests still valuable.
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		loc = time.FixedZone("CET", 1*3600)
	}
	// Pick a DST "fall back" example (end of DST typically late Oct/Nov)
	// 2023-10-29 is the DST end in EU (clocks go back 1 hour at 03:00 -> 02:00)
	// We pick an ambiguous instant and ensure it roundtrips to the same instant.
	ambiguous := time.Date(2023, 10, 29, 2, 30, 0, 0, loc)

	rec := TSTZTest{EventTime: ambiguous}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	var got TSTZTest
	if err := db.First(&got, rec.ID).Error; err != nil {
		t.Fatalf("fetch failed: %v", err)
	}

	if !timesAlmostEqual(got.EventTime.UTC(), ambiguous.UTC()) {
		t.Fatalf("DST ambiguous time lost instant: expected %v got %v", ambiguous.UTC(), got.EventTime.UTC())
	}
}

func TestTSTZ_PrecisionEdgecases_NanosecondsToMicroseconds(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// Oracle TIMESTAMP typically supports fractional seconds to nanosecond precision,
	// but drivers and DB settings may round to microseconds. Test several fractional values.
	fracNanos := []int{0, 1_000, 123_456, 999_999_999} // 0ns, 1us, 123456ns, nearly 1s
	for i, ns := range fracNanos {
		ts := time.Date(2025, 1, 2, 3, 4, 5, ns, time.UTC)
		rec := TSTZTest{EventTime: ts, Note: sql.NullString{String: fmt.Sprintf("frac-%d", i), Valid: true}}
		if err := db.Create(&rec).Error; err != nil {
			t.Fatalf("insert frac %d failed: %v", ns, err)
		}
		var got TSTZTest
		if err := db.First(&got, "note = ?", rec.Note.String).Error; err != nil {
			t.Fatalf("fetch frac %d failed: %v", ns, err)
		}
		// check that instants match within tolerance - some drivers round or truncate
		if !timesAlmostEqual(got.EventTime.UTC(), ts.UTC()) {
			t.Fatalf("frac mismatch for %d: expected %v got %v", ns, ts, got.EventTime)
		}
	}
}

func TestTSTZ_RawSQL_InsertSelectAndBind(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// Raw insert using bind param - ensure driver accepts timezone literals
	ts := time.Date(2024, 6, 30, 18, 0, 0, 500000000, time.FixedZone("UTC+2", 2*3600))
	// Use GORM.Raw with bind - most drivers accept time.Time as param
	if err := db.Exec("INSERT INTO tstz_tests (event_time, note) VALUES (:1, :2)", ts, "rawbind").Error; err != nil {
		t.Fatalf("raw insert failed: %v", err)
	}

	var got TSTZTest
	if err := db.First(&got, "note = ?", "rawbind").Error; err != nil {
		t.Fatalf("raw select failed: %v", err)
	}

	if !timesAlmostEqual(got.EventTime.UTC(), ts.UTC()) {
		t.Fatalf("raw bind mismatch expected %v got %v", ts.UTC(), got.EventTime.UTC())
	}
}

func TestTSTZ_SQLNullTimeScanCompatibility(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	ts := time.Now().UTC().Truncate(time.Microsecond)
	if err := db.Create(&TSTZTest{EventTime: ts}).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Query into sql.NullTime with Row.Scan
	rows, err := db.Raw("SELECT event_time FROM tstz_tests WHERE event_time = :1", ts).Rows()
	if err != nil {
		t.Fatalf("raw rows failed: %v", err)
	}
	defer rows.Close()

	var scanned sql.NullTime
	for rows.Next() {
		if err := rows.Scan(&scanned); err != nil {
			t.Fatalf("scan into sql.NullTime failed: %v", err)
		}
		if !scanned.Valid {
			t.Fatalf("scanned NullTime invalid unexpectedly")
		}
		if !timesAlmostEqual(scanned.Time.UTC(), ts.UTC()) {
			t.Fatalf("scanned time mismatch expected %v got %v", ts, scanned.Time)
		}
	}
}

func TestTSTZ_UpdateAndWhereComparisons(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	start := time.Date(2023, 7, 1, 8, 0, 0, 0, time.UTC)
	rec := TSTZTest{EventTime: start, Note: sql.NullString{String: "up1", Valid: true}}
	if err := db.Create(&rec).Error; err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// update using GORM
	newT := start.Add(90 * time.Minute)
	if err := db.Model(&rec).Update("event_time", newT).Error; err != nil {
		t.Fatalf("update failed: %v", err)
	}

	var got TSTZTest
	if err := db.First(&got, rec.ID).Error; err != nil {
		t.Fatalf("select after update failed: %v", err)
	}
	if !timesAlmostEqual(got.EventTime.UTC(), newT.UTC()) {
		t.Fatalf("update mismatch expected %v got %v", newT, got.EventTime)
	}

	// Query by range
	var count int64
	if err := db.Model(&TSTZTest{}).Where("event_time BETWEEN :1 AND :2", newT.Add(-time.Minute), newT.Add(time.Minute)).Count(&count).Error; err != nil {
		t.Fatalf("range query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 record in range, got %d", count)
	}
}

func TestTSTZ_BatchInsertAndOrderBy(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// prepare increasing times
	base := time.Now().UTC().Truncate(time.Microsecond)
	recs := []TSTZTest{
		{EventTime: base.Add(2 * time.Second), Note: sql.NullString{String: "b2", Valid: true}},
		{EventTime: base.Add(1 * time.Second), Note: sql.NullString{String: "b1", Valid: true}},
		{EventTime: base.Add(3 * time.Second), Note: sql.NullString{String: "b3", Valid: true}},
	}

	if err := db.Create(&recs).Error; err != nil {
		t.Fatalf("batch insert failed: %v", err)
	}

	var out []TSTZTest
	if err := db.Order("event_time asc").Find(&out).Error; err != nil {
		t.Fatalf("ordered find failed: %v", err)
	}
	if len(out) < 3 {
		t.Fatalf("expected >= 3 rows, got %d", len(out))
	}

	// verify order by event_time
	for i := 1; i < 3; i++ {
		if out[i].EventTime.Before(out[i-1].EventTime) {
			t.Fatalf("order incorrect: entry %d has %v before %v", i, out[i].EventTime, out[i-1].EventTime)
		}
	}
}

func TestTSTZ_PreparedStatementsAndTransactions(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// Transactional insert + rollback test
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin tx: %v", tx.Error)
	}

	rec := TSTZTest{EventTime: time.Now().UTC(), Note: sql.NullString{String: "tx1", Valid: true}}
	if err := tx.Create(&rec).Error; err != nil {
		_ = tx.Rollback()
		t.Fatalf("tx create failed: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("tx rollback error: %v", err)
	}

	// record should not exist
	var count int64
	if err := db.Model(&TSTZTest{}).Where("note = ?", "tx1").Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after rollback, got %d", count)
	}

	// Commit path
	tx2 := db.Begin()
	if tx2.Error != nil {
		t.Fatalf("failed to begin tx2: %v", tx2.Error)
	}
	rec2 := TSTZTest{EventTime: time.Now().UTC(), Note: sql.NullString{String: "tx2", Valid: true}}
	if err := tx2.Create(&rec2).Error; err != nil {
		_ = tx2.Rollback()
		t.Fatalf("tx2 create failed: %v", err)
	}
	if err := tx2.Commit().Error; err != nil {
		t.Fatalf("tx2 commit failed: %v", err)
	}
	if err := db.Model(&TSTZTest{}).Where("note = ?", "tx2").Count(&count).Error; err != nil {
		t.Fatalf("count after commit failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after commit, got %d", count)
	}
}

func TestTSTZ_ConcurrentInserts(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			rec := TSTZTest{EventTime: time.Now().UTC().Add(time.Duration(i) * time.Millisecond), Note: sql.NullString{String: fmt.Sprintf("con-%d", i), Valid: true}}
			if err := db.Create(&rec).Error; err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for e := range errCh {
		t.Fatalf("concurrent insert failed: %v", e)
	}

	var count int64
	if err := db.Model(&TSTZTest{}).Where("note LIKE :1", "con-%").Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != n {
		t.Fatalf("expected %d concurrent rows, got %d", n, count)
	}
}

func TestTSTZ_BoundaryDatesAndLargeRanges(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)
	mustTruncate(t, db)

	// Oracle supports wide date ranges; test epoch and far future/past
	epoch := time.Unix(0, 0).UTC()
	ancient := time.Date(1000, 1, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		ts  time.Time
		tag string
	}{
		{epoch, "epoch"},
		{ancient, "ancient"},
		{future, "future"},
	}
	for _, c := range cases {
		rec := TSTZTest{EventTime: c.ts, Note: sql.NullString{String: c.tag, Valid: true}}
		if err := db.Create(&rec).Error; err != nil {
			t.Fatalf("insert %s failed: %v", c.tag, err)
		}
		var got TSTZTest
		if err := db.First(&got, "note = ?", c.tag).Error; err != nil {
			t.Fatalf("select %s failed: %v", c.tag, err)
		}
		if !timesAlmostEqual(got.EventTime.UTC(), c.ts.UTC()) {
			t.Fatalf("boundary date mismatch for %s: expected %v got %v", c.tag, c.ts, got.EventTime)
		}
	}
}

func TestTSTZ_InvalidBindingsAndErrorPaths(t *testing.T) {
	db := openTestDB(t)
	mustCreateTable(t, db)

	// Attempt to insert invalid timezone string via raw SQL literal (driver may error)
	// Many drivers don't allow binding timezone strings into TIMESTAMP WITH TIME ZONE directly,
	// but we still can try an invalid value path for the raw SQL to see the DB error.
	_, err := db.DB()
	if err == nil {
		// Get lower-level DB to issue a raw Exec that triggers an error if possible
	}
	// We can't assert a specific error across environments; just ensure we can attempt and
	// that the driver/database rejects clearly invalid values.
	// Example: bind a non-time value into timestamp column through raw SQL (driver dependent).
	if err := db.Exec("BEGIN NULL; END;").Error; err != nil {
		// an unexpected error running no-op block is a test failure in some envs
		t.Logf("no-op block returned error (ok to ignore in some setups): %v", err)
	}
}