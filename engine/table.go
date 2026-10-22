package engine

import (
	"fmt"
	"simple-rdbms/value"
)

type Column struct {
	Name string
	Type value.Type
}

type Table struct {
	Columns []Column
	Rows    [][]value.Value
}

type Database struct {
	tables map[string]*Table
}

func NewDatabase() *Database {
	return &Database{tables: make(map[string]*Table)}
}

func (db *Database) AddTable(name string, table *Table) {
	db.tables[name] = table
}

func (db *Database) GetTable(name string) (*Table, error) {
	t, ok := db.tables[name]
	if !ok {
		return nil, fmt.Errorf("テーブル %q が見つかりません", name)
	}
	return t, nil
}
