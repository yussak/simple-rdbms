package engine

import (
	"fmt"
	"simple-rdbms/storage"
	"simple-rdbms/value"
)

type Column struct {
	Name string
	Type value.Type
}

type Table struct {
	Columns    []Column
	Rows       [][]value.Value
	relationID int64
}

type Database struct {
	tables         map[string]*Table
	store          *storage.Store
	closed         bool
	nextRelationID int64
}

func NewDatabase() *Database {
	return &Database{tables: make(map[string]*Table)}
}

func (db *Database) AddTable(name string, table *Table) {
	db.tables[name] = table
}

func (db *Database) GetTable(name string) (*Table, error) {
	if db.closed {
		return nil, fmt.Errorf("DBは既に閉じています")
	}
	t, ok := db.tables[name]
	if !ok {
		return nil, fmt.Errorf("テーブル %q が見つかりません", name)
	}
	return t, nil
}
