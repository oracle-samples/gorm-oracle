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

var DB godror.Execer

// StringList is a Go wrapper for Oracle VARRAY type
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

	// detect Oracle type name dynamically via reflection
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

func detectOracleTypeName(value interface{}) (string, error) {
	rt := reflect.TypeOf(value)
	if rt.Kind() != reflect.Struct {
		// Not a struct → cannot detect; return a hint instead of panic
		return "", fmt.Errorf("detectOracleTypeName: not a struct (got %s)", rt.Kind())
	}

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get("gorm")
		if strings.Contains(tag, "type:") {
			re := regexp.MustCompile(`type:"?([a-zA-Z0-9_]+)"?`)
			match := re.FindStringSubmatch(tag)
			if len(match) > 1 {
				return match[1], nil
			}
		}
	}

	return "", fmt.Errorf("no type tag found for %T", value)
}
