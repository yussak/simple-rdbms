package parser

import (
	"reflect"
	"testing"

	"simple-rdbms/lexer"
	"simple-rdbms/value"
)

func TestInsertTokens(t *testing.T) {
	tokens := []lexer.Token{
		{Type: lexer.TOKEN_INSERT, Literal: "INSERT"},
		{Type: lexer.TOKEN_INTO, Literal: "INTO"},
		{Type: lexer.TOKEN_IDENT, Literal: "users"},
		{Type: lexer.TOKEN_VALUES, Literal: "VALUES"},
		{Type: lexer.TOKEN_OPEN, Literal: "("},
		{Type: lexer.TOKEN_INTEGER, Literal: "1"},
		{Type: lexer.TOKEN_COMMA, Literal: ","},
		{Type: lexer.TOKEN_STRING, Literal: "Alice"},
		{Type: lexer.TOKEN_CLOSE, Literal: ")"},
		{Type: lexer.TOKEN_SEMICOLON, Literal: ";"},
		{Type: lexer.TOKEN_EOF, Literal: ""},
	}
	assertInsertStatement(t, tokens, &InsertStatement{
		Table: "users",
		Values: []value.Value{
			{Type: value.TypeInt, Int: 1},
			{Type: value.TypeText, Text: "Alice"},
		},
	})
}

func TestInsert(t *testing.T) {
	tests := []struct {
		name   string
		sql    string
		table  string
		values []value.Value
	}{
		{
			name:  "整数と文字列",
			sql:   "INSERT INTO users VALUES (1, 'Alice');",
			table: "users",
			values: []value.Value{
				{Type: value.TypeInt, Int: 1},
				{Type: value.TypeText, Text: "Alice"},
			},
		},
		{
			name:  "小文字とセミコロン省略",
			sql:   "insert into users values(1,'Alice')",
			table: "users",
			values: []value.Value{
				{Type: value.TypeInt, Int: 1},
				{Type: value.TypeText, Text: "Alice"},
			},
		},
		{
			name:  "整数と同じ表記の文字列",
			sql:   "INSERT INTO users VALUES (1, '1');",
			table: "users",
			values: []value.Value{
				{Type: value.TypeInt, Int: 1},
				{Type: value.TypeText, Text: "1"},
			},
		},
		{
			name:  "最初の値も文字列",
			sql:   "INSERT INTO users VALUES ('1', 'Alice');",
			table: "users",
			values: []value.Value{
				{Type: value.TypeText, Text: "1"},
				{Type: value.TypeText, Text: "Alice"},
			},
		},
		{
			name:  "ゼロと空文字列",
			sql:   "INSERT INTO users VALUES (0, '');",
			table: "users",
			values: []value.Value{
				{Type: value.TypeInt, Int: 0},
				{Type: value.TypeText, Text: ""},
			},
		},
		{
			name:  "引用符と記号を含む文字列",
			sql:   "INSERT INTO users VALUES ('O''Brien, ();', 'こんにちは');",
			table: "users",
			values: []value.Value{
				{Type: value.TypeText, Text: "O'Brien, ();"},
				{Type: value.TypeText, Text: "こんにちは"},
			},
		},
		{
			name:   "存在しないテーブルの単一値",
			sql:    "INSERT INTO missing VALUES (1);",
			table:  "missing",
			values: []value.Value{{Type: value.TypeInt, Int: 1}},
		},
		{
			name:   "先頭にゼロがある十進数",
			sql:    "INSERT INTO users VALUES (0012);",
			table:  "users",
			values: []value.Value{{Type: value.TypeInt, Int: 12}},
		},
		{
			name:   "int64の最大値",
			sql:    "INSERT INTO users VALUES (9223372036854775807);",
			table:  "users",
			values: []value.Value{{Type: value.TypeInt, Int: 9223372036854775807}},
		},
		{
			name:   "整数の範囲外でも文字列なら保持する",
			sql:    "INSERT INTO users VALUES ('9223372036854775808');",
			table:  "users",
			values: []value.Value{{Type: value.TypeText, Text: "9223372036854775808"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertStatement(t, lexer.New(tt.sql).Tokenize(), &InsertStatement{
				Table:  tt.table,
				Values: tt.values,
			})
		})
	}
}

func TestInsertInvalid(t *testing.T) {
	tests := []struct {
		name string
		sql  string
	}{
		{name: "INTOなし", sql: "INSERT users VALUES (1);"},
		{name: "テーブル名なし", sql: "INSERT INTO VALUES (1);"},
		{name: "VALUESなし", sql: "INSERT INTO users (1);"},
		{name: "開く括弧なし", sql: "INSERT INTO users VALUES 1);"},
		{name: "閉じる括弧なし", sql: "INSERT INTO users VALUES (1;"},
		{name: "値なし", sql: "INSERT INTO users VALUES ();"},
		{name: "先頭のカンマ", sql: "INSERT INTO users VALUES (, 1);"},
		{name: "末尾のカンマ", sql: "INSERT INTO users VALUES (1,);"},
		{name: "連続するカンマ", sql: "INSERT INTO users VALUES (1,, 'Alice');"},
		{name: "カンマなし", sql: "INSERT INTO users VALUES (1 'Alice');"},
		{name: "識別子は値にできない", sql: "INSERT INTO users VALUES (Alice);"},
		{name: "型名は値にできない", sql: "INSERT INTO users VALUES (INT);"},
		{name: "閉じていない文字列", sql: "INSERT INTO users VALUES ('Alice);"},
		{name: "整数の範囲外", sql: "INSERT INTO users VALUES (9223372036854775808);"},
		{name: "小数は未対応", sql: "INSERT INTO users VALUES (1.5);"},
		{name: "NULLは未対応", sql: "INSERT INTO users VALUES (NULL);"},
		{name: "列名指定は未対応", sql: "INSERT INTO users (id) VALUES (1);"},
		{name: "複数行は未対応", sql: "INSERT INTO users VALUES (1), (2);"},
		{name: "余分な入力", sql: "INSERT INTO users VALUES (1) extra;"},
		{name: "セミコロンの後に余分な入力", sql: "INSERT INTO users VALUES (1); extra"},
		{name: "複数のセミコロン", sql: "INSERT INTO users VALUES (1);;"},
		{name: "続けて別のSQL", sql: "INSERT INTO users VALUES (1); SELECT * FROM users;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := New(lexer.New(tt.sql).Tokenize()).Parse()
			if err == nil {
				t.Fatalf("不正なINSERTでエラーにならなかった: %s", tt.sql)
			}
			if node != nil {
				t.Fatalf("エラー時のASTがnilではない: %#v", node)
			}
		})
	}
}

func assertInsertStatement(t *testing.T, tokens []lexer.Token, want *InsertStatement) {
	t.Helper()
	node, err := New(tokens).Parse()
	if err != nil {
		t.Fatalf("INSERTのパースに失敗: %v", err)
	}
	got, ok := node.(*InsertStatement)
	if !ok {
		t.Fatalf("InsertStatementではない: %T", node)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ASTが一致しない: got=%#v, want=%#v", got, want)
	}
}
