package parser

import "simple-rdbms/value"

// ADR-0002: interface + 具象構造体方式

type Node interface {
	nodeType() string
}

type SelectStatement struct {
	Columns []string // カラム名のリスト。"*" の場合は ["*"]
	Table   string
	Where   Condition // WHERE がなければ nil
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

// Condition は WHERE の比較条件と論理条件を表す。
type Condition interface {
	conditionType() string
}

type ComparisonOperator uint8

const (
	ComparisonInvalid ComparisonOperator = iota
	ComparisonEqual
	ComparisonNotEqual
	ComparisonLessThan
	ComparisonLessEqual
	ComparisonGreaterThan
	ComparisonGreaterEqual
)

type ComparisonCondition struct {
	Column   string
	Operator ComparisonOperator
	Value    value.Value
}

func (c *ComparisonCondition) conditionType() string { return "COMPARISON" }

type AndCondition struct {
	Left  Condition
	Right Condition
}

func (c *AndCondition) conditionType() string { return "AND" }

type OrCondition struct {
	Left  Condition
	Right Condition
}

func (c *OrCondition) conditionType() string { return "OR" }
