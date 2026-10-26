package engine_test

import (
	"reflect"
	"testing"

	"simple-rdbms/engine"
	"simple-rdbms/lexer"
	"simple-rdbms/parser"
	"simple-rdbms/value"
)

func TestInsertSelectRoundTrip(t *testing.T) {
	db := engine.NewDatabase()
	executeSQL(t, db, "CREATE TABLE users (id INT, name TEXT);")
	executeSQL(t, db, "INSERT INTO users VALUES (1, 'Alice');")
	executeSQL(t, db, "INSERT INTO users VALUES (2, 'Bob');")

	assertQueryResult(t, db, "SELECT * FROM users;", &engine.Result{
		Columns: []string{"id", "name"},
		Rows: [][]value.Value{
			{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
			{{Type: value.TypeInt, Int: 2}, {Type: value.TypeText, Text: "Bob"}},
		},
	})
	assertQueryResult(t, db, "SELECT name FROM users;", &engine.Result{
		Columns: []string{"name"},
		Rows: [][]value.Value{
			{{Type: value.TypeText, Text: "Alice"}},
			{{Type: value.TypeText, Text: "Bob"}},
		},
	})
}

func TestInsertDistinguishesIntegerAndText(t *testing.T) {
	db := engine.NewDatabase()
	executeSQL(t, db, "CREATE TABLE users (id INT, name TEXT);")
	executeSQL(t, db, "INSERT INTO users VALUES (1, '1');")

	assertQueryResult(t, db, "SELECT * FROM users;", &engine.Result{
		Columns: []string{"id", "name"},
		Rows: [][]value.Value{
			{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "1"}},
		},
	})
}

func TestInsertRejectsUnknownTable(t *testing.T) {
	db := engine.NewDatabase()
	executeSQL(t, db, "CREATE TABLE users (id INT, name TEXT);")
	executeSQL(t, db, "INSERT INTO users VALUES (1, 'Alice');")

	node, err := parser.New(lexer.New("INSERT INTO missing VALUES (2, 'Bob');").Tokenize()).Parse()
	if err != nil {
		t.Fatalf("構文として正しい INSERT をパースできない: %v", err)
	}
	if _, err := engine.Execute(db, node); err == nil {
		t.Fatal("存在しないテーブルへの INSERT でエラーにならなかった")
	}
	if _, err := db.GetTable("missing"); err == nil {
		t.Fatal("失敗した INSERT がテーブルを作成した")
	}
	assertQueryResult(t, db, "SELECT * FROM users;", &engine.Result{
		Columns: []string{"id", "name"},
		Rows: [][]value.Value{
			{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
		},
	})
}

func TestInsertRejectsInvalidRows(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{
			name: "値が少ない",
			sql:  "INSERT INTO users VALUES (2);",
		},
		{
			name: "値が多い",
			sql:  "INSERT INTO users VALUES (2, 'Bob', 'extra');",
		},
		{
			name: "INT列に文字列",
			sql:  "INSERT INTO users VALUES ('2', 'Bob');",
		},
		{
			name: "TEXT列に整数",
			sql:  "INSERT INTO users VALUES (2, 3);",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := engine.NewDatabase()
			executeSQL(t, db, "CREATE TABLE users (id INT, name TEXT);")
			executeSQL(t, db, "INSERT INTO users VALUES (1, 'Alice');")

			node, err := parser.New(lexer.New(tt.sql).Tokenize()).Parse()
			if err != nil {
				t.Fatalf("構文として正しい INSERT をパースできない: %v", err)
			}
			if _, err := engine.Execute(db, node); err == nil {
				t.Fatalf("列数・型が不正な INSERT でエラーにならなかった: %s", tt.sql)
			}
			assertQueryResult(t, db, "SELECT * FROM users;", &engine.Result{
				Columns: []string{"id", "name"},
				Rows: [][]value.Value{
					{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
				},
			})
		})
	}
}

func executeSQL(t *testing.T, db *engine.Database, sql string) *engine.Result {
	t.Helper()
	node, err := parser.New(lexer.New(sql).Tokenize()).Parse()
	if err != nil {
		t.Fatalf("パースエラー (%s): %v", sql, err)
	}
	result, err := engine.Execute(db, node)
	if err != nil {
		t.Fatalf("実行エラー (%s): %v", sql, err)
	}
	return result
}

func assertQueryResult(t *testing.T, db *engine.Database, sql string, want *engine.Result) {
	t.Helper()
	got := executeSQL(t, db, sql)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("検索結果が一致しない (%s): got=%+v, want=%+v", sql, got, want)
	}
}
