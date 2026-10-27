package engine_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"simple-rdbms/engine"
	"simple-rdbms/lexer"
	"simple-rdbms/parser"
	"simple-rdbms/value"
)

func TestWhereSelect(t *testing.T) {
	alice := whereRow(1, "Alice")
	bob := whereRow(2, "Bob")
	alice3 := whereRow(3, "Alice")
	carol := whereRow(4, "Carol")
	tests := []struct {
		name    string
		sql     string
		columns []string
		rows    [][]value.Value
	}{
		{"整数の等価", "SELECT * FROM users WHERE id = 2;", nil, [][]value.Value{bob}},
		{"整数の不等価", "SELECT * FROM users WHERE id <> 2;", nil, [][]value.Value{alice, alice3, carol}},
		{"整数の小なり", "SELECT * FROM users WHERE id < 2;", nil, [][]value.Value{alice}},
		{"整数の以下", "SELECT * FROM users WHERE id <= 2;", nil, [][]value.Value{alice, bob}},
		{"整数の大なり", "SELECT * FROM users WHERE id > 2;", nil, [][]value.Value{alice3, carol}},
		{"整数の以上", "SELECT * FROM users WHERE id >= 2;", nil, [][]value.Value{bob, alice3, carol}},
		{"文字列の等価と複数行一致", "SELECT * FROM users WHERE name = 'Alice';", nil, [][]value.Value{alice, alice3}},
		{"文字列の不等価", "SELECT * FROM users WHERE name <> 'Bob';", nil, [][]value.Value{alice, alice3, carol}},
		{"文字列の小なり", "SELECT * FROM users WHERE name < 'Bob';", nil, [][]value.Value{alice, alice3}},
		{"文字列の以下", "SELECT * FROM users WHERE name <= 'Bob';", nil, [][]value.Value{alice, bob, alice3}},
		{"文字列の大なり", "SELECT * FROM users WHERE name > 'Bob';", nil, [][]value.Value{carol}},
		{"文字列の以上", "SELECT * FROM users WHERE name >= 'Bob';", nil, [][]value.Value{bob, carol}},
		{"一致なしでも列情報を返す", "SELECT * FROM users WHERE id = 99;", nil, nil},
		{"全行一致", "SELECT * FROM users WHERE id >= 1;", nil, [][]value.Value{alice, bob, alice3, carol}},
		{"範囲をANDで指定", "SELECT * FROM users WHERE id >= 2 AND id <= 3;", nil, [][]value.Value{bob, alice3}},
		{"OR", "SELECT * FROM users WHERE id = 1 OR id = 4;", nil, [][]value.Value{alice, carol}},
		{"ORの両辺が真でも行は増えない", "SELECT * FROM users WHERE id = 1 OR name = 'Alice';", nil, [][]value.Value{alice, alice3}},
		{"ANDがORより優先", "SELECT * FROM users WHERE id = 1 OR id = 2 AND name = 'Bob';", nil, [][]value.Value{alice, bob}},
		{"括弧で優先順位を変更", "SELECT * FROM users WHERE (id = 1 OR id = 2) AND name = 'Bob';", nil, [][]value.Value{bob}},
		{"ORの右側の括弧", "SELECT * FROM users WHERE id = 4 OR (id = 1 AND name = 'Bob');", nil, [][]value.Value{carol}},
		{"括弧の入れ子", "SELECT * FROM users WHERE id >= 2 AND (name = 'Alice' OR (name = 'Bob' AND id = 2));", nil, [][]value.Value{bob, alice3}},
		{"比較だけを括弧で囲む", "SELECT * FROM users WHERE (((id = 2)));", nil, [][]value.Value{bob}},
		{"ANDの連続", "SELECT * FROM users WHERE id >= 2 AND id <= 3 AND name = 'Alice';", nil, [][]value.Value{alice3}},
		{"ORの連続", "SELECT * FROM users WHERE id = 1 OR id = 2 OR id = 4;", nil, [][]value.Value{alice, bob, carol}},
		{"複数箇所のANDとOR", "SELECT * FROM users WHERE id = 1 AND name = 'Bob' OR id = 3 AND name = 'Alice' OR id = 4;", nil, [][]value.Value{alice3, carol}},
		{"ANDが偽", "SELECT * FROM users WHERE id = 1 AND name = 'Bob';", nil, nil},
		{"ORの両辺が偽", "SELECT * FROM users WHERE id = 99 OR name = 'Missing';", nil, nil},
		{"取得しない列で絞り込む", "SELECT name FROM users WHERE id = 2;", []string{"name"}, [][]value.Value{{whereText("Bob")}}},
		{"取得列の順序", "SELECT name, id FROM users WHERE id = 2;", []string{"name", "id"}, [][]value.Value{{whereText("Bob"), whereInt(2)}}},
		{"指定列で一致なし", "SELECT name FROM users WHERE id = 99;", []string{"name"}, nil},
		{"セミコロンなし", "SELECT * FROM users WHERE id = 2", nil, [][]value.Value{bob}},
		{"キーワードの大文字小文字と空白", "sElEcT * fRoM users wHeRe\n(id>=2 aNd name='Alice') oR id=4;", nil, [][]value.Value{alice3, carol}},
		{"WHEREなしの全列取得", "SELECT * FROM users;", nil, [][]value.Value{alice, bob, alice3, carol}},
		{"WHEREなしの指定列取得", "SELECT name FROM users;", []string{"name"}, [][]value.Value{{whereText("Alice")}, {whereText("Bob")}, {whereText("Alice")}, {whereText("Carol")}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newWhereDB(t, true)
			defer checkWhereTableUnchanged(t, db)()
			columns := tt.columns
			if columns == nil {
				columns = []string{"id", "name"}
			}
			assertWhereResult(t, executeSQL(t, db, tt.sql), columns, tt.rows)
		})
	}
}

func TestWherePreservesDuplicateRows(t *testing.T) {
	db := newWhereDB(t, true)
	executeSQL(t, db, "INSERT INTO users VALUES (2, 'Bob');")
	defer checkWhereTableUnchanged(t, db)()
	assertWhereResult(t, executeSQL(t, db, "SELECT * FROM users WHERE id = 2;"),
		[]string{"id", "name"}, [][]value.Value{whereRow(2, "Bob"), whereRow(2, "Bob")})
}

func TestWhereEmptyTable(t *testing.T) {
	db := newWhereDB(t, false)
	defer checkWhereTableUnchanged(t, db)()
	assertWhereResult(t, executeSQL(t, db, "SELECT * FROM users WHERE id = 1 OR name = 'Alice';"), []string{"id", "name"}, nil)
	assertWhereResult(t, executeSQL(t, db, "SELECT name FROM users WHERE id = 1;"), []string{"name"}, nil)
}

func TestWhereIntegerBoundaries(t *testing.T) {
	db := engine.NewDatabase()
	executeSQL(t, db, "CREATE TABLE numbers (n INT);")
	for _, n := range []int64{0, 2, 10, 9223372036854775807} {
		executeSQL(t, db, fmt.Sprintf("INSERT INTO numbers VALUES (%d);", n))
	}
	tests := []struct {
		condition string
		want      []int64
	}{
		{"n = 0", []int64{0}},
		{"n > 2", []int64{10, 9223372036854775807}},
		{"n = 9223372036854775807", []int64{9223372036854775807}},
		{"n < 9223372036854775807", []int64{0, 2, 10}},
		{"n > 9223372036854775807", nil},
	}
	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			var rows [][]value.Value
			for _, n := range tt.want {
				rows = append(rows, []value.Value{whereInt(n)})
			}
			assertWhereResult(t, executeSQL(t, db, "SELECT n FROM numbers WHERE "+tt.condition+";"), []string{"n"}, rows)
		})
	}
}

func TestWhereTextRules(t *testing.T) {
	db := engine.NewDatabase()
	executeSQL(t, db, "CREATE TABLE words (word TEXT);")
	for _, literal := range []string{"''", "'10'", "'2'", "'Alice'", "'alice'", "'é'", "'é'", "'O''Brien'"} {
		executeSQL(t, db, "INSERT INTO words VALUES ("+literal+");")
	}
	tests := []struct {
		condition string
		want      []string
	}{
		{"word = ''", []string{""}},
		{"word = 'Alice'", []string{"Alice"}},
		{"word = 'alice'", []string{"alice"}},
		{"word < '2'", []string{"", "10"}},
		{"word > 'Alice'", []string{"alice", "é", "é", "O'Brien"}},
		{"word = 'é'", []string{"é"}},
		{"word = 'é'", []string{"é"}},
		{"word > 'z'", []string{"é"}},
		{"word = 'O''Brien'", []string{"O'Brien"}},
	}
	for _, tt := range tests {
		t.Run(tt.condition, func(t *testing.T) {
			var rows [][]value.Value
			for _, word := range tt.want {
				rows = append(rows, []value.Value{whereText(word)})
			}
			assertWhereResult(t, executeSQL(t, db, "SELECT word FROM words WHERE "+tt.condition+";"), []string{"word"}, rows)
		})
	}
}

func TestWhereRejectsInvalidSyntax(t *testing.T) {
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
			sql := "SELECT * FROM users WHERE " + condition
			if _, err := parser.New(lexer.New(sql).Tokenize()).Parse(); err == nil {
				t.Fatalf("不正または未対応の構文を受け付けた: %s", sql)
			}
		})
	}
}

func TestWhereRejectsInvalidConditions(t *testing.T) {
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
				defer checkWhereTableUnchanged(t, db)()
				sql := "SELECT * FROM users WHERE " + condition + ";"
				node, err := parser.New(lexer.New(sql).Tokenize()).Parse()
				if err != nil {
					t.Fatalf("構文として正しい条件をパースできない: %v", err)
				}
				if _, err := engine.Execute(db, node); err == nil {
					t.Fatalf("不明な列または型の不一致でエラーにならなかった: %s", sql)
				}
			})
		}
	}
}

func TestWhereRejectsUnknownTable(t *testing.T) {
	db := newWhereDB(t, true)
	defer checkWhereTableUnchanged(t, db)()
	node, err := parser.New(lexer.New("SELECT * FROM missing WHERE id = 1;").Tokenize()).Parse()
	if err != nil {
		t.Fatalf("パースエラー: %v", err)
	}
	if _, err := engine.Execute(db, node); err == nil {
		t.Fatal("存在しないテーブルでエラーにならなかった")
	}
}

func newWhereDB(t *testing.T, populated bool) *engine.Database {
	t.Helper()
	db := engine.NewDatabase()
	executeSQL(t, db, "CREATE TABLE users (id INT, name TEXT);")
	if populated {
		for _, row := range []string{"(1, 'Alice')", "(2, 'Bob')", "(3, 'Alice')", "(4, 'Carol')"} {
			executeSQL(t, db, "INSERT INTO users VALUES "+row+";")
		}
	}
	return db
}

// 行順と nil / 空スライスの違いは仕様に含めず、列順と行の重複数を検証する。
func assertWhereResult(t *testing.T, got *engine.Result, columns []string, rows [][]value.Value) {
	t.Helper()
	if got == nil {
		t.Fatal("SELECT の結果が nil")
	}
	if !slices.Equal(got.Columns, columns) {
		t.Errorf("列が一致しない: got=%v, want=%v", got.Columns, columns)
	}
	counts := func(input [][]value.Value) map[string]int {
		result := make(map[string]int)
		for _, row := range input {
			result[fmt.Sprintf("%#v", row)]++
		}
		return result
	}
	if !reflect.DeepEqual(counts(got.Rows), counts(rows)) {
		t.Errorf("行が一致しない: got=%#v, want=%#v", got.Rows, rows)
	}
}

// 元データをコピーしてから、成功・失敗を問わず検索後に変更がないことを確認する。
func checkWhereTableUnchanged(t *testing.T, db *engine.Database) func() {
	t.Helper()
	table, err := db.GetTable("users")
	if err != nil {
		t.Fatal(err)
	}
	want := engine.Table{Columns: slices.Clone(table.Columns), Rows: slices.Clone(table.Rows)}
	for i := range want.Rows {
		want.Rows[i] = slices.Clone(want.Rows[i])
	}
	return func() {
		t.Helper()
		got, err := db.GetTable("users")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(*got, want) {
			t.Error("SELECT により保存済みのテーブルが変更された")
		}
	}
}

func whereRow(id int64, name string) []value.Value {
	return []value.Value{whereInt(id), whereText(name)}
}

func whereInt(n int64) value.Value { return value.Value{Type: value.TypeInt, Int: n} }

func whereText(s string) value.Value { return value.Value{Type: value.TypeText, Text: s} }
