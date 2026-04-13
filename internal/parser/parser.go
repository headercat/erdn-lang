package parser

import (
	"fmt"
	"strings"

	"github.com/headercat/erdn-lang/internal/ast"
	"github.com/headercat/erdn-lang/internal/lexer"
)

// parser is a recursive descent parser for erdn-lang.
type parser struct {
	tokens []lexer.Token
	pos    int
}

// Parse converts a token slice into an AST Program.
func Parse(tokens []lexer.Token) (*ast.Program, error) {
	p := &parser{tokens: tokens}
	return p.parseProgram()
}

// peek returns the next non-newline, non-line-comment token without consuming it.
func (p *parser) peek() lexer.Token {
	i := p.pos
	for i < len(p.tokens) {
		tok := p.tokens[i]
		if tok.Type == lexer.TOKEN_NEWLINE || tok.Type == lexer.TOKEN_LINE_COMMENT {
			i++
			continue
		}
		return tok
	}
	return lexer.Token{Type: lexer.TOKEN_EOF}
}

// peekRaw returns the next token without skipping newlines/line-comments.
func (p *parser) peekRaw() lexer.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lexer.Token{Type: lexer.TOKEN_EOF}
}

// advance consumes and returns the next non-newline, non-line-comment token.
func (p *parser) advance() lexer.Token {
	for p.pos < len(p.tokens) {
		t := p.tokens[p.pos]
		p.pos++
		if t.Type == lexer.TOKEN_NEWLINE || t.Type == lexer.TOKEN_LINE_COMMENT {
			continue
		}
		return t
	}
	return lexer.Token{Type: lexer.TOKEN_EOF}
}

func (p *parser) expect(tt lexer.TokenType) (lexer.Token, error) {
	tok := p.advance()
	if tok.Type != tt {
		return tok, fmt.Errorf("line %d col %d: expected token type %d, got %q (%d)",
			tok.Line, tok.Col, tt, tok.Value, tok.Type)
	}
	return tok, nil
}

// collectComments collects any pending # comments before the next non-comment token.
func (p *parser) collectComments() []string {
	var comments []string
	for p.pos < len(p.tokens) {
		tok := p.tokens[p.pos]
		switch tok.Type {
		case lexer.TOKEN_NEWLINE, lexer.TOKEN_LINE_COMMENT:
			p.pos++
		case lexer.TOKEN_HASH_COMMENT:
			comments = append(comments, tok.Value)
			p.pos++
		default:
			return comments
		}
	}
	return comments
}

func (p *parser) parseProgram() (*ast.Program, error) {
	prog := &ast.Program{}
	for {
		comments := p.collectComments()
		tok := p.peek()
		if tok.Type == lexer.TOKEN_EOF {
			break
		}
		switch tok.Type {
		case lexer.TOKEN_KEYWORD_TABLE:
			tbl, err := p.parseTable(comments)
			if err != nil {
				return nil, err
			}
			prog.Tables = append(prog.Tables, tbl)
		case lexer.TOKEN_KEYWORD_LINK:
			lnk, err := p.parseLink(comments)
			if err != nil {
				return nil, err
			}
			prog.Links = append(prog.Links, lnk)
		default:
			tok = p.advance()
			return nil, fmt.Errorf("line %d col %d: unexpected token %q", tok.Line, tok.Col, tok.Value)
		}
	}
	return prog, nil
}

func (p *parser) parseTable(comments []string) (*ast.Table, error) {
	tok, err := p.expect(lexer.TOKEN_KEYWORD_TABLE)
	if err != nil {
		return nil, err
	}
	tbl := &ast.Table{Comments: comments, Line: tok.Line}

	nameTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	tbl.Name = nameTok.Value

	// expect ( or {
	open := p.advance()
	var closeType lexer.TokenType
	switch open.Type {
	case lexer.TOKEN_LPAREN:
		closeType = lexer.TOKEN_RPAREN
	case lexer.TOKEN_LBRACE:
		closeType = lexer.TOKEN_RBRACE
	default:
		return nil, fmt.Errorf("line %d col %d: expected '(' or '{' after table name, got %q", open.Line, open.Col, open.Value)
	}

	for {
		colComments := p.collectComments()
		next := p.peek()
		if next.Type == closeType || next.Type == lexer.TOKEN_EOF {
			p.advance()
			break
		}
		col, err := p.parseColumn(colComments)
		if err != nil {
			return nil, err
		}
		tbl.Columns = append(tbl.Columns, col)
	}
	return tbl, nil
}

func (p *parser) parseColumn(comments []string) (*ast.Column, error) {
	nameTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	col := &ast.Column{Name: nameTok.Value, Comments: comments, Line: nameTok.Line}

	// type name
	typeTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return nil, err
	}
	col.Type = typeTok.Value

	// optional type params: (255) or (10,2)
	if p.peek().Type == lexer.TOKEN_LPAREN {
		p.advance() // consume (
		for {
			param := p.advance()
			if param.Type == lexer.TOKEN_NUMBER || param.Type == lexer.TOKEN_IDENT || param.Type == lexer.TOKEN_STRING {
				col.TypeParams = append(col.TypeParams, param.Value)
			} else if param.Type == lexer.TOKEN_RPAREN {
				break
			} else if param.Type == lexer.TOKEN_COMMA {
				continue
			} else {
				return nil, fmt.Errorf("line %d col %d: unexpected token in type params: %q", param.Line, param.Col, param.Value)
			}
			next := p.peek()
			if next.Type == lexer.TOKEN_RPAREN {
				p.advance()
				break
			} else if next.Type == lexer.TOKEN_COMMA {
				p.advance()
			}
		}
	}

	// modifiers until newline or close paren/brace
	for {
		next := p.peekRaw()
		if next.Type == lexer.TOKEN_NEWLINE || next.Type == lexer.TOKEN_EOF ||
			next.Type == lexer.TOKEN_RPAREN || next.Type == lexer.TOKEN_RBRACE ||
			next.Type == lexer.TOKEN_HASH_COMMENT || next.Type == lexer.TOKEN_LINE_COMMENT {
			break
		}
		mod, err := p.parseModifier()
		if err != nil {
			return nil, err
		}
		col.Modifiers = append(col.Modifiers, mod)
	}
	return col, nil
}

func (p *parser) parseModifier() (ast.Modifier, error) {
	tok := p.advance()
	switch tok.Type {
	case lexer.TOKEN_KEYWORD_PRIMARY_KEY:
		return ast.Modifier{Kind: ast.ModPrimaryKey}, nil
	case lexer.TOKEN_KEYWORD_NULLABLE:
		return ast.Modifier{Kind: ast.ModNullable}, nil
	case lexer.TOKEN_KEYWORD_NOT_NULL:
		return ast.Modifier{Kind: ast.ModNotNull}, nil
	case lexer.TOKEN_KEYWORD_AUTO_INCREMENT:
		return ast.Modifier{Kind: ast.ModAutoIncrement}, nil
	case lexer.TOKEN_KEYWORD_INDEXED:
		return ast.Modifier{Kind: ast.ModIndexed}, nil
	case lexer.TOKEN_KEYWORD_DEFAULT:
		// expect ( value )
		if _, err := p.expect(lexer.TOKEN_LPAREN); err != nil {
			return ast.Modifier{}, err
		}
		val := p.advance()
		var valStr string
		switch val.Type {
		case lexer.TOKEN_STRING:
			valStr = `"` + val.Value + `"`
		case lexer.TOKEN_NUMBER:
			valStr = val.Value
		case lexer.TOKEN_IDENT:
			// Check for function-call syntax: IDENT()
			if p.peek().Type == lexer.TOKEN_LPAREN {
				p.advance() // consume (
				if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
					return ast.Modifier{}, err
				}
				valStr = val.Value + "()"
			} else {
				valStr = val.Value
			}
		default:
			return ast.Modifier{}, fmt.Errorf("line %d col %d: expected default value, got %q", val.Line, val.Col, val.Value)
		}
		if _, err := p.expect(lexer.TOKEN_RPAREN); err != nil {
			return ast.Modifier{}, err
		}
		return ast.Modifier{Kind: ast.ModDefault, Value: valStr}, nil
	default:
		return ast.Modifier{}, fmt.Errorf("line %d col %d: unknown modifier %q", tok.Line, tok.Col, tok.Value)
	}
}

func (p *parser) parseLink(comments []string) (*ast.Link, error) {
	tok, err := p.expect(lexer.TOKEN_KEYWORD_LINK)
	if err != nil {
		return nil, err
	}
	lnk := &ast.Link{Comments: comments, Line: tok.Line}

	fromCard, err := p.parseCardinality()
	if err != nil {
		return nil, err
	}
	lnk.FromCardinality = fromCard

	fromTable, fromCol, err := p.parseTableColumn()
	if err != nil {
		return nil, err
	}
	lnk.FromTable = fromTable
	lnk.FromColumn = fromCol

	if _, err := p.expect(lexer.TOKEN_KEYWORD_TO); err != nil {
		return nil, err
	}

	toCard, err := p.parseCardinality()
	if err != nil {
		return nil, err
	}
	lnk.ToCardinality = toCard

	toTable, toCol, err := p.parseTableColumn()
	if err != nil {
		return nil, err
	}
	lnk.ToTable = toTable
	lnk.ToColumn = toCol

	return lnk, nil
}

func (p *parser) parseCardinality() (ast.Cardinality, error) {
	tok := p.advance()
	switch tok.Type {
	case lexer.TOKEN_KEYWORD_ONE:
		return ast.CardOne, nil
	case lexer.TOKEN_KEYWORD_MANY:
		return ast.CardMany, nil
	default:
		return 0, fmt.Errorf("line %d col %d: expected 'one' or 'many', got %q", tok.Line, tok.Col, tok.Value)
	}
}

func (p *parser) parseTableColumn() (string, string, error) {
	tableTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return "", "", err
	}
	if _, err := p.expect(lexer.TOKEN_DOT); err != nil {
		return "", "", err
	}
	colTok, err := p.expect(lexer.TOKEN_IDENT)
	if err != nil {
		return "", "", err
	}
	return tableTok.Value, colTok.Value, nil
}

// ParseString is a convenience function that lexes and parses a source string.
func ParseString(src string) (*ast.Program, error) {
	l := &lexer.Lexer{}
	tokens, err := l.Tokenize(src)
	if err != nil {
		return nil, err
	}
	return Parse(tokens)
}

// formatTypeWithParams returns "type" or "type(p1,p2)".
func formatTypeWithParams(col *ast.Column) string {
	if len(col.TypeParams) == 0 {
		return col.Type
	}
	return col.Type + "(" + strings.Join(col.TypeParams, ",") + ")"
}

// FormatType is exported for use by render.
func FormatType(col *ast.Column) string {
	return formatTypeWithParams(col)
}
