package engine_test

import (
	"fmt"
	"reflect"
	"testing"

	"simple-rdbms/engine"
	"simple-rdbms/lexer"
	"simple-rdbms/parser"
	"simple-rdbms/value"
)

func TestDeleteSelectRoundTrip(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	alice := whereRow(1, "Alice")
	bob := whereRow(2, "Bob")
	alice3 := whereRow(3, "Alice")
	carol := whereRow(4, "Carol")
	tests := []struct {
		name string
		sql  string
		rows [][]value.Value
	}{
		{"整数で一行削除", "DELETE FROM users WHERE id = 2;", [][]value.Value{alice, alice3, carol}},
		{"文字列で複数行削除", "DELETE FROM users WHERE name = 'Alice';", [][]value.Value{bob, carol}},
		{"ANDで範囲指定", "DELETE FROM users WHERE id >= 2 AND id <= 3;", [][]value.Value{alice, carol}},
		{"OR", "DELETE FROM users WHERE id = 1 OR id = 4;", [][]value.Value{bob, alice3}},
		{"ANDがORより優先", "DELETE FROM users WHERE id = 1 OR id = 2 AND name = 'Bob';", [][]value.Value{alice3, carol}},
		{"括弧で優先順位を変更", "DELETE FROM users WHERE (id = 1 OR id = 2) AND name = 'Bob';", [][]value.Value{alice, alice3, carol}},
		{"括弧の入れ子", "DELETE FROM users WHERE id >= 2 AND (name = 'Alice' OR (name = 'Bob' AND id = 2));", [][]value.Value{alice, carol}},
		{"一致なし", "DELETE FROM users WHERE id = 99;", [][]value.Value{alice, bob, alice3, carol}},
		{"全行一致", "DELETE FROM users WHERE id >= 1;", nil},
		{"WHEREなし", "DELETE FROM users;", nil},
		{"セミコロンなし", "DELETE FROM users WHERE id = 2", [][]value.Value{alice, alice3, carol}},
		{"WHEREもセミコロンもなし", "DELETE FROM users", nil},
		{"キーワードの大文字小文字と空白", "dElEtE\nFrOm users WhErE id=2;", [][]value.Value{alice, alice3, carol}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newWhereDB(t, true)
			before := snapshotTable(t, db)
			if result := executeSQL(t, db, tt.sql); result != nil {
				t.Fatalf("DELETE の結果は nil を期待した: %+v", result)
			}
			assertWhereResult(t, executeSQL(t, db, "SELECT * FROM users;"), []string{"id", "name"}, tt.rows)
			table, err := db.GetTable("users")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(table.Columns, before.Columns) {
				t.Fatal("DELETE により列定義が変更された")
			}
		})
	}
}

func TestDeleteRemovesAllMatchingDuplicates(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	db := newWhereDB(t, true)
	executeSQL(t, db, "INSERT INTO users VALUES (2, 'Bob');")
	executeSQL(t, db, "INSERT INTO users VALUES (4, 'Carol');")
	executeSQL(t, db, "DELETE FROM users WHERE id = 2;")
	assertWhereResult(t, executeSQL(t, db, "SELECT * FROM users;"), []string{"id", "name"},
		[][]value.Value{whereRow(1, "Alice"), whereRow(3, "Alice"), whereRow(4, "Carol"), whereRow(4, "Carol")})
}

func TestDeleteEmptyTable(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	for _, sql := range []string{"DELETE FROM users;", "DELETE FROM users WHERE id = 1 OR name = 'Alice';"} {
		t.Run(sql, func(t *testing.T) {
			db := newWhereDB(t, false)
			executeSQL(t, db, sql)
			assertWhereResult(t, executeSQL(t, db, "SELECT * FROM users;"), []string{"id", "name"}, nil)
		})
	}
}

func TestDeleteThenInsert(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	for _, sql := range []string{"DELETE FROM users WHERE id = 2;", "DELETE FROM users WHERE id >= 1;", "DELETE FROM users;"} {
		t.Run(sql, func(t *testing.T) {
			db := newWhereDB(t, true)
			executeSQL(t, db, "CREATE TABLE other (id INT, name TEXT);")
			executeSQL(t, db, "INSERT INTO other VALUES (2, 'Other');")
			executeSQL(t, db, sql)
			executeSQL(t, db, "INSERT INTO users VALUES (2, 'New Bob');")
			want := [][]value.Value{whereRow(2, "New Bob")}
			if sql == "DELETE FROM users WHERE id = 2;" {
				want = append(want, whereRow(1, "Alice"), whereRow(3, "Alice"), whereRow(4, "Carol"))
			}
			assertWhereResult(t, executeSQL(t, db, "SELECT * FROM users;"), []string{"id", "name"}, want)
			assertWhereResult(t, executeSQL(t, db, "SELECT name FROM users WHERE id = 2;"),
				[]string{"name"}, [][]value.Value{{whereText("New Bob")}})
			assertWhereResult(t, executeSQL(t, db, "SELECT * FROM other;"),
				[]string{"id", "name"}, [][]value.Value{whereRow(2, "Other")})
		})
	}
}

func TestDeleteRejectsInvalidConditionsWithoutChanges(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	for _, populated := range []bool{false, true} {
		for _, condition := range []string{
			"missing = 1", "id = '2'", "name = 2",
			"id >= 1 OR missing = 1", "missing = 1 OR id >= 1",
			"id = 99 AND missing = 1", "missing = 1 AND id = 99",
			"id >= 1 OR name = 2", "name = 2 OR id >= 1",
			"id = 99 AND id = '2'", "id = '2' AND id = 99",
			"id >= 1 OR (id = 99 AND missing = 1)",
		} {
			t.Run(fmt.Sprintf("行あり=%t/%s", populated, condition), func(t *testing.T) {
				db := newWhereDB(t, populated)
				before := snapshotTable(t, db)
				defer assertDeleteTableUnchanged(t, db, before)
				sql := "DELETE FROM users WHERE " + condition + ";"
				node, err := parser.New(lexer.New(sql).Tokenize()).Parse()
				if err != nil {
					t.Fatalf("構文として正しい DELETE をパースできない: %v", err)
				}
				if _, err := engine.Execute(db, node); err == nil {
					t.Fatalf("不明な列または型の不一致でエラーにならなかった: %s", sql)
				}
			})
		}
	}
}

func TestDeleteRejectsUnknownTableWithoutChanges(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	for _, sql := range []string{"DELETE FROM missing;", "DELETE FROM missing WHERE id = 1;"} {
		t.Run(sql, func(t *testing.T) {
			db := newWhereDB(t, true)
			before := snapshotTable(t, db)
			defer assertDeleteTableUnchanged(t, db, before)
			node, err := parser.New(lexer.New(sql).Tokenize()).Parse()
			if err != nil {
				t.Fatalf("構文として正しい DELETE をパースできない: %v", err)
			}
			if _, err := engine.Execute(db, node); err == nil {
				t.Fatal("存在しないテーブルでエラーにならなかった")
			}
			if _, err := db.GetTable("missing"); err == nil {
				t.Fatal("失敗した DELETE がテーブルを作成した")
			}
		})
	}
}

func TestDeleteRejectsInvalidSyntaxWithoutChanges(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	for _, sql := range []string{
		"DELETE users;", "DELETE FROM;", "DELETE FROM users WHERE;",
		"DELETE FROM users WHERE id =;", "DELETE FROM users WHERE id = 1 OR;",
		"DELETE FROM users WHERE (id = 1;", "DELETE FROM users WHERE ();",
		"DELETE FROM users garbage;", "DELETE FROM users;;",
		"DELETE FROM users; SELECT * FROM users;",
		"DELETE FROM users WHERE id = 1; garbage",
		"DELETE FROM users WHERE id = 9223372036854775808;",
		"DELETE FROM users WHERE name = 'Alice",
		"DELETE FROM users LIMIT 1;", "DELETE FROM users WHERE id = 1 RETURNING *;",
	} {
		t.Run(sql, func(t *testing.T) {
			db := newWhereDB(t, true)
			before := snapshotTable(t, db)
			defer assertDeleteTableUnchanged(t, db, before)
			node, err := parser.New(lexer.New(sql).Tokenize()).Parse()
			if err == nil {
				// 誤って解析に成功した場合も実行し、部分的に削除されないか確認する。
				_, executeErr := engine.Execute(db, node)
				t.Fatalf("不正または未対応の構文を受け付けた: %s (実行エラー=%v)", sql, executeErr)
			}
		})
	}
}

func TestDeletePreservesPreviousSelectResults(t *testing.T) {
	t.Skip("未実装: DELETE の字句解析・構文解析・実行")

	db := newWhereDB(t, true)
	before := snapshotTable(t, db)
	all := executeSQL(t, db, "SELECT * FROM users;")
	filtered := executeSQL(t, db, "SELECT * FROM users WHERE id >= 1;")
	executeSQL(t, db, "DELETE FROM users WHERE id <= 2;")
	assertWhereResult(t, executeSQL(t, db, "SELECT * FROM users;"), []string{"id", "name"},
		[][]value.Value{whereRow(3, "Alice"), whereRow(4, "Carol")})
	assertWhereResult(t, all, []string{"id", "name"}, before.Rows)
	assertWhereResult(t, filtered, []string{"id", "name"}, before.Rows)
}

func assertDeleteTableUnchanged(t *testing.T, db *engine.Database, before engine.Table) {
	t.Helper()
	table, err := db.GetTable("users")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*table, before) {
		t.Error("失敗した DELETE により既存データまたは列定義が変更された")
	}
}
