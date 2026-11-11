package engine_test

import (
	"path/filepath"
	"testing"

	"simple-rdbms/engine"
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

// 永続DBの開閉APIができたら、このヘルパーだけ実際のAPIに置き換える。
func openPersistentDatabase(t *testing.T, path string) (*engine.Database, func() error) {
	t.Helper()
	t.Skip("永続DBの開閉APIは未実装")
	return nil, nil
}
