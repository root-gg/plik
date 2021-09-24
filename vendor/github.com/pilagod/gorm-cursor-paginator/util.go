package paginator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"time"
)

var (
	rfc3339 = regexp.MustCompile("^([0-9]+)-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])[Tt]([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9]|60)(\\.[0-9]+)?(([Zz])|([\\+|\\-]([01][0-9]|2[0-3]):[0-5][0-9]))$")
)

// Encode encodes properties in order defined by keys on the struct of v
// Deprecated: Encode will remove in v2, use CursorEncoder instead
func Encode(v reflect.Value, keys []string) string {
	return NewCursorEncoder(keys...).Encode(v)
}

// Decode decodes cursor into values in the same order as encoding
// Deprecated: Decode will remove in v2, use CursorDecoder instead
func Decode(cursor string) []interface{} {
	b, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil
	}
	var fields []interface{}
	err = json.Unmarshal(b, &fields)
	if err != nil {
		return decodeOld(b)
	}
	// ensure forward compatibility
	for i, field := range fields {
		value := fmt.Sprintf("%v", field)
		if rfc3339.Match([]byte(value)) {
			t, _ := time.Parse(time.RFC3339Nano, value)
			fields[i] = t
		} else {
			fields[i] = value
		}
	}
	return fields
}

/* util */

func toReflectValue(value interface{}) reflect.Value {
	rv, ok := value.(reflect.Value)
	if !ok {
		return reflect.ValueOf(value)
	}
	return rv
}
