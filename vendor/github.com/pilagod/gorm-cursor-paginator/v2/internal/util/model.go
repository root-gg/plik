package util

import (
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func ParseSchema(db *gorm.DB, model interface{}) (*schema.Schema, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return nil, err
	}
	return stmt.Schema, nil
}
