package lexer

import (
	"reflect"
	"testing"
)

func TestInsert(t *testing.T) {
	want := []Token{
		{Type: TOKEN_INSERT, Literal: "INSERT"},
		{Type: TOKEN_INTO, Literal: "INTO"},
		{Type: TOKEN_IDENT, Literal: "users"},
		{Type: TOKEN_VALUES, Literal: "VALUES"},
		{Type: TOKEN_OPEN, Literal: "("},
		{Type: TOKEN_INTEGER, Literal: "1"},
		{Type: TOKEN_COMMA, Literal: ","},
		{Type: TOKEN_STRING, Literal: "Alice"},
		{Type: TOKEN_CLOSE, Literal: ")"},
		{Type: TOKEN_SEMICOLON, Literal: ";"},
		{Type: TOKEN_EOF, Literal: ""},
	}
	for _, input := range []string{
		"INSERT INTO users VALUES (1, 'Alice');",
		"insert into users values (1, 'Alice');",
		"InSeRt InTo users VaLuEs(1,'Alice');",
	} {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, want)
		})
	}
}

func TestIntegerLiterals(t *testing.T) {
	for _, input := range []string{"0", "12345", "9223372036854775808"} {
		t.Run(input, func(t *testing.T) {
			// 整数への変換と範囲の検証はパーサーで行う。
			assertTokens(t, input, []Token{
				{Type: TOKEN_INTEGER, Literal: input},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestTypeNamesAndLiterals(t *testing.T) {
	assertTokens(t, "INT 1 TEXT '1'", []Token{
		{Type: TOKEN_INT, Literal: "INT"},
		{Type: TOKEN_INTEGER, Literal: "1"},
		{Type: TOKEN_TEXT, Literal: "TEXT"},
		{Type: TOKEN_STRING, Literal: "1"},
		{Type: TOKEN_EOF, Literal: ""},
	})
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "'Alice'", want: "Alice"},
		{input: "''", want: ""},
		{input: "' Alice, Bob; () '", want: " Alice, Bob; () "},
		{input: "'こんにちは'", want: "こんにちは"},
		{input: "'O''Brien'", want: "O'Brien"},
		{input: "''''", want: "'"},
		{input: "'INSERT INTO users VALUES'", want: "INSERT INTO users VALUES"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assertTokens(t, tt.input, []Token{
				{Type: TOKEN_STRING, Literal: tt.want},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestUnterminatedString(t *testing.T) {
	for _, input := range []string{"'", "'Alice", "'O''Brien"} {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, []Token{
				{Type: TOKEN_ILLEGAL, Literal: input},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestNegativeIntegerNotImplemented(t *testing.T) {
	t.Skip("未対応: 負の整数リテラル")
	assertTokens(t, "-1", []Token{
		{Type: TOKEN_INTEGER, Literal: "-1"},
		{Type: TOKEN_EOF, Literal: ""},
	})
}

func assertTokens(t *testing.T, input string, want []Token) {
	t.Helper()
	got := New(input).Tokenize()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("トークンが一致しない (%q): got=%+v, want=%+v", input, got, want)
	}
}
