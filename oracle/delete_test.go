package oracle

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBuildWhereClause_ExpandsInSlices(t *testing.T) {
	cases := []struct {
		name     string
		value    interface{}
		wantSQL  string
		wantVars []interface{}
	}{
		{"uint", []uint{1, 2, 3}, `"record_id" IN (:1, :2, :3)`, []interface{}{uint(1), uint(2), uint(3)}},
		{"int64", []int64{4, 5}, `"record_id" IN (:1, :2)`, []interface{}{int64(4), int64(5)}},
		{"string", []string{"a", "b"}, `"record_id" IN (:1, :2)`, []interface{}{"a", "b"}},
		{"interface", []interface{}{6, "x"}, `"record_id" IN (:1, :2)`, []interface{}{6, "x"}},
		{"array", [2]int{7, 8}, `"record_id" IN (:1, :2)`, []interface{}{7, 8}},
		{"empty", []uint{}, `"record_id" IN (NULL)`, nil},
		{"bytes", []byte("ab"), `"record_id" IN :1`, []interface{}{[]byte("ab")}},
		{"valuer", datatypes.JSON(`[1]`), `"record_id" IN :1`, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := &gorm.DB{Statement: &gorm.Statement{}}
			var sql strings.Builder
			buildWhereClause(db, &sql, []clause.Expression{
				clause.Expr{SQL: `"record_id" IN ?`, Vars: []interface{}{tc.value}},
			})

			if sql.String() != tc.wantSQL {
				t.Fatalf("expected SQL %q, got %q", tc.wantSQL, sql.String())
			}
			if tc.name == "valuer" {
				if len(db.Statement.Vars) != 1 {
					t.Fatalf("expected one bind var, got %#v", db.Statement.Vars)
				}
				return
			}
			if !reflect.DeepEqual(db.Statement.Vars, tc.wantVars) {
				t.Fatalf("expected vars %#v, got %#v", tc.wantVars, db.Statement.Vars)
			}
		})
	}
}
