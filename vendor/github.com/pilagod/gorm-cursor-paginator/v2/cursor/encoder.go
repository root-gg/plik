package cursor

import (
	"encoding/base64"
	"encoding/json"
	"reflect"

	"github.com/pilagod/gorm-cursor-paginator/v2/internal/util"
)

// NewEncoder creates cursor encoder
func NewEncoder(keys ...string) *Encoder {
	return &Encoder{keys}
}

// Encoder cursor encoder
type Encoder struct {
	keys []string
}

// Encode encodes model into cursor
func (e *Encoder) Encode(model interface{}) (string, error) {
	b, err := e.marshalJSON(model)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (e *Encoder) marshalJSON(model interface{}) ([]byte, error) {
	rv := util.ReflectValue(model)
	fields := make([]interface{}, len(e.keys))
	for i, key := range e.keys {
		f := rv.FieldByName(key)
		if f == (reflect.Value{}) {
			return nil, ErrInvalidModel
		}
		if e.isNilable(f) && f.IsZero() {
			fields[i] = nil
		} else {
			fields[i] = util.ReflectValue(f).Interface()
		}
	}
	result, err := json.Marshal(fields)
	if err != nil {
		return nil, ErrInvalidModel
	}
	return result, nil
}

func (e *Encoder) isNilable(v reflect.Value) bool {
	return v.Kind() >= 18 && v.Kind() <= 23
}
