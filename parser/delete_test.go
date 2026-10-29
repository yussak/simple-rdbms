package parser

import (
	"reflect"
	"testing"

	"simple-rdbms/lexer"
	"simple-rdbms/value"
)

func TestDeleteWithoutWhere(t *testing.T) {
	for _, sql := range []string{
		"DELETE FROM users;", "DELETE FROM users",
		"delete from users;", "DeLeTe\nFrOm\tusers",
	} {
		t.Run(sql, func(t *testing.T) {
			assertDeleteStatement(t, sql, &DeleteStatement{Table: "users"})
		})
	}
}

func TestDeleteWhereComparisons(t *testing.T) {
	tests := []struct {
		operator string
		want     ComparisonOperator
	}{
		{"=", ComparisonEqual}, {"<>", ComparisonNotEqual},
		{"<", ComparisonLessThan}, {"<=", ComparisonLessEqual},
		{">", ComparisonGreaterThan}, {">=", ComparisonGreaterEqual},
	}
	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			assertDeleteStatement(t, "DELETE FROM users WHERE id"+tt.operator+"2;", &DeleteStatement{
				Table: "users",
				Where: &ComparisonCondition{
					Column: "id", Operator: tt.want, Value: value.Value{Type: value.TypeInt, Int: 2},
				},
			})
		})
	}
}

func TestDeleteWhereValues(t *testing.T) {
	tests := []struct {
		literal string
		want    value.Value
	}{
		{"0", value.Value{Type: value.TypeInt, Int: 0}},
		{"0012", value.Value{Type: value.TypeInt, Int: 12}},
		{"9223372036854775807", value.Value{Type: value.TypeInt, Int: 9223372036854775807}},
		{"''", value.Value{Type: value.TypeText, Text: ""}},
		{"'2'", value.Value{Type: value.TypeText, Text: "2"}},
		{"'O''Brien'", value.Value{Type: value.TypeText, Text: "O'Brien"}},
	}
	for _, tt := range tests {
		t.Run(tt.literal, func(t *testing.T) {
			// テーブル・列の存在と型の照合は実行器で行う。
			assertDeleteStatement(t, "DELETE FROM missing WHERE unknown_column = "+tt.literal+";", &DeleteStatement{
				Table: "missing",
				Where: &ComparisonCondition{Column: "unknown_column", Operator: ComparisonEqual, Value: tt.want},
			})
		})
	}
}

func TestDeleteWhereLogicalTree(t *testing.T) {
	a := &ComparisonCondition{Column: "id", Operator: ComparisonEqual, Value: value.Value{Type: value.TypeInt, Int: 1}}
	b := &ComparisonCondition{Column: "id", Operator: ComparisonGreaterEqual, Value: value.Value{Type: value.TypeInt, Int: 2}}
	c := &ComparisonCondition{Column: "name", Operator: ComparisonNotEqual, Value: value.Value{Type: value.TypeText, Text: "Bob"}}
	tests := []struct {
		condition string
		want      Condition
	}{
		{"id = 1 AND id >= 2", &AndCondition{Left: a, Right: b}},
		{"id = 1 OR id >= 2", &OrCondition{Left: a, Right: b}},
		{"id = 1 OR id >= 2 AND name <> 'Bob'", &OrCondition{Left: a, Right: &AndCondition{Left: b, Right: c}}},
		{"(id = 1 OR id >= 2) AND name <> 'Bob'", &AndCondition{Left: &OrCondition{Left: a, Right: b}, Right: c}},
		{"id = 1 AND (id >= 2 OR (name <> 'Bob' AND id = 1))", &AndCondition{
			Left: a, Right: &OrCondition{Left: b, Right: &AndCondition{Left: c, Right: a}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			assertDeleteStatement(t, "DELETE FROM users WHERE "+tt.condition+";", &DeleteStatement{
				Table: "users", Where: tt.want,
			})
		})
	}
}

func TestDeleteWhereOptionalSemicolonAndCase(t *testing.T) {
	for _, sql := range []string{
		"DELETE FROM users WHERE id = 1;",
		"DELETE FROM users WHERE id = 1",
		"dElEtE fRoM users wHeRe\n(id=1);",
	} {
		t.Run(sql, func(t *testing.T) {
			assertDeleteStatement(t, sql, &DeleteStatement{
				Table: "users",
				Where: &ComparisonCondition{
					Column: "id", Operator: ComparisonEqual, Value: value.Value{Type: value.TypeInt, Int: 1},
				},
			})
		})
	}
}

func TestDeleteInvalidSyntax(t *testing.T) {
	for _, sql := range []string{
		"DELETE", "DELETE users;", "DELETE * FROM users;",
		"DELETE FROM", "DELETE FROM;", "DELETE FROM WHERE id = 1;",
		"DELETE FROM 123;", "DELETE FROM users WHERE;", "DELETE FROM users WHERE id =;",
		"DELETE FROM users WHERE id = 1 AND;", "DELETE FROM users WHERE id = 1 OR;",
		"DELETE FROM users WHERE (id = 1;", "DELETE FROM users WHERE id = 1);",
		"DELETE FROM users WHERE ();", "DELETE FROM users WHERE name = 'Alice",
		"DELETE FROM users WHERE id = 9223372036854775808;",
		"DELETE FROM users WHERE id = -1;", "DELETE FROM users WHERE id != 1;",
		"DELETE FROM users WHERE id = id;", "DELETE FROM users WHERE id = NULL;",
		"DELETE FROM users extra", "DELETE FROM users; extra",
		"DELETE FROM users;;", "DELETE FROM users; DELETE FROM users;",
		"DELETE FROM users; WHERE id = 1;",
		"DELETE FROM users WHERE id = 1 garbage",
		"DELETE FROM users WHERE id = 1; garbage",
		"DELETE FROM users WHERE id = 1;;",
		"DELETE FROM users WHERE id = 1; DELETE FROM users;",
		"DELETE FROM users LIMIT 1;", "DELETE FROM users ORDER BY id;",
		"DELETE FROM users USING other;", "DELETE FROM users RETURNING *;",
	} {
		t.Run(sql, func(t *testing.T) {
			node, err := New(lexer.New(sql).Tokenize()).Parse()
			if err == nil {
				t.Fatalf("不正なDELETEでエラーにならなかった: %s", sql)
			}
			if node != nil {
				t.Fatalf("エラー時のASTがnilではない: %#v", node)
			}
		})
	}
}

func assertDeleteStatement(t *testing.T, sql string, want *DeleteStatement) {
	t.Helper()
	node, err := New(lexer.New(sql).Tokenize()).Parse()
	if err != nil {
		t.Fatalf("DELETEのパースに失敗: %v", err)
	}
	got, ok := node.(*DeleteStatement)
	if !ok {
		t.Fatalf("DeleteStatementではない: %T", node)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ASTが一致しない: got=%#v, want=%#v", got, want)
	}
}
