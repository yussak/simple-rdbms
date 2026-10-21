package engine

import (
	"reflect"
	"simple-rdbms/value"
	"testing"
)

// 型の土台が満たすべき仕様を先に定義する。
// value.Value / TypeInt / TypeText と Column は今後実装する API を前提とする。
// 未実装のためコンパイルに失敗するが、スキップや代替実装で成功扱いにしない。
func TestCreateTablePreservesColumnDefinitions(t *testing.T) {
	db := NewDatabase()
	node, err := parse("CREATE TABLE users (name TEXT, id INT, age INT);")
	if err != nil {
		t.Fatalf("CREATE のパースに失敗: %v", err)
	}
	if _, err := Execute(db, node); err != nil {
		t.Fatalf("CREATE の実行に失敗: %v", err)
	}

	table, err := db.GetTable("users")
	if err != nil {
		t.Fatalf("作成したテーブルの取得に失敗: %v", err)
	}
	want := []Column{
		{Name: "name", Type: value.TypeText},
		{Name: "id", Type: value.TypeInt},
		{Name: "age", Type: value.TypeInt},
	}
	if !reflect.DeepEqual(table.Columns, want) {
		t.Errorf("列名・順序・型が保持されていない: got=%#v, want=%#v", table.Columns, want)
	}
	if len(table.Rows) != 0 {
		t.Errorf("作成直後の行数: got=%d, want=0", len(table.Rows))
	}
}

func TestSelectAsteriskPreservesValueTypes(t *testing.T) {
	db := setupTypedUsersDB(t, [][]value.Value{
		{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
		{{Type: value.TypeInt, Int: 2}, {Type: value.TypeText, Text: "Bob"}},
	})

	assertTypedSelect(t, db, "SELECT * FROM users;", []string{"id", "name"}, [][]value.Value{
		{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
		{{Type: value.TypeInt, Int: 2}, {Type: value.TypeText, Text: "Bob"}},
	})
}

func TestSelectColumnsPreservesValueTypesAndOrder(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		columns []string
		rows    [][]value.Value
	}{
		{
			name:    "列順を入れ替える",
			sql:     "SELECT name, id FROM users;",
			columns: []string{"name", "id"},
			rows: [][]value.Value{
				{{Type: value.TypeText, Text: "Alice"}, {Type: value.TypeInt, Int: 1}},
				{{Type: value.TypeText, Text: "Bob"}, {Type: value.TypeInt, Int: 2}},
			},
		},
		{
			name:    "一部の列を取得する",
			sql:     "SELECT name FROM users;",
			columns: []string{"name"},
			rows: [][]value.Value{
				{{Type: value.TypeText, Text: "Alice"}},
				{{Type: value.TypeText, Text: "Bob"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTypedUsersDB(t, [][]value.Value{
				{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
				{{Type: value.TypeInt, Int: 2}, {Type: value.TypeText, Text: "Bob"}},
			})
			assertTypedSelect(t, db, tt.sql, tt.columns, tt.rows)
		})
	}
}

func TestSelectDistinguishesIntegerAndText(t *testing.T) {
	db := setupTypedUsersDB(t, [][]value.Value{
		{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "1"}},
	})

	// 表示上同じ「1」でも、取得結果の整数と文字列は区別される。
	assertTypedSelect(t, db, "SELECT id, name FROM users;", []string{"id", "name"}, [][]value.Value{
		{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "1"}},
	})
}

func TestSelectPreservesZeroAndEmptyText(t *testing.T) {
	db := setupTypedUsersDB(t, [][]value.Value{
		{{Type: value.TypeInt, Int: 0}, {Type: value.TypeText, Text: ""}},
	})

	assertTypedSelect(t, db, "SELECT name, id FROM users;", []string{"name", "id"}, [][]value.Value{
		{{Type: value.TypeText, Text: ""}, {Type: value.TypeInt, Int: 0}},
	})
}

func TestSelectEmptyTypedTable(t *testing.T) {
	tests := []struct {
		sql     string
		columns []string
	}{
		{sql: "SELECT * FROM users;", columns: []string{"id", "name"}},
		{sql: "SELECT name, id FROM users;", columns: []string{"name", "id"}},
	}
	for _, tt := range tests {
		t.Run(tt.sql, func(t *testing.T) {
			db := setupTypedUsersDB(t, nil)
			node, err := parse(tt.sql)
			if err != nil {
				t.Fatalf("SELECT のパースに失敗: %v", err)
			}
			result, err := Execute(db, node)
			if err != nil {
				t.Fatalf("空テーブルの SELECT に失敗: %v", err)
			}
			if result == nil {
				t.Fatal("SELECT の結果が nil")
			}
			if !reflect.DeepEqual(result.Columns, tt.columns) {
				t.Errorf("取得列: got=%v, want=%v", result.Columns, tt.columns)
			}
			// 空の結果は nil スライスでも空スライスでもよい。
			if len(result.Rows) != 0 {
				t.Errorf("取得行数: got=%d, want=0", len(result.Rows))
			}
		})
	}
}

func setupTypedUsersDB(t *testing.T, rows [][]value.Value) *Database {
	t.Helper()
	db := NewDatabase()
	node, err := parse("CREATE TABLE users (id INT, name TEXT);")
	if err != nil {
		t.Fatalf("準備用 CREATE のパースに失敗: %v", err)
	}
	if _, err := Execute(db, node); err != nil {
		t.Fatalf("準備用 CREATE の実行に失敗: %v", err)
	}
	table, err := db.GetTable("users")
	if err != nil {
		t.Fatalf("準備用テーブルの取得に失敗: %v", err)
	}
	// INSERT への依存を避け、型の保存・取得だけを検証する。
	table.Rows = rows
	return db
}

func assertTypedSelect(t *testing.T, db *Database, sql string, columns []string, rows [][]value.Value) {
	t.Helper()
	node, err := parse(sql)
	if err != nil {
		t.Fatalf("SELECT のパースに失敗: %v", err)
	}
	result, err := Execute(db, node)
	if err != nil {
		t.Fatalf("SELECT の実行に失敗: %v", err)
	}
	if result == nil {
		t.Fatal("SELECT の結果が nil")
	}
	if !reflect.DeepEqual(result.Columns, columns) {
		t.Errorf("取得列: got=%v, want=%v", result.Columns, columns)
	}
	if !reflect.DeepEqual(result.Rows, rows) {
		t.Errorf("取得した値・型・行順: got=%#v, want=%#v", result.Rows, rows)
	}
}
