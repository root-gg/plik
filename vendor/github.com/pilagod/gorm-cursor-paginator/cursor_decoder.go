package paginator

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"time"
)

// CursorDecoder decoder for cursor
type CursorDecoder interface {
	Decode(cursor string) []interface{}
}

// NewCursorDecoder creates cursor decoder
func NewCursorDecoder(ref interface{}, keys ...string) (CursorDecoder, error) {
	decoder := &cursorDecoder{keys: keys}
	err := decoder.initKeyKinds(ref)
	if err != nil {
		return nil, err
	}
	return decoder, nil
}

// Errors for decoders
var (
	ErrInvalidDecodeReference = errors.New("decode reference should be struct")
	ErrInvalidField           = errors.New("invalid field")
	ErrInvalidFieldType       = errors.New("invalid field type")
)

type kind uint

const (
	kindInvalid kind = iota
	kindBool
	kindInt
	kindUint
	kindFloat
	kindString
	kindTime
)

func toKind(rt reflect.Type) kind {
	// reduce reflect type to underlying value
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	// Kind() treats time.Time as struct, so we need specific test for time.Time
	if rt.ConvertibleTo(reflect.TypeOf(time.Time{})) {
		return kindTime
	}
	switch rt.Kind() {
	case reflect.Bool:
		return kindBool
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return kindInt
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return kindUint
	case reflect.Float32, reflect.Float64:
		return kindFloat
	default:
		return kindString
	}
}

type cursorDecoder struct {
	keys     []string
	keyKinds []kind
}

func (d *cursorDecoder) Decode(cursor string) []interface{} {
	b, err := base64.StdEncoding.DecodeString(cursor)
	// @TODO: return proper error
	if err != nil {
		return nil
	}
	var fields []interface{}
	err = json.Unmarshal(b, &fields)
	// ensure backward compatibility, should be deprecated in v2
	if err != nil {
		return decodeOld(b)
	}
	return d.castJSONFields(fields)
}

func (d *cursorDecoder) initKeyKinds(ref interface{}) error {
	// @TODO: zero value error
	rt := toReflectValue(ref).Type()
	// reduce reflect type to underlying struct
	for rt.Kind() == reflect.Slice || rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		// element of out must be struct, if not, just pass it to gorm to handle the error
		return ErrInvalidDecodeReference
	}
	d.keyKinds = make([]kind, len(d.keys))
	for i, key := range d.keys {
		field, ok := rt.FieldByName(key)
		if !ok {
			return ErrInvalidField
		}
		d.keyKinds[i] = toKind(field.Type)
	}
	return nil
}

func (d *cursorDecoder) castJSONFields(fields []interface{}) []interface{} {
	var result []interface{}
	for i, field := range fields {
		kind := d.keyKinds[i]
		switch f := field.(type) {
		case bool:
			bv, err := castJSONBool(f, kind)
			if err != nil {
				return nil
			}
			result = append(result, bv)
		case float64:
			fv, err := castJSONFloat(f, kind)
			if err != nil {
				return nil
			}
			result = append(result, fv)
		case string:
			sv, err := castJSONString(f, kind)
			if err != nil {
				return nil
			}
			result = append(result, sv)
		default:
			// invalid field
			return nil
		}
	}
	return result
}

func castJSONBool(value bool, kind kind) (interface{}, error) {
	if kind != kindBool {
		return nil, ErrInvalidFieldType
	}
	return value, nil
}

func castJSONFloat(value float64, kind kind) (interface{}, error) {
	switch kind {
	case kindInt:
		return int(value), nil
	case kindUint:
		return uint(value), nil
	case kindFloat:
		return value, nil
	}
	return nil, ErrInvalidFieldType
}

func castJSONString(value string, kind kind) (interface{}, error) {
	if kind != kindString && kind != kindTime {
		return nil, ErrInvalidFieldType
	}
	if kind == kindString {
		return value, nil
	}
	tv, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return nil, ErrInvalidFieldType
	}
	return tv, nil
}

/* deprecated */

func decodeOld(b []byte) []interface{} {
	fieldsWithType := strings.Split(string(b), ",")
	fields := make([]interface{}, len(fieldsWithType))
	for i, fieldWithType := range fieldsWithType {
		fields[i] = revert(fieldWithType)
	}
	return fields
}

type fieldType string

const (
	fieldString fieldType = "STRING"
	fieldTime   fieldType = "TIME"
)

func revert(fieldWithType string) interface{} {
	field, fieldType := parse(fieldWithType)
	switch fieldType {
	case fieldTime:
		t, err := time.Parse(time.RFC3339Nano, field)
		if err != nil {
			t = time.Now().UTC()
		}
		return t
	default:
		return field
	}
}

func parse(fieldWithType string) (string, fieldType) {
	sep := strings.LastIndex(fieldWithType, "?")
	field := fieldWithType[:sep]
	fieldType := fieldType(fieldWithType[sep+1:])
	return field, fieldType
}
