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
	switch statement := node.(type) {
	case *parser.SelectStatement:
		return executeSelect(db, statement)
	case *parser.CreateStatement:
		return executeCreate(db, statement)
	case *parser.InsertStatement:
		return executeInsert(db, statement)
	default:
		return nil, fmt.Errorf("未対応の文: %T", node)
	}
}

func executeCreate(db *Database, statement *parser.CreateStatement) (*Result, error) {
	var columns []Column
	for _, col := range statement.ColumnDef {
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
	db.AddTable(statement.Table, &Table{
		Columns: columns,
		Rows:    [][]value.Value{},
	})
	return nil, nil
}

func executeInsert(db *Database, statement *parser.InsertStatement) (*Result, error) {
	table, err := db.GetTable(statement.Table)
	if err != nil {
		return nil, err
	}

	if len(statement.Values) != len(table.Columns) {
		return nil, fmt.Errorf(
			"テーブル %q の列数が一致しません: 期待=%d, 実際=%d",
			statement.Table, len(table.Columns), len(statement.Values),
		)
	}

	// 同じ位置にある列と値の型を照合する。
	for i, col := range table.Columns {
		if statement.Values[i].Type != col.Type {
			return nil, fmt.Errorf(
				"テーブル %q のカラム %q の型が一致しません",
				statement.Table, col.Name,
			)
		}
	}

	// 全検証の成功後、ASTとスライスを共有しない行を保存する。
	row := make([]value.Value, len(statement.Values))
	copy(row, statement.Values)
	table.Rows = append(table.Rows, row)
	return nil, nil
}

func executeSelect(db *Database, statement *parser.SelectStatement) (*Result, error) {
	table, err := db.GetTable(statement.Table)
	if err != nil {
		return nil, err
	}

	if len(statement.Columns) == 1 && statement.Columns[0] == "*" {
		var names []string
		for _, col := range table.Columns {
			names = append(names, col.Name)
		}
		return &Result{
			Columns: names,
			Rows:    table.Rows,
		}, nil
	}

	indices, err := resolveColumnIndices(table, statement.Columns)
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
		Columns: statement.Columns,
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
