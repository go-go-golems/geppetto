package helpers

import (
	"github.com/pkg/errors"
	"reflect"
	"strings"
)

func FillStructFromKV(kv map[string]string, v interface{}) error {
	// go over the struct, and for each field with a struct tag kv,
	// fill with the value from the kv. If the field is not marked optional, and the
	// value is missing from the kv, or empty, return an error

	// check that v is a pointer to a struct
	if reflect.TypeOf(v).Kind() != reflect.Ptr {
		return errors.New("v is not a pointer")
	}
	if reflect.TypeOf(v).Elem().Kind() != reflect.Struct {
		return errors.New("v is not a pointer to a struct")
	}

	// go over the struct fields
	s := reflect.ValueOf(v).Elem()
	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)
		fieldType := s.Type().Field(i)

		kvTag := fieldType.Tag.Get("kv")
		if kvTag == "" {
			continue
		}

		// check if the field is optional
		optional := false
		if strings.HasSuffix(kvTag, ",optional") {
			optional = true
			kvTag = strings.TrimSuffix(kvTag, ",optional")
		}

		// check if the field is present in the kv
		value, ok := kv[kvTag]
		if !ok {
			if !optional {
				return errors.Errorf("field %s is not optional, but not present in kv", kvTag)
			}
			continue
		}

		// check if the field is empty
		if value == "" {
			if !optional {
				return errors.Errorf("field %s is not optional, but empty in kv", kvTag)
			}
			continue
		}

		// set the field
		field.SetString(value)
	}

	return nil
}

// ParseKV transforms a simple format. Each line has a key value pair separated by :.
// Empty lines are ignored
func ParseKV(s string) map[string]string {
	m := make(map[string]string)

	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		m[parts[0]] = parts[1]
	}

	return m
}
