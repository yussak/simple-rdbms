package engine_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"simple-rdbms/engine"
	"simple-rdbms/lexer"
	"simple-rdbms/parser"
	"simple-rdbms/value"
)

func TestCreateInsertSelectAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, closeDB := openPersistentDatabase(t, path)

	executeSQL(t, db, "CREATE TABLE users (id INT, name TEXT);")
	executeSQL(t, db, "INSERT INTO users VALUES (1, 'Alice');")
	want := &engine.Result{
		Columns: []string{"id", "name"},
		Rows: [][]value.Value{{
			{Type: value.TypeInt, Int: 1},
			{Type: value.TypeText, Text: "Alice"},
		}},
	}
	assertQueryResult(t, db, "SELECT * FROM users;", want)

	if err := closeDB(); err != nil {
		t.Fatalf("DBを閉じられない: %v", err)
	}

	// 元のDBを再利用せず、同じ保存先から新しく開く。
	reopened, closeReopened := openPersistentDatabase(t, path)
	defer func() {
		if err := closeReopened(); err != nil {
			t.Errorf("再オープンしたDBを閉じられない: %v", err)
		}
	}()
	assertQueryResult(t, reopened, "SELECT * FROM users;", want)
}

func TestPersistentDatabaseMultipleRowsAndInvalidInsert(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, closeDB := openPersistentDatabase(t, path)
	executeSQL(t, db, "CREATE TABLE users (id INT, name TEXT);")
	executeSQL(t, db, "INSERT INTO users VALUES (1, 'Alice');")
	bad, err := parser.New(lexer.New("INSERT INTO users VALUES ('wrong', 'Bob');").Tokenize()).Parse()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Execute(db, bad); err == nil {
		t.Fatal("型の異なる行を受け入れた")
	}
	executeSQL(t, db, "INSERT INTO users VALUES (2, 'Bob');")
	if err := closeDB(); err != nil {
		t.Fatal(err)
	}
	reopened, closeReopened := openPersistentDatabase(t, path)
	defer closeReopened()
	assertQueryResult(t, reopened, "SELECT name FROM users WHERE id = 2;", &engine.Result{
		Columns: []string{"name"},
		Rows:    [][]value.Value{{{Type: value.TypeText, Text: "Bob"}}},
	})
	assertQueryResult(t, reopened, "SELECT * FROM users;", &engine.Result{
		Columns: []string{"id", "name"},
		Rows: [][]value.Value{
			{{Type: value.TypeInt, Int: 1}, {Type: value.TypeText, Text: "Alice"}},
			{{Type: value.TypeInt, Int: 2}, {Type: value.TypeText, Text: "Bob"}},
		},
	})
}

func TestPersistentDatabaseAcrossPages(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, closeDB := openPersistentDatabase(t, path)
	executeSQL(t, db, "CREATE TABLE notes (id INT, body TEXT);")
	want := &engine.Result{Columns: []string{"id", "body"}}
	for i := 0; i < 20; i++ {
		body := strings.Repeat(string(rune('a'+i)), 1000)
		executeSQL(t, db, fmt.Sprintf("INSERT INTO notes VALUES (%d, '%s');", i, body))
		want.Rows = append(want.Rows, []value.Value{{Type: value.TypeInt, Int: int64(i)}, {Type: value.TypeText, Text: body}})
	}
	if err := closeDB(); err != nil {
		t.Fatal(err)
	}
	reopened, closeReopened := openPersistentDatabase(t, path)
	defer closeReopened()
	assertQueryResult(t, reopened, "SELECT * FROM notes;", want)
}

func TestPersistentDeleteAfterReopen(t *testing.T) {
	t.Skip("永続DBでのDELETEは未実装")
	path := filepath.Join(t.TempDir(), "test.db")
	db, closeDB := openPersistentDatabase(t, path)
	executeSQL(t, db, "CREATE TABLE notes (id INT);")
	executeSQL(t, db, "INSERT INTO notes VALUES (1);")
	executeSQL(t, db, "DELETE FROM notes WHERE id = 1;")
	if err := closeDB(); err != nil {
		t.Fatal(err)
	}
	reopened, closeReopened := openPersistentDatabase(t, path)
	defer closeReopened()
	assertQueryResult(t, reopened, "SELECT * FROM notes;", &engine.Result{Columns: []string{"id"}})
}

func openPersistentDatabase(t *testing.T, path string) (*engine.Database, func() error) {
	t.Helper()
	db, err := engine.OpenDatabase(path)
	if err != nil {
		t.Fatalf("永続DBを開けない: %v", err)
	}
	return db, db.Close
}
