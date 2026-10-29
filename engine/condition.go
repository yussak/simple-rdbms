package engine

import (
	"cmp"
	"fmt"

	"simple-rdbms/parser"
	"simple-rdbms/value"
)

type rowCondition interface {
	matches(row []value.Value) bool
}

type comparisonCondition struct {
	columnIndex int
	operator    parser.ComparisonOperator
	value       value.Value
}

func (c *comparisonCondition) matches(row []value.Value) bool {
	left := row[c.columnIndex]
	var order int
	if c.value.Type == value.TypeInt {
		// 差を計算せず、int64 の両端でもオーバーフローしない比較を行う。
		order = cmp.Compare(left.Int, c.value.Int)
	} else {
		order = cmp.Compare(left.Text, c.value.Text)
	}
	switch c.operator {
	case parser.ComparisonEqual:
		return order == 0
	case parser.ComparisonNotEqual:
		return order != 0
	case parser.ComparisonLessThan:
		return order < 0
	case parser.ComparisonLessEqual:
		return order <= 0
	case parser.ComparisonGreaterThan:
		return order > 0
	case parser.ComparisonGreaterEqual:
		return order >= 0
	default:
		// 未対応の演算子は準備時に拒否する。
		return false
	}
}

type andCondition struct {
	left  rowCondition
	right rowCondition
}

func (c *andCondition) matches(row []value.Value) bool {
	return c.left.matches(row) && c.right.matches(row)
}

type orCondition struct {
	left  rowCondition
	right rowCondition
}

func (c *orCondition) matches(row []value.Value) bool {
	return c.left.matches(row) || c.right.matches(row)
}

// 条件の木全体を検証し、列名を列位置に置き換えた実行用の条件を作る。
func prepareCondition(table *Table, condition parser.Condition) (rowCondition, error) {
	switch c := condition.(type) {
	case *parser.ComparisonCondition:
		if c == nil {
			return nil, fmt.Errorf("比較条件がありません")
		}
		indices, err := resolveColumnIndices(table, []string{c.Column})
		if err != nil {
			return nil, err
		}
		column := table.Columns[indices[0]]
		if column.Type != value.TypeInt && column.Type != value.TypeText {
			return nil, fmt.Errorf("カラム %q は比較できない型です", c.Column)
		}
		if column.Type != c.Value.Type {
			return nil, fmt.Errorf("カラム %q と比較する値の型が一致しません", c.Column)
		}
		switch c.Operator {
		case parser.ComparisonEqual, parser.ComparisonNotEqual,
			parser.ComparisonLessThan, parser.ComparisonLessEqual,
			parser.ComparisonGreaterThan, parser.ComparisonGreaterEqual:
		default:
			return nil, fmt.Errorf("未対応の比較演算子: %d", c.Operator)
		}
		return &comparisonCondition{columnIndex: indices[0], operator: c.Operator, value: c.Value}, nil
	case *parser.AndCondition:
		if c == nil {
			return nil, fmt.Errorf("AND 条件がありません")
		}
		left, err := prepareCondition(table, c.Left)
		if err != nil {
			return nil, err
		}
		right, err := prepareCondition(table, c.Right)
		if err != nil {
			return nil, err
		}
		return &andCondition{left: left, right: right}, nil
	case *parser.OrCondition:
		if c == nil {
			return nil, fmt.Errorf("OR 条件がありません")
		}
		left, err := prepareCondition(table, c.Left)
		if err != nil {
			return nil, err
		}
		right, err := prepareCondition(table, c.Right)
		if err != nil {
			return nil, err
		}
		return &orCondition{left: left, right: right}, nil
	default:
		return nil, fmt.Errorf("不正な WHERE 条件: %T", condition)
	}
}

func filterRows(table *Table, condition parser.Condition) ([][]value.Value, error) {
	if condition == nil {
		return table.Rows, nil
	}
	prepared, err := prepareCondition(table, condition)
	if err != nil {
		return nil, err
	}
	var matchedRows [][]value.Value
	for _, row := range table.Rows {
		if prepared.matches(row) {
			matchedRows = append(matchedRows, row)
		}
	}
	return matchedRows, nil
}
