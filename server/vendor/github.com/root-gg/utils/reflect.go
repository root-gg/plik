package utils

import (
	"fmt"
	"reflect"
)

/*
 * Assign a map[string]interface{} to a struct mapping the map pairs to
 * the structure members by name using reflexion.
 */
func Assign(config interface{}, values map[string]interface{}) {
	s := reflect.ValueOf(config).Elem()
	t := reflect.TypeOf(config)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	for key, val := range values {
		if typ, ok := t.FieldByName(key); ok {
			s.FieldByName(key).Set(reflect.ValueOf(val).Convert(typ.Type))
		}
	}
}

/*
 * Transform []T to []interface{}
 */
func ToInterfaceArray(v interface{}) (res []interface{}) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(v)
		res = make([]interface{}, s.Len())
		for i := 0; i < s.Len(); i++ {
			res[i] = s.Index(i).Interface()
		}
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
	return res
}
