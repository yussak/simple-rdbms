package engine

import (
	"reflect"
	"testing"

	"simple-rdbms/parser"
	"simple-rdbms/value"
)

func TestSelectRejectsInvalidConditionAST(t *testing.T) {
	valid := &parser.ComparisonCondition{
		Column:   "id",
		Operator: parser.ComparisonEqual,
		Value:    value.Value{Type: value.TypeInt, Int: 1},
	}
	tests := []struct {
		name      string
		condition parser.Condition
	}{
		{
			name: "未指定の演算子",
			condition: &parser.ComparisonCondition{
				Column: "id", Operator: parser.ComparisonInvalid, Value: value.Value{Type: value.TypeInt, Int: 1},
			},
		},
		{
			name: "未対応の演算子",
			condition: &parser.ComparisonCondition{
				Column: "id", Operator: parser.ComparisonOperator(255), Value: value.Value{Type: value.TypeInt, Int: 1},
			},
		},
		{
			name: "無効な値の型",
			condition: &parser.ComparisonCondition{
				Column: "id", Operator: parser.ComparisonEqual, Value: value.Value{},
			},
		},
		{name: "nilの比較条件", condition: (*parser.ComparisonCondition)(nil)},
		{name: "nilのAND条件", condition: (*parser.AndCondition)(nil)},
		{name: "nilのOR条件", condition: (*parser.OrCondition)(nil)},
		{name: "ANDの左辺なし", condition: &parser.AndCondition{Right: valid}},
		{name: "ANDの右辺なし", condition: &parser.AndCondition{Left: valid}},
		{name: "ORの左辺なし", condition: &parser.OrCondition{Right: valid}},
		{name: "ORの右辺なし", condition: &parser.OrCondition{Left: valid}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := NewDatabase()
			db.AddTable("users", &Table{Columns: []Column{{Name: "id", Type: value.TypeInt}}})
			// SQL を経由しない AST でも、行の有無によらず条件を検証する。
			result, err := Execute(db, &parser.SelectStatement{
				Columns: []string{"*"}, Table: "users", Where: tt.condition,
			})
			if err == nil {
				t.Fatal("不正な条件のASTでエラーにならなかった")
			}
			if result != nil {
				t.Fatalf("エラー時に結果が返された: %#v", result)
			}
		})
	}
}

func TestSelectConditionInt64Ordering(t *testing.T) {
	const minInt64 = -9223372036854775808
	const maxInt64 = 9223372036854775807
	db := NewDatabase()
	db.AddTable("numbers", &Table{
		Columns: []Column{{Name: "n", Type: value.TypeInt}},
		Rows: [][]value.Value{
			{{Type: value.TypeInt, Int: minInt64}},
			{{Type: value.TypeInt, Int: maxInt64}},
		},
	})
	tests := []struct {
		name     string
		operator parser.ComparisonOperator
		right    int64
		want     int64
	}{
		{name: "最小値は最大値より小さい", operator: parser.ComparisonLessThan, right: maxInt64, want: minInt64},
		{name: "最大値は最小値より大きい", operator: parser.ComparisonGreaterThan, right: minInt64, want: maxInt64},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Execute(db, &parser.SelectStatement{
				Columns: []string{"n"},
				Table:   "numbers",
				Where: &parser.ComparisonCondition{
					Column: "n", Operator: tt.operator, Value: value.Value{Type: value.TypeInt, Int: tt.right},
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			want := &Result{Columns: []string{"n"}, Rows: [][]value.Value{{{Type: value.TypeInt, Int: tt.want}}}}
			if !reflect.DeepEqual(result, want) {
				t.Fatalf("整数の比較結果が一致しない: got=%#v, want=%#v", result, want)
			}
		})
	}
}
