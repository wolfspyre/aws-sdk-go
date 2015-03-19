package awsutil

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var indexRe = regexp.MustCompile(`(.+)\[(-?\d+)\]$`)
var rnil = reflect.ValueOf(nil)

func rValueAtPath(i interface{}, path string, create bool) reflect.Value {
	value := reflect.Indirect(reflect.ValueOf(i))
	components := strings.Split(path, ".")
	for len(components) > 0 {
		var index *int64
		c := components[0]
		if c == "" { // no actual component, illegal syntax
			return rnil
		} else if strings.ToLower(c[0:1]) == c[0:1] {
			// TODO normalize case for user
			return rnil // don't support unexported fields
		}

		if m := indexRe.FindStringSubmatch(c); m != nil {
			c = m[1]
			i, _ := strconv.ParseInt(m[2], 10, 32)
			index = &i
		}

		// pull component name out of struct member
		if value.Kind() != reflect.Struct {
			return rnil
		}
		value = value.FieldByName(c)

		if create && value.Kind() == reflect.Ptr && value.IsNil() {
			value.Set(reflect.New(value.Type().Elem()))
			value = value.Elem()
		} else {
			value = reflect.Indirect(value)
		}

		// pull out index
		if index != nil {
			if value.Kind() != reflect.Slice {
				return rnil
			}
			i := int(*index)
			if i >= value.Len() { // check out of bounds
				if create {
					// TODO resize slice
				} else {
					return rnil
				}
			} else if i < 0 { // support negative indexing
				i = value.Len() + i
			}
			value = reflect.Indirect(value.Index(i))
		}

		if !value.IsValid() {
			return rnil
		}

		components = components[1:]
	}
	return value
}

// ValueAtPath returns an object at the lexical path inside of a structure
func ValueAtPath(i interface{}, path string) interface{} {
	if v := rValueAtPath(i, path, false); v.IsValid() {
		return v.Interface()
	}
	return nil
}

// SetValueAtPath sets an object at the lexical path inside of a structure
func SetValueAtPath(i interface{}, path string, v interface{}) {
	if rv := rValueAtPath(i, path, true); rv.IsValid() {
		rv.Set(reflect.ValueOf(v))
	}
}
