package paginator

import "github.com/pilagod/gorm-cursor-paginator/v2/internal/util"

// Rule for paginator
type Rule struct {
	Key             string
	Order           Order
	SQLRepr         string
	NULLReplacement interface{}
}

func (r *Rule) validate(dest interface{}) (err error) {
	if _, ok := util.ReflectType(dest).FieldByName(r.Key); !ok {
		return ErrInvalidModel
	}
	if r.Order != "" {
		if err = r.Order.validate(); err != nil {
			return
		}
	}
	return nil
}
