package parser

import (
	"fmt"
	"strconv"

	"simple-rdbms/lexer"
	"simple-rdbms/value"
)

type Parser struct {
	tokens []lexer.Token
	pos    int
}

func New(tokens []lexer.Token) *Parser {
	return &Parser{tokens: tokens}
}

func (p *Parser) Parse() (Node, error) {
	tok := p.current()
	switch tok.Type {
	case lexer.TOKEN_SELECT:
		stmt, err := p.parseSelect()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case lexer.TOKEN_CREATE:
		return p.parseCreate()
	case lexer.TOKEN_INSERT:
		stmt, err := p.parseInsert()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	case lexer.TOKEN_DELETE:
		stmt, err := p.parseDelete()
		if err != nil {
			return nil, err
		}
		return stmt, nil
	default:
		return nil, fmt.Errorf("予期しないトークン: %q", tok.Literal)
	}
}

func (p *Parser) parseSelect() (*SelectStatement, error) {
	p.advance() // SELECT を消費

	columns, err := p.parseColumns()
	if err != nil {
		return nil, err
	}

	if err := p.expect(lexer.TOKEN_FROM); err != nil {
		return nil, err
	}

	table, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	var condition Condition
	if p.current().Type == lexer.TOKEN_WHERE {
		p.advance()
		condition, err = p.parseOrCondition()
		if err != nil {
			return nil, err
		}
	}

	// セミコロンがあれば消費する（なくてもよい）
	if p.current().Type == lexer.TOKEN_SEMICOLON {
		p.advance()
	}
	if err := p.expect(lexer.TOKEN_EOF); err != nil {
		return nil, err
	}

	return &SelectStatement{Columns: columns, Table: table, Where: condition}, nil
}

func (p *Parser) parseCreate() (*CreateStatement, error) {
	p.advance()

	if err := p.expect(lexer.TOKEN_TABLE); err != nil {
		return nil, err
	}

	table, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	columnDef, err := p.parseColumnsForCreate()
	if err != nil {
		return nil, err
	}

	// セミコロンがあれば消費する（なくてもよい）
	if p.current().Type == lexer.TOKEN_SEMICOLON {
		p.advance()
	}

	return &CreateStatement{ColumnDef: columnDef, Table: table}, nil
}

func (p *Parser) parseInsert() (*InsertStatement, error) {
	p.advance() // INSERT を消費
	if err := p.expect(lexer.TOKEN_INTO); err != nil {
		return nil, err
	}
	table, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TOKEN_VALUES); err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TOKEN_OPEN); err != nil {
		return nil, err
	}

	first, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	values := []value.Value{first}
	for p.current().Type == lexer.TOKEN_COMMA {
		p.advance()
		v, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	if err := p.expect(lexer.TOKEN_CLOSE); err != nil {
		return nil, err
	}

	// 単一行の INSERT として文末まで読み切る。
	if p.current().Type == lexer.TOKEN_SEMICOLON {
		p.advance()
	}
	if err := p.expect(lexer.TOKEN_EOF); err != nil {
		return nil, err
	}
	return &InsertStatement{Table: table, Values: values}, nil
}

func (p *Parser) parseDelete() (*DeleteStatement, error) {
	p.advance() // DELETE を消費
	if err := p.expect(lexer.TOKEN_FROM); err != nil {
		return nil, err
	}
	table, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	var condition Condition
	if p.current().Type == lexer.TOKEN_WHERE {
		p.advance()
		condition, err = p.parseOrCondition()
		if err != nil {
			return nil, err
		}
	}

	// 条件や文末の余分な入力を読み捨てず、文全体を検証する。
	if p.current().Type == lexer.TOKEN_SEMICOLON {
		p.advance()
	}
	if err := p.expect(lexer.TOKEN_EOF); err != nil {
		return nil, err
	}
	return &DeleteStatement{Table: table, Where: condition}, nil
}

func (p *Parser) parseValue() (value.Value, error) {
	tok := p.current()
	switch tok.Type {
	case lexer.TOKEN_INTEGER:
		n, err := strconv.ParseInt(tok.Literal, 10, 64)
		if err != nil {
			return value.Value{}, fmt.Errorf("不正な整数リテラル %q: %w", tok.Literal, err)
		}
		p.advance()
		return value.Value{Type: value.TypeInt, Int: n}, nil
	case lexer.TOKEN_STRING:
		p.advance()
		return value.Value{Type: value.TypeText, Text: tok.Literal}, nil
	default:
		return value.Value{}, fmt.Errorf("値が必要ですが %q が来ました", tok.Literal)
	}
}

// CREATE TABLE users (id INT, name TEXT);

//        CreateTableStatement
//        /                \
//     Table            Columns
//       |             /       \
//    "users"    ColumnDef    ColumnDef
//               /     \      /     \
//            "id"   "INT" "name" "TEXT"

func (p *Parser) parseColumnsForCreate() ([]ColumnDef, error) {
	if err := p.expect(lexer.TOKEN_OPEN); err != nil {
		return nil, err
	}

	var columnDef []ColumnDef

	// 1組目を読む
	columnName, err := p.expectIdent()
	if err != nil {
		return nil, fmt.Errorf("カラム名が必要です: %w", err)
	}
	if p.current().Type != lexer.TOKEN_INT && p.current().Type != lexer.TOKEN_TEXT {
		return nil, fmt.Errorf("型名が必要です: %q", p.current().Literal)
	}
	typeName := p.current().Literal
	p.advance()
	columnDef = append(columnDef, ColumnDef{ColumnName: columnName, TypeName: typeName})

	// , がある限り繰り返す
	for p.current().Type == lexer.TOKEN_COMMA {
		p.advance()
		columnName, err := p.expectIdent()
		if err != nil {
			return nil, fmt.Errorf("カラム名が必要です: %w", err)
		}
		if p.current().Type != lexer.TOKEN_INT && p.current().Type != lexer.TOKEN_TEXT {
			return nil, fmt.Errorf("型名が必要です: %q", p.current().Literal)
		}
		typeName := p.current().Literal
		p.advance()
		columnDef = append(columnDef, ColumnDef{ColumnName: columnName, TypeName: typeName})
	}

	if err := p.expect(lexer.TOKEN_CLOSE); err != nil {
		return nil, err
	}

	return columnDef, nil
}

func (p *Parser) parseColumns() ([]string, error) {
	if p.current().Type == lexer.TOKEN_ASTERISK {
		p.advance()
		return []string{"*"}, nil
	}

	var columns []string
	name, err := p.expectIdent()
	if err != nil {
		return nil, fmt.Errorf("カラム名が必要です: %w", err)
	}
	columns = append(columns, name)

	for p.current().Type == lexer.TOKEN_COMMA {
		p.advance() // , を消費
		name, err := p.expectIdent()
		if err != nil {
			return nil, fmt.Errorf("カラム名が必要です: %w", err)
		}
		columns = append(columns, name)
	}

	return columns, nil
}

func (p *Parser) current() lexer.Token {
	if p.pos >= len(p.tokens) {
		return lexer.Token{Type: lexer.TOKEN_EOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	p.pos++
}

func (p *Parser) expect(expected lexer.TokenType) error {
	if p.current().Type != expected {
		return fmt.Errorf("期待: %d, 実際: %q", expected, p.current().Literal)
	}
	p.advance()
	return nil
}

func (p *Parser) expectIdent() (string, error) {
	if p.current().Type != lexer.TOKEN_IDENT {
		return "", fmt.Errorf("識別子が必要ですが %q が来ました", p.current().Literal)
	}
	lit := p.current().Literal
	p.advance()
	return lit, nil
}
