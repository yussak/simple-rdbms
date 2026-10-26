package engine

import (
	"reflect"
	"strings"
	"testing"

	"simple-rdbms/parser"
	"simple-rdbms/value"
)

func TestExecuteInsertAppendsRows(t *testing.T) {
	db := NewDatabase()
	table := &Table{
		Columns: []Column{
			{Name: "id", Type: value.TypeInt},
			{Name: "name", Type: value.TypeText},
			{Name: "city", Type: value.TypeText},
		},
	}
	db.AddTable("users", table)

	want := [][]value.Value{
		{
			{Type: value.TypeInt, Int: 0},
			{Type: value.TypeText, Text: ""},
			{Type: value.TypeText, Text: "Tokyo"},
		},
		{
			{Type: value.TypeInt, Int: 2},
			{Type: value.TypeText, Text: "Osaka"},
			{Type: value.TypeText, Text: "Bob"},
		},
	}
	for i, values := range want {
		statement := &parser.InsertStatement{
			Table:  "users",
			Values: append([]value.Value(nil), values...),
		}
		result, err := Execute(db, statement)
		if err != nil {
			t.Fatalf("INSERTに失敗: %v", err)
		}
		if result != nil {
			t.Fatalf("INSERTの結果: got=%+v, want=nil", result)
		}
		// 同じ型の列でも、値を指定された位置のまま保存する。
		if !reflect.DeepEqual(table.Rows, want[:i+1]) {
			t.Fatalf("%d回目のINSERT後の行: got=%#v, want=%#v", i+1, table.Rows, want[:i+1])
		}
	}
}

func TestExecuteInsertRejectsInvalidRows(t *testing.T) {
	tests := []struct {
		name    string
		table   string
		values  []value.Value
		message string
	}{
		{
			name: "テーブル不在", table: "missing",
			values:  []value.Value{{Type: value.TypeInt, Int: 3}, {Type: value.TypeText, Text: "Carol"}},
			message: `テーブル "missing" が見つかりません`,
		},
		{
			name: "値がない", table: "users",
			message: "列数が一致しません",
		},
		{
			name: "値が少ない", table: "users",
			values:  []value.Value{{Type: value.TypeInt, Int: 3}},
			message: "列数が一致しません",
		},
		{
			name: "値が多い", table: "users",
			values:  []value.Value{{Type: value.TypeInt, Int: 3}, {Type: value.TypeText, Text: "Carol"}, {Type: value.TypeInt, Int: 4}},
			message: "列数が一致しません",
		},
		{
			name: "INT列に文字列", table: "users",
			values:  []value.Value{{Type: value.TypeText, Text: "3"}, {Type: value.TypeText, Text: "Carol"}},
			message: `カラム "id" の型が一致しません`,
		},
		{
			name: "TEXT列に整数", table: "users",
			values:  []value.Value{{Type: value.TypeInt, Int: 3}, {Type: value.TypeInt, Int: 4}},
			message: `カラム "name" の型が一致しません`,
		},
		{
			name: "型を指定していない値", table: "users",
			values:  []value.Value{{}, {Type: value.TypeText, Text: "Carol"}},
			message: `カラム "id" の型が一致しません`,
		},
		{
			name: "未対応の型の値", table: "users",
			values:  []value.Value{{Type: value.Type(255)}, {Type: value.TypeText, Text: "Carol"}},
			message: `カラム "id" の型が一致しません`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDB()
			table, err := db.GetTable("users")
			if err != nil {
				t.Fatal(err)
			}
			// 失敗前の状態を、行のスライスも含めて独立して保存する。
			want := &Table{
				Columns: append([]Column(nil), table.Columns...),
				Rows:    make([][]value.Value, len(table.Rows)),
			}
			for i, row := range table.Rows {
				want.Rows[i] = append([]value.Value(nil), row...)
			}
			result, err := Execute(db, &parser.InsertStatement{
				Table: tt.table, Values: tt.values,
			})
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("エラー: got=%v, want=%qを含むエラー", err, tt.message)
			}
			if result != nil {
				t.Fatalf("失敗時の結果がnilではない: %+v", result)
			}
			if !reflect.DeepEqual(table, want) {
				t.Fatalf("失敗したINSERTが既存データを変更した: got=%#v, want=%#v", table, want)
			}
			if len(db.tables) != 1 {
				t.Fatalf("失敗したINSERTがテーブルを追加した: %v", db.tables)
			}
		})
	}
}

func TestExecuteInsertCopiesStatementValues(t *testing.T) {
	db := NewDatabase()
	table := &Table{Columns: []Column{
		{Name: "id", Type: value.TypeInt},
		{Name: "name", Type: value.TypeText},
	}}
	db.AddTable("users", table)
	statement := &parser.InsertStatement{
		Table: "users",
		Values: []value.Value{
			{Type: value.TypeInt, Int: 1},
			{Type: value.TypeText, Text: "Alice"},
		},
	}
	if _, err := Execute(db, statement); err != nil {
		t.Fatal(err)
	}
	// 同じASTを書き換えて再利用しても、挿入済みの行は変わらない。
	statement.Values[0].Int = 2
	statement.Values[1].Text = "Bob"
	if _, err := Execute(db, statement); err != nil {
		t.Fatal(err)
	}
	statement.Values[0].Int = 3
	statement.Values[1].Text = "Carol"
	want := [][]value.Value{
		{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
		{{Type: value.TypeInt, Int: 2}, {Type: value.TypeText, Text: "Bob"}},
	}
	if !reflect.DeepEqual(table.Rows, want) {
		t.Fatalf("ASTの変更が保存済みの行に反映された: got=%#v, want=%#v", table.Rows, want)
	}
}
