package paginator

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/gorm"
)

const (
	defaultLimit = 10
	defaultOrder = DESC
)

// New inits paginator
func New() *Paginator {
	return &Paginator{}
}

// Paginator a builder doing pagination
type Paginator struct {
	cursor    Cursor
	next      Cursor
	keys      []string
	tableKeys []string
	limit     int
	order     Order
}

// SetAfterCursor sets paging after cursor
func (p *Paginator) SetAfterCursor(afterCursor string) {
	p.cursor.After = &afterCursor
}

// SetBeforeCursor sets paging before cursor
func (p *Paginator) SetBeforeCursor(beforeCursor string) {
	p.cursor.Before = &beforeCursor
}

// SetKeys sets paging keys
func (p *Paginator) SetKeys(keys ...string) {
	p.keys = append(p.keys, keys...)
}

// SetLimit sets paging limit
func (p *Paginator) SetLimit(limit int) {
	p.limit = limit
}

// SetOrder sets paging order
func (p *Paginator) SetOrder(order Order) {
	p.order = order
}

// GetNextCursor returns cursor for next pagination
func (p *Paginator) GetNextCursor() Cursor {
	return p.next
}

// Paginate paginates data
func (p *Paginator) Paginate(stmt *gorm.DB, out interface{}) *gorm.DB {
	p.initOptions()
	p.initTableKeys(stmt, out)
	result := p.appendPagingQuery(stmt, out).Find(out)
	// out must be a pointer or gorm will panic above
	elems := reflect.ValueOf(out).Elem()
	if elems.Kind() == reflect.Slice && elems.Len() > 0 {
		p.postProcess(out)
	}
	return result
}

/* private */

func (p *Paginator) initOptions() {
	if len(p.keys) == 0 {
		p.keys = append(p.keys, "ID")
	}
	if p.limit == 0 {
		p.limit = defaultLimit
	}
	if p.order == "" {
		p.order = defaultOrder
	}
}

func (p *Paginator) initTableKeys(db *gorm.DB, out interface{}) {
	table := db.NewScope(out).TableName()
	for _, key := range p.keys {
		p.tableKeys = append(p.tableKeys, fmt.Sprintf("%s.%s", table, strcase.ToSnake(key)))
	}
}

func (p *Paginator) appendPagingQuery(stmt *gorm.DB, out interface{}) *gorm.DB {
	decoder, _ := NewCursorDecoder(out, p.keys...)
	var fields []interface{}
	if p.hasAfterCursor() {
		fields = decoder.Decode(*p.cursor.After)
	} else if p.hasBeforeCursor() {
		fields = decoder.Decode(*p.cursor.Before)
	}
	if len(fields) > 0 {
		stmt = stmt.Where(
			p.getCursorQuery(),
			p.getCursorQueryArgs(fields)...,
		)
	}
	stmt = stmt.Limit(p.limit + 1)
	stmt = stmt.Order(p.getOrder())
	return stmt
}

func (p *Paginator) hasAfterCursor() bool {
	return p.cursor.After != nil
}

func (p *Paginator) hasBeforeCursor() bool {
	return !p.hasAfterCursor() && p.cursor.Before != nil
}

func (p *Paginator) getCursorQuery() string {
	qs := make([]string, len(p.tableKeys))
	op := p.getOperator()
	composite := ""
	for i, sqlKey := range p.tableKeys {
		qs[i] = fmt.Sprintf("%s%s %s ?", composite, sqlKey, op)
		composite = fmt.Sprintf("%s%s = ? AND ", composite, sqlKey)
	}
	return strings.Join(qs, " OR ")
}

func (p *Paginator) getCursorQueryArgs(fields []interface{}) (args []interface{}) {
	for i := 1; i <= len(fields); i++ {
		args = append(args, fields[:i]...)
	}
	return
}

func (p *Paginator) getOperator() string {
	if (p.hasAfterCursor() && p.order == ASC) ||
		(p.hasBeforeCursor() && p.order == DESC) {
		return ">"
	}
	return "<"
}

func (p *Paginator) getOrder() string {
	order := p.order
	if p.hasBeforeCursor() {
		order = flip(p.order)
	}
	orders := make([]string, len(p.tableKeys))
	for index, sqlKey := range p.tableKeys {
		orders[index] = fmt.Sprintf("%s %s", sqlKey, order)
	}
	return strings.Join(orders, ", ")
}

func (p *Paginator) postProcess(out interface{}) {
	elems := reflect.ValueOf(out).Elem()
	hasMore := elems.Len() > p.limit
	if hasMore {
		elems.Set(elems.Slice(0, elems.Len()-1))
	}
	if p.hasBeforeCursor() {
		elems.Set(reverse(elems))
	}
	encoder := NewCursorEncoder(p.keys...)
	if p.hasBeforeCursor() || hasMore {
		cursor := encoder.Encode(elems.Index(elems.Len() - 1))
		p.next.After = &cursor
	}
	if p.hasAfterCursor() || (hasMore && p.hasBeforeCursor()) {
		cursor := encoder.Encode(elems.Index(0))
		p.next.Before = &cursor
	}
	return
}

func reverse(v reflect.Value) reflect.Value {
	result := reflect.MakeSlice(v.Type(), 0, v.Cap())
	for i := v.Len() - 1; i >= 0; i-- {
		result = reflect.Append(result, v.Index(i))
	}
	return result
}
