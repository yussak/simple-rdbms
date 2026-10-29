package lexer

type TokenType int

const (
	TOKEN_ILLEGAL TokenType = iota
	TOKEN_EOF

	// SELECT 文で使うキーワード
	TOKEN_SELECT
	TOKEN_FROM
	TOKEN_WHERE
	TOKEN_AND
	TOKEN_OR

	// CREATE 文で使うキーワード
	TOKEN_CREATE
	TOKEN_TABLE

	// INSERT 文で使うキーワード
	TOKEN_INSERT
	TOKEN_INTO
	TOKEN_VALUES

	// DELETE 文で使うキーワード
	TOKEN_DELETE

	// 識別子（テーブル名、カラム名）
	TOKEN_IDENT

	// 記号
	TOKEN_COMMA     // ,
	TOKEN_ASTERISK  // *
	TOKEN_SEMICOLON // ;
	TOKEN_OPEN      // (
	TOKEN_CLOSE     // )

	// 比較演算子
	TOKEN_EQUAL         // =
	TOKEN_NOT_EQUAL     // <>
	TOKEN_LESS_THAN     // <
	TOKEN_LESS_EQUAL    // <=
	TOKEN_GREATER_THAN  // >
	TOKEN_GREATER_EQUAL // >=

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
