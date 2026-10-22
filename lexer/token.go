package lexer

type TokenType int

const (
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF

	// SELECT 文で使うキーワード
	TOKEN_SELECT
	TOKEN_FROM

	// CREATE 文で使うキーワード
	TOKEN_CREATE
	TOKEN_TABLE

	// INSERT 文で使うキーワード
	TOKEN_INSERT
	TOKEN_INTO
	TOKEN_VALUES

	// 識別子（テーブル名、カラム名）
	TOKEN_IDENT

	// 記号
	TOKEN_COMMA     // ,
	TOKEN_ASTERISK  // *
	TOKEN_SEMICOLON // ;
	TOKEN_OPEN      // (
	TOKEN_CLOSE     // )

	// 型
	TOKEN_TEXT
	TOKEN_INT

	// 値
	TOKEN_INTEGER
	TOKEN_STRING
)

type Token struct {
	Type    TokenType
	Literal string
}
