package oracle

import (
	"context"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/godror/godror"
)

var DB godror.Execer // set in tests

// EmailList is a Go wrapper for Oracle VARRAY type EMAIL_LIST_ARR.
type StringList []string

// Scan implements sql.Scanner (Oracle -> Go)
func (e *StringList) Scan(src interface{}) error {
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
func (s StringList) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	if DB == nil {
		return nil, fmt.Errorf("oracle.DB not initialized")
	}

	ctx := context.Background()

	// Try to detect Oracle type name dynamically via reflection
	typeName, err := detectOracleTypeName(s)
	if err != nil {
		return nil, err
	}

	objType, err := godror.GetObjectType(ctx, DB, fmt.Sprintf("\"%s\"", typeName))
	if err != nil {
		return nil, fmt.Errorf("get object type %q: %w", typeName, err)
	}
	defer objType.Close()

	obj, err := objType.NewObject()
	if err != nil {
		return nil, fmt.Errorf("new object: %w", err)
	}
	coll := obj.Collection()

	for _, v := range s {
		if err := coll.Append(v); err != nil {
			obj.Close()
			return nil, fmt.Errorf("append: %w", err)
		}
	}

	return obj, nil
}

// detectOracleTypeName uses reflection to look up the `gorm:"type:..."` tag.
func detectOracleTypeName(value interface{}) (string, error) {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// walk up the call stack and look for the struct field tag
	// (GORM provides the value as part of the struct — this works during model serialization)
	rt := reflect.TypeOf(value)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if tag := field.Tag.Get("gorm"); strings.Contains(tag, "type:") {
			re := regexp.MustCompile(`type:"?([a-zA-Z0-9_]+)"?`)
			match := re.FindStringSubmatch(tag)
			if len(match) > 1 {
				return strings.ToUpper(match[1]), nil
			}
		}
	}

	// fallback
	return "", fmt.Errorf("cannot detect Oracle type name for %T", value)
}
