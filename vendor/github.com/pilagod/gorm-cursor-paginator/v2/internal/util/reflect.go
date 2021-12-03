package util

import (
	"reflect"
)

// ReflectValue returns reflect value underlying given value, unwrapping pointer and slice
func ReflectValue(v interface{}) reflect.Value {
	rv, ok := v.(reflect.Value)
	if !ok {
		rv = reflect.ValueOf(v)
	}
	for rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Slice {
		rv = rv.Elem()
	}
	return rv
}

// ReflectType returns reflect type underlying given value, unwrapping pointer and slice
func ReflectType(v interface{}) reflect.Type {
	var rt reflect.Type
	if rvt, ok := v.(reflect.Type); ok {
		rt = rvt
	} else {
		rv, ok := v.(reflect.Value)
		if !ok {
			rv = reflect.ValueOf(v)
		}
		rt = rv.Type()
	}
	for rt.Kind() == reflect.Ptr || rt.Kind() == reflect.Slice {
		rt = rt.Elem()
	}
	return rt
}
