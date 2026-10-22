package engine

import (
	"reflect"
	"simple-rdbms/lexer"
	"simple-rdbms/parser"
	"simple-rdbms/value"
	"testing"
)

func setupTestDB() *Database {
	db := NewDatabase()
	db.AddTable("users", &Table{
		Columns: []Column{
			{Name: "id", Type: value.TypeInt},
			{Name: "name", Type: value.TypeText},
		},
		Rows: [][]value.Value{
			{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
			{{Type: value.TypeInt, Int: 2}, {Type: value.TypeText, Text: "Bob"}},
		},
	})
	return db
}

func parse(sql string) (parser.Node, error) {
	tokens := lexer.New(sql).Tokenize()
	return parser.New(tokens).Parse()
}

func TestSelectAsterisk(t *testing.T) {
	db := setupTestDB()
	node, err := parse("SELECT * FROM users")
	if err != nil {
		t.Fatalf("パースエラー: %v", err)
	}

	result, err := Execute(db, node)
	if err != nil {
		t.Fatalf("実行エラー: %v", err)
	}

	if len(result.Columns) != 2 || result.Columns[0] != "id" || result.Columns[1] != "name" {
		t.Errorf("Columns: got=%v, want=[id, name]", result.Columns)
	}
	if len(result.Rows) != 2 {
		t.Fatalf("行数: got=%d, want=2", len(result.Rows))
	}
	if result.Rows[0][1] != (value.Value{Type: value.TypeText, Text: "Alice"}) ||
		result.Rows[1][1] != (value.Value{Type: value.TypeText, Text: "Bob"}) {
		t.Errorf("Rows: got=%v", result.Rows)
	}
}

func TestSelectColumns(t *testing.T) {
	db := setupTestDB()
	node, err := parse("SELECT name FROM users")
	if err != nil {
		t.Fatalf("パースエラー: %v", err)
	}

	result, err := Execute(db, node)
	if err != nil {
		t.Fatalf("実行エラー: %v", err)
	}

	if len(result.Columns) != 1 || result.Columns[0] != "name" {
		t.Errorf("Columns: got=%v, want=[name]", result.Columns)
	}
	if result.Rows[0][0] != (value.Value{Type: value.TypeText, Text: "Alice"}) ||
		result.Rows[1][0] != (value.Value{Type: value.TypeText, Text: "Bob"}) {
		t.Errorf("Rows: got=%v", result.Rows)
	}
}

func TestSelectUnknownTable(t *testing.T) {
	db := setupTestDB()
	node, err := parse("SELECT * FROM unknown")
	if err != nil {
		t.Fatalf("パースエラー: %v", err)
	}

	_, err = Execute(db, node)
	if err == nil {
		t.Fatal("存在しないテーブルでエラーにならなかった")
	}
}

func TestCreateTable(t *testing.T) {
	db := NewDatabase()
	node, err := parse("CREATE TABLE users (id INT, name TEXT);")
	if err != nil {
		t.Fatalf("パースエラー: %v", err)
	}

	_, err = Execute(db, node)
	if err != nil {
		t.Fatalf("実行エラー: %v", err)
	}

	table, err := db.GetTable("users")
	if err != nil {
		t.Fatalf("テーブルが作成されていない: %v", err)
	}
	wantColumns := []Column{
		{Name: "id", Type: value.TypeInt},
		{Name: "name", Type: value.TypeText},
	}
	if !reflect.DeepEqual(table.Columns, wantColumns) {
		t.Errorf("Columns: got=%#v, want=%#v", table.Columns, wantColumns)
	}
	if len(table.Rows) != 0 {
		t.Errorf("行数: got=%d, want=0", len(table.Rows))
	}
}

func TestCreateUnknownTypeDoesNotRegisterTable(t *testing.T) {
	db := NewDatabase()
	// パーサを通さず、不正な型を持つ AST が実行器に渡された場合を検証する。
	node := &parser.CreateStatement{
		Table: "users",
		ColumnDef: []parser.ColumnDef{
			{ColumnName: "id", TypeName: "INT"},
			{ColumnName: "enabled", TypeName: "BOOL"},
		},
	}
	if _, err := Execute(db, node); err == nil {
		t.Fatal("未対応の型でエラーにならなかった")
	}
	if _, err := db.GetTable("users"); err == nil {
		t.Fatal("失敗した CREATE のテーブルが登録されている")
	}
}

func TestSelectUnknownColumn(t *testing.T) {
	db := setupTestDB()
	node, err := parse("SELECT age FROM users")
	if err != nil {
		t.Fatalf("パースエラー: %v", err)
	}

	_, err = Execute(db, node)
	if err == nil {
		t.Fatal("存在しないカラムでエラーにならなかった")
	}
}
