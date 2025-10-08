package oracle

import (
	"context"
	"database/sql/driver"
	"fmt"

	"github.com/godror/godror"
)

var DB godror.Execer // set in tests

// EmailList is a Go wrapper for Oracle VARRAY type EMAIL_LIST_ARR.
type EmailList []string

// Scan implements sql.Scanner (Oracle -> Go)
func (e *EmailList) Scan(src interface{}) error {
	obj, ok := src.(*godror.Object)
	if !ok {
		return fmt.Errorf("expected *godror.Object, got %T", src)
	}
	defer obj.Close()

	coll := obj.Collection()
	length, err := coll.Len()
	if err != nil {
		return fmt.Errorf("get collection length: %w", err)
	}

	var list []string
	for i := 0; i < length; i++ {
		var data godror.Data
		if err := coll.GetItem(&data, i); err != nil {
			return fmt.Errorf("GetItem %d: %w", i, err)
		}
		val := data.Get()
		switch v := val.(type) {
		case string:
			list = append(list, v)
		case []byte:
			list = append(list, string(v))
		default:
			if v != nil {
				list = append(list, fmt.Sprint(v))
			}
		}
	}
	*e = list
	return nil
}

// Value implements driver.Valuer (Go -> Oracle)
func (e EmailList) Value() (driver.Value, error) {
	if e == nil {
		return nil, nil
	}
	if DB == nil {
		return nil, fmt.Errorf("oracle.DB not initialized")
	}

	ctx := context.Background()

	objType, err := godror.GetObjectType(ctx, DB, "EMAIL_LIST_ARR")
	if err != nil {
		return nil, fmt.Errorf("get object type: %w", err)
	}
	defer objType.Close()

	obj, err := objType.NewObject()
	if err != nil {
		return nil, fmt.Errorf("new object: %w", err)
	}
	coll := obj.Collection()

	for _, s := range e {
		if err := coll.Append(s); err != nil {
			obj.Close()
			return nil, fmt.Errorf("append: %w", err)
		}
	}

	return obj, nil
}
