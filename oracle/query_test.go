package oracle

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestBeforeQuery_ConvertsVarsAndHandlesAlias(t *testing.T) {
	vTrue := true
	vFalse := false

	stmt := &gorm.Statement{
		TableExpr: &clause.Expr{SQL: `"users" "u"`},
		Vars:      []interface{}{true, false, &vTrue, &vFalse, (*bool)(nil), "name", 42},
	}

	BeforeQuery(&gorm.DB{Statement: stmt})

	if stmt.Table != "u" {
		t.Fatalf("expected alias table name %q, got %q", "u", stmt.Table)
	}

	for i, expected := range []int{1, 0, 1, 0} {
		got, ok := stmt.Vars[i].(int)
		if !ok || got != expected {
			t.Fatalf("expected vars[%d] to be int(%d), got %#v", i, expected, stmt.Vars[i])
		}
	}

	if v, ok := stmt.Vars[4].(*int); !ok || v != nil {
		t.Fatalf("expected vars[4] to be (*int)(nil), got %#v", stmt.Vars[4])
	}

	if got, ok := stmt.Vars[5].(string); !ok || got != "name" {
		t.Fatalf("expected vars[5] to stay string %q, got %#v", "name", stmt.Vars[5])
	}

	if got, ok := stmt.Vars[6].(int); !ok || got != 42 {
		t.Fatalf("expected vars[6] to stay int(42), got %#v", stmt.Vars[6])
	}
}

func TestBeforeQuery_ConvertsVarsWithoutTableExpr(t *testing.T) {
	stmt := &gorm.Statement{
		Vars: []interface{}{true},
	}

	BeforeQuery(&gorm.DB{Statement: stmt})

	if got, ok := stmt.Vars[0].(int); !ok || got != 1 {
		t.Fatalf("expected vars[0] to be int(1), got %#v", stmt.Vars[0])
	}
}

