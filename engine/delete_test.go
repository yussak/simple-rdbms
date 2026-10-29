package engine

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"simple-rdbms/parser"
	"simple-rdbms/value"
)

func TestDeleteRejectsInvalidConditionAST(t *testing.T) {
	valid := &parser.ComparisonCondition{
		Column: "id", Operator: parser.ComparisonEqual, Value: value.Value{Type: value.TypeInt, Int: 1},
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
	for _, populated := range []bool{false, true} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("行あり=%t/%s", populated, tt.name), func(t *testing.T) {
				db := setupTestDB()
				table, err := db.GetTable("users")
				if err != nil {
					t.Fatal(err)
				}
				if !populated {
					table.Rows = nil
				}
				// 行の値までコピーし、失敗前後を比較する。
				before := Table{Columns: slices.Clone(table.Columns), Rows: slices.Clone(table.Rows)}
				for i := range before.Rows {
					before.Rows[i] = slices.Clone(before.Rows[i])
				}
				result, err := Execute(db, &parser.DeleteStatement{Table: "users", Where: tt.condition})
				if err == nil {
					t.Fatal("不正な条件のASTでエラーにならなかった")
				}
				if result != nil {
					t.Errorf("エラー時に結果が返された: %#v", result)
				}
				after, err := db.GetTable("users")
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(*after, before) {
					t.Error("失敗した DELETE により既存データまたは列定義が変更された")
				}
			})
		}
	}
}
