package lexer

import "testing"

func TestDeleteKeywords(t *testing.T) {
	for _, input := range []string{"DELETE", "delete", "DeLeTe"} {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, []Token{
				{Type: TOKEN_DELETE, Literal: "DELETE"},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestDeleteKeywordIdentifiers(t *testing.T) {
	for _, input := range []string{"deleted", "delete1", "delete_id", "DELETE_log", "soft_delete", "_delete"} {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, []Token{
				{Type: TOKEN_IDENT, Literal: input},
				{Type: TOKEN_EOF, Literal: ""},
			})
		})
	}
}

func TestDeleteTokens(t *testing.T) {
	want := []Token{
		{Type: TOKEN_DELETE, Literal: "DELETE"},
		{Type: TOKEN_FROM, Literal: "FROM"},
		{Type: TOKEN_IDENT, Literal: "users"},
		{Type: TOKEN_WHERE, Literal: "WHERE"},
		{Type: TOKEN_IDENT, Literal: "id"},
		{Type: TOKEN_EQUAL, Literal: "="},
		{Type: TOKEN_INTEGER, Literal: "1"},
		{Type: TOKEN_SEMICOLON, Literal: ";"},
		{Type: TOKEN_EOF, Literal: ""},
	}
	for _, input := range []string{
		"DELETE FROM users WHERE id = 1;",
		"delete from users where id = 1;",
		"DeLeTe\nFrOm\tusers WhErE id=1;",
	} {
		t.Run(input, func(t *testing.T) {
			assertTokens(t, input, want)
		})
	}
}

func TestDeleteWithoutWhereTokens(t *testing.T) {
	assertTokens(t, "DELETE FROM users;", []Token{
		{Type: TOKEN_DELETE, Literal: "DELETE"},
		{Type: TOKEN_FROM, Literal: "FROM"},
		{Type: TOKEN_IDENT, Literal: "users"},
		{Type: TOKEN_SEMICOLON, Literal: ";"},
		{Type: TOKEN_EOF, Literal: ""},
	})
}

func TestDeleteInsideString(t *testing.T) {
	assertTokens(t, "DELETE FROM users WHERE name = 'DELETE';", []Token{
		{Type: TOKEN_DELETE, Literal: "DELETE"},
		{Type: TOKEN_FROM, Literal: "FROM"},
		{Type: TOKEN_IDENT, Literal: "users"},
		{Type: TOKEN_WHERE, Literal: "WHERE"},
		{Type: TOKEN_IDENT, Literal: "name"},
		{Type: TOKEN_EQUAL, Literal: "="},
		{Type: TOKEN_STRING, Literal: "DELETE"},
		{Type: TOKEN_SEMICOLON, Literal: ";"},
		{Type: TOKEN_EOF, Literal: ""},
	})
}
