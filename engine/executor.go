package engine

import (
	"fmt"
	"simple-rdbms/parser"
	"simple-rdbms/value"
)

type Result struct {
	Columns []string
	Rows    [][]value.Value
}

func Execute(db *Database, node parser.Node) (*Result, error) {
	switch stmt := node.(type) {
	case *parser.SelectStatement:
		return executeSelect(db, stmt)
	case *parser.CreateStatement:
		return executeCreate(db, stmt)
	default:
		return nil, fmt.Errorf("未対応の文: %T", node)
	}
}

func executeCreate(db *Database, stmt *parser.CreateStatement) (*Result, error) {
	var columns []Column
	for _, col := range stmt.ColumnDef {
		var columnType value.Type
		switch col.TypeName {
		case "INT":
			columnType = value.TypeInt
		case "TEXT":
			columnType = value.TypeText
		default:
			return nil, fmt.Errorf("未対応の型: %q", col.TypeName)
		}
		columns = append(columns, Column{
			Name: col.ColumnName,
			Type: columnType,
		})
	}
	db.AddTable(stmt.Table, &Table{
		Columns: columns,
		Rows:    [][]value.Value{},
	})
	return nil, nil
}

func executeSelect(db *Database, stmt *parser.SelectStatement) (*Result, error) {
	table, err := db.GetTable(stmt.Table)
	if err != nil {
		return nil, err
	}

	if len(stmt.Columns) == 1 && stmt.Columns[0] == "*" {
		var names []string
		for _, col := range table.Columns {
			names = append(names, col.Name)
		}
		return &Result{
			Columns: names,
			Rows:    table.Rows,
		}, nil
	}

	indices, err := resolveColumnIndices(table, stmt.Columns)
	if err != nil {
		return nil, err
	}

	var rows [][]value.Value
	for _, row := range table.Rows {
		var picked []value.Value
		for _, idx := range indices {
			picked = append(picked, row[idx])
		}
		rows = append(rows, picked)
	}

	return &Result{
		Columns: stmt.Columns,
		Rows:    rows,
	}, nil
}

func resolveColumnIndices(table *Table, columns []string) ([]int, error) {
	var indices []int
	for _, col := range columns {
		found := false
		for i, tc := range table.Columns {
			if tc.Name == col {
				indices = append(indices, i)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("カラム %q が見つかりません", col)
		}
	}
	return indices, nil
}
