package parser

import "simple-rdbms/value"

// ADR-0002: interface + 具象構造体方式

type Node interface {
	nodeType() string
}

type SelectStatement struct {
	Columns []string // カラム名のリスト。"*" の場合は ["*"]
	Table   string
}

func (s *SelectStatement) nodeType() string { return "SELECT" }

type CreateStatement struct {
	ColumnDef []ColumnDef
	Table     string
}

func (s *CreateStatement) nodeType() string { return "CREATE" }

type InsertStatement struct {
	Table  string
	Values []value.Value
}

func (s *InsertStatement) nodeType() string { return "INSERT" }

type ColumnDef struct {
	ColumnName string
	TypeName   string
}
