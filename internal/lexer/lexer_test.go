package lexer

import (
	"testing"
)

func tokenize(t *testing.T, input string) []Token {
	t.Helper()
	l := &Lexer{}
	tokens, err := l.Tokenize(input)
	if err != nil {
		t.Fatalf("tokenize error: %v", err)
	}
	return tokens
}

func findType(tokens []Token, tt TokenType) *Token {
	for i := range tokens {
		if tokens[i].Type == tt {
			return &tokens[i]
		}
	}
	return nil
}

func TestTokenizeIdentifier(t *testing.T) {
	tokens := tokenize(t, "hello world")
	idents := []string{}
	for _, tok := range tokens {
		if tok.Type == TOKEN_IDENT {
			idents = append(idents, tok.Value)
		}
	}
	if len(idents) != 2 || idents[0] != "hello" || idents[1] != "world" {
		t.Errorf("expected [hello world], got %v", idents)
	}
}

func TestTokenizeKeywords(t *testing.T) {
	cases := []struct {
		input string
		want  TokenType
	}{
		{"table", TOKEN_KEYWORD_TABLE},
		{"link", TOKEN_KEYWORD_LINK},
		{"one", TOKEN_KEYWORD_ONE},
		{"many", TOKEN_KEYWORD_MANY},
		{"to", TOKEN_KEYWORD_TO},
		{"primary-key", TOKEN_KEYWORD_PRIMARY_KEY},
		{"nullable", TOKEN_KEYWORD_NULLABLE},
		{"not-null", TOKEN_KEYWORD_NOT_NULL},
		{"auto-increment", TOKEN_KEYWORD_AUTO_INCREMENT},
		{"indexed", TOKEN_KEYWORD_INDEXED},
		{"default", TOKEN_KEYWORD_DEFAULT},
	}
	for _, c := range cases {
		l := &Lexer{}
		tokens, err := l.Tokenize(c.input)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", c.input, err)
			continue
		}
		if tokens[0].Type != c.want {
			t.Errorf("%q: expected type %d, got %d", c.input, c.want, tokens[0].Type)
		}
	}
}

func TestTokenizeHashComment(t *testing.T) {
	tokens := tokenize(t, "# this is a comment\ntable")
	tok := findType(tokens, TOKEN_HASH_COMMENT)
	if tok == nil {
		t.Fatal("expected hash comment token")
	}
	if tok.Value != "this is a comment" {
		t.Errorf("expected comment text, got %q", tok.Value)
	}
}

func TestTokenizeLineComment(t *testing.T) {
	tokens := tokenize(t, "// ignored\ntable")
	tok := findType(tokens, TOKEN_LINE_COMMENT)
	if tok == nil {
		t.Fatal("expected line comment token")
	}
}

func TestTokenizeString(t *testing.T) {
	tokens := tokenize(t, `"hello world"`)
	tok := findType(tokens, TOKEN_STRING)
	if tok == nil {
		t.Fatal("expected string token")
	}
	if tok.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", tok.Value)
	}
}

func TestTokenizeNumber(t *testing.T) {
	tokens := tokenize(t, "42 3.14")
	nums := []string{}
	for _, tok := range tokens {
		if tok.Type == TOKEN_NUMBER {
			nums = append(nums, tok.Value)
		}
	}
	if len(nums) != 2 || nums[0] != "42" || nums[1] != "3.14" {
		t.Errorf("expected [42 3.14], got %v", nums)
	}
}

func TestTokenizeSymbols(t *testing.T) {
	tokens := tokenize(t, "( ) { } . ,")
	expected := []TokenType{TOKEN_LPAREN, TOKEN_RPAREN, TOKEN_LBRACE, TOKEN_RBRACE, TOKEN_DOT, TOKEN_COMMA}
	j := 0
	for _, tok := range tokens {
		if tok.Type == TOKEN_EOF || tok.Type == TOKEN_NEWLINE {
			continue
		}
		if j >= len(expected) {
			t.Fatalf("too many tokens")
		}
		if tok.Type != expected[j] {
			t.Errorf("token[%d]: expected %d, got %d (%q)", j, expected[j], tok.Type, tok.Value)
		}
		j++
	}
}

func TestTokenizeLineNumbers(t *testing.T) {
	tokens := tokenize(t, "table\nlink")
	var tableToken, linkToken *Token
	for i := range tokens {
		if tokens[i].Type == TOKEN_KEYWORD_TABLE {
			tableToken = &tokens[i]
		}
		if tokens[i].Type == TOKEN_KEYWORD_LINK {
			linkToken = &tokens[i]
		}
	}
	if tableToken == nil || tableToken.Line != 1 {
		t.Errorf("expected table on line 1, got %v", tableToken)
	}
	if linkToken == nil || linkToken.Line != 2 {
		t.Errorf("expected link on line 2, got %v", linkToken)
	}
}

func TestTokenizeTypeWithParams(t *testing.T) {
	tokens := tokenize(t, "varchar(255)")
	types := map[TokenType]string{}
	for _, tok := range tokens {
		types[tok.Type] = tok.Value
	}
	if types[TOKEN_IDENT] != "varchar" {
		t.Errorf("expected varchar ident")
	}
	if types[TOKEN_NUMBER] != "255" {
		t.Errorf("expected 255 number")
	}
}
