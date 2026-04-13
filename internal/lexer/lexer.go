package lexer

import (
	"fmt"
	"strings"
	"unicode"
)

// TokenType identifies the kind of lexical token.
type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_IDENT
	TOKEN_NUMBER
	TOKEN_STRING
	TOKEN_HASH_COMMENT  // # comment text
	TOKEN_LINE_COMMENT  // // comment text
	TOKEN_LPAREN        // (
	TOKEN_RPAREN        // )
	TOKEN_LBRACE        // {
	TOKEN_RBRACE        // }
	TOKEN_DOT           // .
	TOKEN_COMMA         // ,
	TOKEN_KEYWORD_TABLE
	TOKEN_KEYWORD_LINK
	TOKEN_KEYWORD_ONE
	TOKEN_KEYWORD_MANY
	TOKEN_KEYWORD_TO
	TOKEN_KEYWORD_PRIMARY_KEY
	TOKEN_KEYWORD_NULLABLE
	TOKEN_KEYWORD_NOT_NULL
	TOKEN_KEYWORD_AUTO_INCREMENT
	TOKEN_KEYWORD_INDEXED
	TOKEN_KEYWORD_DEFAULT
	TOKEN_NEWLINE
)

// Token is a single lexical unit.
type Token struct {
	Type  TokenType
	Value string
	Line  int
	Col   int
}

var keywords = map[string]TokenType{
	"table":         TOKEN_KEYWORD_TABLE,
	"link":          TOKEN_KEYWORD_LINK,
	"one":           TOKEN_KEYWORD_ONE,
	"many":          TOKEN_KEYWORD_MANY,
	"to":            TOKEN_KEYWORD_TO,
	"primary-key":   TOKEN_KEYWORD_PRIMARY_KEY,
	"nullable":      TOKEN_KEYWORD_NULLABLE,
	"not-null":      TOKEN_KEYWORD_NOT_NULL,
	"auto-increment": TOKEN_KEYWORD_AUTO_INCREMENT,
	"indexed":       TOKEN_KEYWORD_INDEXED,
	"default":       TOKEN_KEYWORD_DEFAULT,
}

// Lexer tokenizes erdn-lang source input.
type Lexer struct{}

// Tokenize converts the input string into a slice of Tokens.
func (l *Lexer) Tokenize(input string) ([]Token, error) {
	var tokens []Token
	runes := []rune(input)
	i := 0
	line := 1
	lineStart := 0

	col := func() int { return i - lineStart + 1 }

	for i < len(runes) {
		ch := runes[i]

		// newline
		if ch == '\n' {
			tokens = append(tokens, Token{Type: TOKEN_NEWLINE, Value: "\n", Line: line, Col: col()})
			i++
			line++
			lineStart = i
			continue
		}

		// skip other whitespace
		if ch == '\r' || ch == '\t' || ch == ' ' {
			i++
			continue
		}

		// line comment //
		if ch == '/' && i+1 < len(runes) && runes[i+1] == '/' {
			start := i
			startCol := col()
			i += 2
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			tokens = append(tokens, Token{Type: TOKEN_LINE_COMMENT, Value: string(runes[start:i]), Line: line, Col: startCol})
			continue
		}

		// hash comment #
		if ch == '#' {
			startCol := col()
			i++ // skip #
			for i < len(runes) && runes[i] == ' ' {
				i++
			}
			start := i
			for i < len(runes) && runes[i] != '\n' {
				i++
			}
			text := strings.TrimSpace(string(runes[start:i]))
			tokens = append(tokens, Token{Type: TOKEN_HASH_COMMENT, Value: text, Line: line, Col: startCol})
			continue
		}

		// string literal
		if ch == '"' {
			startCol := col()
			i++
			var sb strings.Builder
			for i < len(runes) && runes[i] != '"' {
				if runes[i] == '\\' && i+1 < len(runes) {
					i++
					switch runes[i] {
					case 'n':
						sb.WriteRune('\n')
					case 't':
						sb.WriteRune('\t')
					case '"':
						sb.WriteRune('"')
					case '\\':
						sb.WriteRune('\\')
					default:
						sb.WriteRune('\\')
						sb.WriteRune(runes[i])
					}
				} else {
					sb.WriteRune(runes[i])
				}
				i++
			}
			if i >= len(runes) {
				return nil, fmt.Errorf("line %d: unterminated string literal", line)
			}
			i++ // closing "
			tokens = append(tokens, Token{Type: TOKEN_STRING, Value: sb.String(), Line: line, Col: startCol})
			continue
		}

		// symbols
		switch ch {
		case '(':
			tokens = append(tokens, Token{Type: TOKEN_LPAREN, Value: "(", Line: line, Col: col()})
			i++
			continue
		case ')':
			tokens = append(tokens, Token{Type: TOKEN_RPAREN, Value: ")", Line: line, Col: col()})
			i++
			continue
		case '{':
			tokens = append(tokens, Token{Type: TOKEN_LBRACE, Value: "{", Line: line, Col: col()})
			i++
			continue
		case '}':
			tokens = append(tokens, Token{Type: TOKEN_RBRACE, Value: "}", Line: line, Col: col()})
			i++
			continue
		case '.':
			tokens = append(tokens, Token{Type: TOKEN_DOT, Value: ".", Line: line, Col: col()})
			i++
			continue
		case ',':
			tokens = append(tokens, Token{Type: TOKEN_COMMA, Value: ",", Line: line, Col: col()})
			i++
			continue
		}

		// numbers
		if unicode.IsDigit(ch) {
			startCol := col()
			start := i
			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}
			tokens = append(tokens, Token{Type: TOKEN_NUMBER, Value: string(runes[start:i]), Line: line, Col: startCol})
			continue
		}

		// identifiers and keywords (including hyphenated keywords)
		if unicode.IsLetter(ch) || ch == '_' {
			startCol := col()
			start := i
			for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i]) || runes[i] == '_') {
				i++
			}
			word := string(runes[start:i])

			// check for hyphenated keywords: primary-key, not-null, auto-increment
			if i < len(runes) && runes[i] == '-' {
				// peek ahead
				j := i + 1
				for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
					j++
				}
				if j > i+1 {
					candidate := word + "-" + string(runes[i+1:j])
					if tt, ok := keywords[candidate]; ok {
						tokens = append(tokens, Token{Type: tt, Value: candidate, Line: line, Col: startCol})
						i = j
						continue
					}
				}
			}

			if tt, ok := keywords[word]; ok {
				tokens = append(tokens, Token{Type: tt, Value: word, Line: line, Col: startCol})
			} else {
				tokens = append(tokens, Token{Type: TOKEN_IDENT, Value: word, Line: line, Col: startCol})
			}
			continue
		}

		return nil, fmt.Errorf("line %d col %d: unexpected character %q", line, col(), ch)
	}

	tokens = append(tokens, Token{Type: TOKEN_EOF, Value: "", Line: line, Col: col()})
	return tokens, nil
}
