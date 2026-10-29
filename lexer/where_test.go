package lexer

import "testing"

func TestWhereKeywords(t *testing.T) {
	for _, input := range []string{
		"WHERE AND OR",
		"where and or",
		"WhErE aNd oR",
	} {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, []Token{
				{Type: TOKEN_WHERE, Literal: "WHERE"},
				{Type: TOKEN_AND, Literal: "AND"},
				{Type: TOKEN_OR, Literal: "OR"},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestWhereComparisonOperators(t *testing.T) {
	tests := []struct {
		input string
		want  TokenType
	}{
		{input: "=", want: TOKEN_EQUAL},
		{input: "<>", want: TOKEN_NOT_EQUAL},
		{input: "<", want: TOKEN_LESS_THAN},
		{input: "<=", want: TOKEN_LESS_EQUAL},
		{input: ">", want: TOKEN_GREATER_THAN},
		{input: ">=", want: TOKEN_GREATER_EQUAL},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assertTokens(t, tt.input, []Token{
				{Type: tt.want, Literal: tt.input},
				{Type: TOKEN_EOF, Literal: ""},
			})
			assertTokens(t, "id"+tt.input+"2", []Token{
				{Type: TOKEN_IDENT, Literal: "id"},
				{Type: tt.want, Literal: tt.input},
				{Type: TOKEN_INTEGER, Literal: "2"},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestWhereSelectTokens(t *testing.T) {
	assertTokens(t, "SELECT name FROM users\nWHERE(id>=2 AND name<>'Alice')OR id<=1;", []Token{
		{Type: TOKEN_SELECT, Literal: "SELECT"},
		{Type: TOKEN_IDENT, Literal: "name"},
		{Type: TOKEN_FROM, Literal: "FROM"},
		{Type: TOKEN_IDENT, Literal: "users"},
		{Type: TOKEN_WHERE, Literal: "WHERE"},
		{Type: TOKEN_OPEN, Literal: "("},
		{Type: TOKEN_IDENT, Literal: "id"},
		{Type: TOKEN_GREATER_EQUAL, Literal: ">="},
		{Type: TOKEN_INTEGER, Literal: "2"},
		{Type: TOKEN_AND, Literal: "AND"},
		{Type: TOKEN_IDENT, Literal: "name"},
		{Type: TOKEN_NOT_EQUAL, Literal: "<>"},
		{Type: TOKEN_STRING, Literal: "Alice"},
		{Type: TOKEN_CLOSE, Literal: ")"},
		{Type: TOKEN_OR, Literal: "OR"},
		{Type: TOKEN_IDENT, Literal: "id"},
		{Type: TOKEN_LESS_EQUAL, Literal: "<="},
		{Type: TOKEN_INTEGER, Literal: "1"},
		{Type: TOKEN_SEMICOLON, Literal: ";"},
		{Type: TOKEN_EOF, Literal: ""},
	})
}

func TestWhereKeywordIdentifiers(t *testing.T) {
	for _, input := range []string{"where_id", "and2", "OR_name", "somewhere", "android", "origin"} {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, []Token{
				{Type: TOKEN_IDENT, Literal: input},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestWhereSymbolsInsideString(t *testing.T) {
	assertTokens(t, "'WHERE AND OR = <> < <= > >='", []Token{
		{Type: TOKEN_STRING, Literal: "WHERE AND OR = <> < <= > >="},
		{Type: TOKEN_EOF, Literal: ""},
	})
}

func TestWhereUnsupportedBang(t *testing.T) {
	assertTokens(t, "!", []Token{
		{Type: TOKEN_ILLEGAL, Literal: "!"},
		{Type: TOKEN_EOF, Literal: ""},
	})
	assertTokens(t, "!=", []Token{
		{Type: TOKEN_ILLEGAL, Literal: "!"},
		{Type: TOKEN_EQUAL, Literal: "="},
		{Type: TOKEN_EOF, Literal: ""},
	})
}
