package parser

import (
	"reflect"
	"testing"

	"simple-rdbms/lexer"
	"simple-rdbms/value"
)

func TestWhereComparison(t *testing.T) {
	tests := []struct {
		operator string
		want     ComparisonOperator
	}{
		{operator: "=", want: ComparisonEqual},
		{operator: "<>", want: ComparisonNotEqual},
		{operator: "<", want: ComparisonLessThan},
		{operator: "<=", want: ComparisonLessEqual},
		{operator: ">", want: ComparisonGreaterThan},
		{operator: ">=", want: ComparisonGreaterEqual},
	}
	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			assertWhereStatement(t, "SELECT * FROM users WHERE id"+tt.operator+"2;", &SelectStatement{
				Columns: []string{"*"},
				Table:   "users",
				Where: &ComparisonCondition{
					Column:   "id",
					Operator: tt.want,
					Value:    value.Value{Type: value.TypeInt, Int: 2},
				},
			})
		})
	}
}

func TestWhereTextValue(t *testing.T) {
	assertWhereStatement(t, "SELECT name, id FROM users WHERE name = 'O''Brien';", &SelectStatement{
		Columns: []string{"name", "id"},
		Table:   "users",
		Where: &ComparisonCondition{
			Column:   "name",
			Operator: ComparisonEqual,
			Value:    value.Value{Type: value.TypeText, Text: "O'Brien"},
		},
	})
}

func TestWhereLogicalTree(t *testing.T) {
	// 異なる条件を使い、左右や括弧の位置を区別する。
	a := &ComparisonCondition{Column: "id", Operator: ComparisonEqual, Value: value.Value{Type: value.TypeInt, Int: 1}}
	b := &ComparisonCondition{Column: "id", Operator: ComparisonGreaterEqual, Value: value.Value{Type: value.TypeInt, Int: 2}}
	c := &ComparisonCondition{Column: "name", Operator: ComparisonNotEqual, Value: value.Value{Type: value.TypeText, Text: "Bob"}}
	tests := []struct {
		name      string
		condition string
		want      Condition
	}{
		{
			name:      "AND",
			condition: "id = 1 AND id >= 2",
			want:      &AndCondition{Left: a, Right: b},
		},
		{
			name:      "OR",
			condition: "id = 1 OR id >= 2",
			want:      &OrCondition{Left: a, Right: b},
		},
		{
			name:      "ORの右辺のANDが優先",
			condition: "id = 1 OR id >= 2 AND name <> 'Bob'",
			want:      &OrCondition{Left: a, Right: &AndCondition{Left: b, Right: c}},
		},
		{
			name:      "ORの左辺のANDが優先",
			condition: "id = 1 AND id >= 2 OR name <> 'Bob'",
			want:      &OrCondition{Left: &AndCondition{Left: a, Right: b}, Right: c},
		},
		{
			name:      "括弧で優先順位を変更",
			condition: "(id = 1 OR id >= 2) AND name <> 'Bob'",
			want:      &AndCondition{Left: &OrCondition{Left: a, Right: b}, Right: c},
		},
		{
			name:      "右辺の括弧",
			condition: "id = 1 AND (id >= 2 OR name <> 'Bob')",
			want:      &AndCondition{Left: a, Right: &OrCondition{Left: b, Right: c}},
		},
		{
			name:      "括弧の入れ子",
			condition: "((id = 1) AND (id >= 2 OR (name <> 'Bob' AND id = 1)))",
			want: &AndCondition{
				Left: a,
				Right: &OrCondition{
					Left:  b,
					Right: &AndCondition{Left: c, Right: a},
				},
			},
		},
		{
			name:      "単一比較の多重括弧はASTに残さない",
			condition: "(((id = 1)))",
			want:      a,
		},
		{
			name:      "ANDの連続は左結合",
			condition: "id = 1 AND id >= 2 AND name <> 'Bob'",
			want:      &AndCondition{Left: &AndCondition{Left: a, Right: b}, Right: c},
		},
		{
			name:      "ORの連続は左結合",
			condition: "id = 1 OR id >= 2 OR name <> 'Bob'",
			want:      &OrCondition{Left: &OrCondition{Left: a, Right: b}, Right: c},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertWhereStatement(t, "SELECT * FROM users WHERE "+tt.condition+";", &SelectStatement{
				Columns: []string{"*"},
				Table:   "users",
				Where:   tt.want,
			})
		})
	}
}

func TestWhereLiteralValues(t *testing.T) {
	tests := []struct {
		literal string
		want    value.Value
	}{
		{literal: "0", want: value.Value{Type: value.TypeInt, Int: 0}},
		{literal: "0012", want: value.Value{Type: value.TypeInt, Int: 12}},
		{literal: "9223372036854775807", want: value.Value{Type: value.TypeInt, Int: 9223372036854775807}},
		{literal: "''", want: value.Value{Type: value.TypeText, Text: ""}},
		{literal: "'2'", want: value.Value{Type: value.TypeText, Text: "2"}},
		{literal: "'こんにちは'", want: value.Value{Type: value.TypeText, Text: "こんにちは"}},
	}
	for _, tt := range tests {
		t.Run(tt.literal, func(t *testing.T) {
			assertWhereStatement(t, "SELECT * FROM missing WHERE unknown_column = "+tt.literal+";", &SelectStatement{
				Columns: []string{"*"},
				Table:   "missing",
				Where: &ComparisonCondition{
					Column:   "unknown_column",
					Operator: ComparisonEqual,
					Value:    tt.want,
				},
			})
		})
	}
}

func TestWhereOptionalSemicolonAndCase(t *testing.T) {
	for _, sql := range []string{
		"SELECT * FROM users WHERE id = 1;",
		"SELECT * FROM users WHERE id = 1",
		"sElEcT * fRoM users wHeRe\n(id=1);",
	} {
		t.Run(sql, func(t *testing.T) {
			assertWhereStatement(t, sql, &SelectStatement{
				Columns: []string{"*"},
				Table:   "users",
				Where: &ComparisonCondition{
					Column:   "id",
					Operator: ComparisonEqual,
					Value:    value.Value{Type: value.TypeInt, Int: 1},
				},
			})
		})
	}
}

func TestSelectWithoutWhere(t *testing.T) {
	for _, sql := range []string{"SELECT name FROM users", "SELECT name FROM users;"} {
		t.Run(sql, func(t *testing.T) {
			assertWhereStatement(t, sql, &SelectStatement{Columns: []string{"name"}, Table: "users"})
		})
	}
}

func TestWhereInvalidSyntax(t *testing.T) {
	for _, condition := range []string{
		"", "id", "id =", "= 1", "id 1", "id = 1 AND", "id = 1 OR",
		"AND id = 1", "OR id = 1", "id = 1 AND OR id = 2",
		"(id = 1", "id = 1)", "()", "(())", "(id = 1 OR)",
		"id = 1 id = 2", "id = 1; garbage", "id = 1;;",
		"id = 1; SELECT * FROM users", "id = 9223372036854775808",
		"name = 'Alice", "id = -1", "id = 1.5", "id = 1e2",
		"id != 1", "NOT id = 1", "id = NULL", "id IS NULL", "id IS NOT NULL",
		"id IN (1, 2)", "id BETWEEN 1 AND 2", "name LIKE 'A%'",
		"id = id", "1 = id", "id = 1 = 1", "id + 1 = 2",
		"name = lower('Alice')", "id = (SELECT id FROM users)",
		"id = '1'::INT", "name COLLATE \"C\" = 'Alice'",
	} {
		t.Run(condition, func(t *testing.T) {
			assertSelectParseError(t, "SELECT * FROM users WHERE "+condition)
		})
	}
}

func TestSelectTrailingInput(t *testing.T) {
	for _, sql := range []string{
		"SELECT * FROM users extra",
		"SELECT * FROM users; extra",
		"SELECT * FROM users;;",
		"SELECT * FROM users; SELECT * FROM users;",
		"SELECT * FROM users; WHERE id = 1",
	} {
		t.Run(sql, func(t *testing.T) {
			assertSelectParseError(t, sql)
		})
	}
}

func assertWhereStatement(t *testing.T, sql string, want *SelectStatement) {
	t.Helper()
	node, err := New(lexer.New(sql).Tokenize()).Parse()
	if err != nil {
		t.Fatalf("SELECTのパースに失敗: %v", err)
	}
	got, ok := node.(*SelectStatement)
	if !ok {
		t.Fatalf("SelectStatementではない: %T", node)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ASTが一致しない: got=%#v, want=%#v", got, want)
	}
}

func assertSelectParseError(t *testing.T, sql string) {
	t.Helper()
	node, err := New(lexer.New(sql).Tokenize()).Parse()
	if err == nil {
		t.Fatalf("不正なSELECTでエラーにならなかった: %s", sql)
	}
	if node != nil {
		t.Fatalf("エラー時のASTがnilではない: %#v", node)
	}
}
