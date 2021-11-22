package paginator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// CursorEncoder encoder for cursor
type CursorEncoder interface {
	Encode(v interface{}) string
}

// NewCursorEncoder creates cursor encoder
func NewCursorEncoder(keys ...string) CursorEncoder {
	return &cursorEncoder{keys}
}

type cursorEncoder struct {
	keys []string
}

func (e *cursorEncoder) Encode(v interface{}) string {
	return base64.StdEncoding.EncodeToString(e.marshalJSON(v))
}

func (e *cursorEncoder) marshalJSON(value interface{}) []byte {
	rv := toReflectValue(value)
	// reduce reflect value to underlying value
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	fields := make([]interface{}, len(e.keys))
	for i, key := range e.keys {
		fields[i] = rv.FieldByName(key).Interface()
	}
	// @TODO: return proper error
	b, _ := json.Marshal(fields)
	return b
}

/* deprecated */

func encodeOld(rv reflect.Value, keys []string) string {
	fields := make([]string, len(keys))
	for index, key := range keys {
		if rv.Kind() == reflect.Ptr {
			fields[index] = convert(reflect.Indirect(rv).FieldByName(key).Interface())
		} else {
			fields[index] = convert(rv.FieldByName(key).Interface())
		}
	}
	return base64.StdEncoding.EncodeToString([]byte(strings.Join(fields, ",")))
}

func convert(field interface{}) string {
	switch field.(type) {
	case time.Time:
		return fmt.Sprintf("%s?%s", field.(time.Time).UTC().Format(time.RFC3339Nano), fieldTime)
	default:
		return fmt.Sprintf("%v?%s", field, fieldString)
	}
}
