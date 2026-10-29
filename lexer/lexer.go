package lexer

import (
	"strings"
	"unicode"
)

var keywords = map[string]TokenType{
	"SELECT": TOKEN_SELECT,
	"FROM":   TOKEN_FROM,
	"WHERE":  TOKEN_WHERE,
	"AND":    TOKEN_AND,
	"OR":     TOKEN_OR,
	"CREATE": TOKEN_CREATE,
	"TABLE":  TOKEN_TABLE,
	"INSERT": TOKEN_INSERT,
	"INTO":   TOKEN_INTO,
	"VALUES": TOKEN_VALUES,
	"INT":    TOKEN_INT,
	"TEXT":   TOKEN_TEXT,
}

type Lexer struct {
	input   []rune
	pos     int
	readPos int
	ch      rune
}

func New(input string) *Lexer {
	l := &Lexer{input: []rune(input)}
	l.readChar()
	return l
}

func (l *Lexer) Tokenize() []Token {
	var tokens []Token
	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TOKEN_EOF {
			break
		}
	}
	return tokens
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	switch l.ch {
	case '=':
		l.readChar()
		return Token{Type: TOKEN_EQUAL, Literal: "="}
	case '<':
		l.readChar()
		switch l.ch {
		case '=':
			l.readChar()
			return Token{Type: TOKEN_LESS_EQUAL, Literal: "<="}
		case '>':
			l.readChar()
			return Token{Type: TOKEN_NOT_EQUAL, Literal: "<>"}
		default:
			return Token{Type: TOKEN_LESS_THAN, Literal: "<"}
		}
	case '>':
		l.readChar()
		if l.ch == '=' {
			l.readChar()
			return Token{Type: TOKEN_GREATER_EQUAL, Literal: ">="}
		}
		return Token{Type: TOKEN_GREATER_THAN, Literal: ">"}
	case ',':
		tok := Token{Type: TOKEN_COMMA, Literal: ","}
		l.readChar()
		return tok
	case '*':
		tok := Token{Type: TOKEN_ASTERISK, Literal: "*"}
		l.readChar()
		return tok
	case ';':
		tok := Token{Type: TOKEN_SEMICOLON, Literal: ";"}
		l.readChar()
		return tok
	case '(':
		tok := Token{Type: TOKEN_OPEN, Literal: "("}
		l.readChar()
		return tok
	case ')':
		tok := Token{Type: TOKEN_CLOSE, Literal: ")"}
		l.readChar()
		return tok
	case '\'':
		return l.readString()
	case 0:
		return Token{Type: TOKEN_EOF, Literal: ""}
	default:
		if l.ch >= '0' && l.ch <= '9' {
			return l.readNumber()
		}
		if unicode.IsLetter(l.ch) || l.ch == '_' {
			return l.readIdentifier()
		}
		tok := Token{Type: TOKEN_ILLEGAL, Literal: string(l.ch)}
		l.readChar()
		return tok
	}
}

func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() Token {
	start := l.pos
	for unicode.IsLetter(l.ch) || unicode.IsDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	literal := string(l.input[start:l.pos])
	upper := strings.ToUpper(literal)
	if tokType, ok := keywords[upper]; ok {
		return Token{Type: tokType, Literal: upper}
	}
	return Token{Type: TOKEN_IDENT, Literal: literal}
}

func (l *Lexer) readNumber() Token {
	start := l.pos
	for l.ch >= '0' && l.ch <= '9' {
		l.readChar()
	}
	return Token{Type: TOKEN_INTEGER, Literal: string(l.input[start:l.pos])}
}

func (l *Lexer) readString() Token {
	start := l.pos
	var value strings.Builder
	l.readChar() // 開く引用符を消費する
	for {
		switch l.ch {
		case 0:
			return Token{Type: TOKEN_ILLEGAL, Literal: string(l.input[start:l.pos])}
		case '\'':
			l.readChar()
			if l.ch != '\'' {
				return Token{Type: TOKEN_STRING, Literal: value.String()}
			}
			// 連続する二つの引用符は、文字列内の一つの引用符として扱う。
			value.WriteRune('\'')
			l.readChar()
		default:
			value.WriteRune(l.ch)
			l.readChar()
		}
	}
}
