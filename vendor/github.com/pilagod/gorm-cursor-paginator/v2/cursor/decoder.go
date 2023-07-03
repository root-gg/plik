package cursor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"reflect"

	"github.com/pilagod/gorm-cursor-paginator/v2/internal/util"
)

// NewDecoder creates cursor decoder for model
func NewDecoder(fields []DecoderField) *Decoder {
	return &Decoder{fields: fields}
}

// Decoder cursor decoder
type Decoder struct {
	fields []DecoderField
}

// DecoderField contains information about one decoder field.
type DecoderField struct {
	Key  string
	Type *reflect.Type
}

// Decode decodes cursor into values (without pointer) by referencing field type on model.
func (d *Decoder) Decode(cursor string, model interface{}) (fields []interface{}, err error) {
	if err = d.validate(model); err != nil {
		return
	}
	b, err := base64.StdEncoding.DecodeString(cursor)
	// ensure cursor content is json
	if err != nil || !json.Valid(b) {
		return nil, ErrInvalidCursor
	}
	jd := json.NewDecoder(bytes.NewBuffer(b))
	// ensure cursor content is json array
	if t, err := jd.Token(); err != nil || t != json.Delim('[') {
		return nil, ErrInvalidCursor
	}
	for _, field := range d.fields {
		// prefer field.Type when set; this is needed for getting the right type for custom types
		var t reflect.Type
		if field.Type != nil {
			t = *field.Type
		} else {
			// key is already validated at beginning
			f, _ := util.ReflectType(model).FieldByName(field.Key)
			t = f.Type
		}

		v := reflect.New(t).Interface()
		if err := jd.Decode(v); err != nil {
			return nil, ErrInvalidCursor
		}
		fields = append(fields, reflect.ValueOf(v).Elem().Interface())
	}
	// cursor must be a valid json after previous checks,
	// so no need to check whether "]" is the last token
	return
}

// DecodeStruct decodes cursor into model, model must be a pointer to struct or it will panic.
func (d *Decoder) DecodeStruct(cursor string, model interface{}) (err error) {
	fields, err := d.Decode(cursor, model)
	if err != nil {
		return
	}
	elem := reflect.ValueOf(model).Elem()
	for i, field := range d.fields {
		elem.FieldByName(field.Key).Set(reflect.ValueOf(fields[i]))
	}
	return
}

func (d *Decoder) validate(model interface{}) error {
	modelType := util.ReflectType(model)
	// model's underlying type must be a struct
	if modelType.Kind() != reflect.Struct {
		return ErrInvalidModel
	}
	for _, field := range d.fields {
		if _, ok := modelType.FieldByName(field.Key); !ok {
			return ErrInvalidModel
		}
	}
	return nil
}
