package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
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

//
// The following functions aim to override struct fields from string values
//

func IsPointer(kind reflect.Kind) bool {
	return kind == reflect.Ptr
}

func IsString(kind reflect.Kind) bool {
	return kind == reflect.String
}

func AssignStringString(elem reflect.Value, value string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign string value : %s", r)
		}
	}()

	elem.SetString(value)
	return nil
}

func IsBool(kind reflect.Kind) bool {
	return kind == reflect.Bool
}

func AssignBoolString(elem reflect.Value, value string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign bool value : %s", r)
		}
	}()

	if strings.ToLower(value) == "true" {
		elem.SetBool(true)
	} else if strings.ToLower(value) == "false" {
		elem.SetBool(false)
	} else {
		return fmt.Errorf("invalid boolean value")
	}
	return nil
}

var intKinds = []reflect.Kind{
	reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
}

func IsInt(kind reflect.Kind) bool {
	for _, k := range intKinds {
		if kind == k {
			return true
		}
	}
	return false
}

func AssignIntString(elem reflect.Value, value string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign int value : %s", r)
		}
	}()

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid int value")
	}

	elem.SetInt(intValue)

	return
}

var uintKinds = []reflect.Kind{
	reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
}

func IsUint(kind reflect.Kind) bool {
	for _, k := range uintKinds {
		if kind == k {
			return true
		}
	}
	return false
}

func AssignUintString(elem reflect.Value, value string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign uint value : %s", r)
		}
	}()

	uintValue, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid uint value")
	}

	elem.SetUint(uintValue)

	return
}

var floatKinds = []reflect.Kind{
	reflect.Float32, reflect.Float64,
}

func IsFloat(kind reflect.Kind) bool {
	for _, k := range floatKinds {
		if kind == k {
			return true
		}
	}
	return false
}

func AssignFloatString(elem reflect.Value, value string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign float value : %s", r)
		}
	}()

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fmt.Errorf("invalid float value")
	}

	elem.SetFloat(floatValue)

	return
}

func IsSlice(kind reflect.Kind) bool {
	return kind == reflect.Slice
}

func AssignJsonSliceString(elem reflect.Value, strValue string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign json string to slice : %s", r)
		}
	}()

	var sliceValue []interface{}
	err = json.Unmarshal([]byte(strValue), &sliceValue)
	if err != nil {
		return fmt.Errorf("invalid slice value : %s", err)
	}

	return AssignSliceValues(elem, sliceValue)
}

func AssignSliceValues(elem reflect.Value, values []interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign slice values : %s", r)
		}
	}()

	sliceType := elem.Type().Elem()
	slice := reflect.Zero(reflect.SliceOf(sliceType)).Interface()
	elem.Set(reflect.ValueOf(slice))

	for _, value := range values {
		elem.Set(reflect.Append(elem, reflect.ValueOf(value).Convert(sliceType)))
	}

	return nil
}

func IsMap(kind reflect.Kind) bool {
	return kind == reflect.Map
}

func AssignJsonMapString(elem reflect.Value, strValue string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign json string to map : %s", r)
		}
	}()

	mapValues := make(map[string]interface{})
	err = json.Unmarshal([]byte(strValue), &mapValues)
	if err != nil {
		return fmt.Errorf("invalid map value : %s", err)
	}

	return AssignMapValues(elem, mapValues)
}

func AssignMapValues(elem reflect.Value, values map[string]interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unable to assign map values : %s", r)
		}
	}()

	if elem.IsNil() {
		elem.Set(reflect.MakeMap(elem.Type()))
	}

	for key, val := range values {
		elem.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
	}

	return nil
}

func AssignString(elem reflect.Value, value string) (err error) {
	kind := elem.Kind()

	if IsString(kind) {
		return AssignStringString(elem, value)
	}

	if IsBool(kind) {
		return AssignBoolString(elem, value)
	}

	if IsInt(kind) {
		return AssignIntString(elem, value)
	}

	if IsUint(kind) {
		return AssignUintString(elem, value)
	}

	if IsFloat(kind) {
		return AssignFloatString(elem, value)
	}

	if IsSlice(kind) {
		return AssignJsonSliceString(elem, value)
	}

	if IsMap(kind) {
		return AssignJsonMapString(elem, value)
	}

	return fmt.Errorf("unsupported assignment for type %s", kind.String())
}

// This function will call getStringValue with the name of each Exported/Public field of the given object
// If getStringValue return ok == true : then it will try to assign the returned string value will to the corresponding field
// If getStringValue return ok == false : the field is skipped
//
// For slice and maps types the string is deserialized from json
// This can be useful to override config from environment variables
// /!\ No support for pointers or custom structs /!\
func AssignStrings(value interface{}, getStringValue func(bool string) (strValue string, ok bool)) (err error) {
	val := reflect.ValueOf(value).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		elem := val.Field(i)

		// Skip unexported/private fields
		if len(field.PkgPath) != 0 {
			continue
		}

		stringValue, ok := getStringValue(field.Name)
		if !ok {
			continue
		}

		err = AssignString(elem, stringValue)
		if err != nil {
			return fmt.Errorf("unable to assign %s : %s", field.Name, err)
		}
	}
	return nil
}
