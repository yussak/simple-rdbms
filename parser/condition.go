package parser

import (
	"fmt"

	"simple-rdbms/lexer"
)

// OR の各要素を AND 条件として読むことで、AND を先に結び付ける。
func (p *Parser) parseOrCondition() (Condition, error) {
	left, err := p.parseAndCondition()
	if err != nil {
		return nil, err
	}
	for p.current().Type == lexer.TOKEN_OR {
		p.advance()
		right, err := p.parseAndCondition()
		if err != nil {
			return nil, err
		}
		left = &OrCondition{Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseAndCondition() (Condition, error) {
	left, err := p.parseBasicCondition()
	if err != nil {
		return nil, err
	}
	for p.current().Type == lexer.TOKEN_AND {
		p.advance()
		right, err := p.parseBasicCondition()
		if err != nil {
			return nil, err
		}
		left = &AndCondition{Left: left, Right: right}
	}
	return left, nil
}

func (p *Parser) parseBasicCondition() (Condition, error) {
	if p.current().Type != lexer.TOKEN_OPEN {
		return p.parseComparisonCondition()
	}
	p.advance()
	// 括弧の中は OR から読み直し、括弧によるまとまりを木構造に反映する。
	condition, err := p.parseOrCondition()
	if err != nil {
		return nil, err
	}
	if err := p.expect(lexer.TOKEN_CLOSE); err != nil {
		return nil, err
	}
	return condition, nil
}

func (p *Parser) parseComparisonCondition() (*ComparisonCondition, error) {
	column, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	var operator ComparisonOperator
	switch p.current().Type {
	case lexer.TOKEN_EQUAL:
		operator = ComparisonEqual
	case lexer.TOKEN_NOT_EQUAL:
		operator = ComparisonNotEqual
	case lexer.TOKEN_LESS_THAN:
		operator = ComparisonLessThan
	case lexer.TOKEN_LESS_EQUAL:
		operator = ComparisonLessEqual
	case lexer.TOKEN_GREATER_THAN:
		operator = ComparisonGreaterThan
	case lexer.TOKEN_GREATER_EQUAL:
		operator = ComparisonGreaterEqual
	default:
		return nil, fmt.Errorf("比較演算子が必要ですが %q が来ました", p.current().Literal)
	}
	p.advance()

	v, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	return &ComparisonCondition{Column: column, Operator: operator, Value: v}, nil
}
