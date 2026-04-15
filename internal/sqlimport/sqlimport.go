package sqlimport

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/headercat/erdn-lang/internal/ast"
	"github.com/headercat/erdn-lang/internal/parser"
)

// ParseDDL parses SQL CREATE TABLE statements and converts them into an ERDN AST.
// It supports table-level/inline primary keys and foreign keys.
func ParseDDL(sql string) (*ast.Program, error) {
	clean := stripSQLComments(sql)

	var tables []*ast.Table
	var links []*ast.Link
	seenLinks := map[string]bool{}

	rest := clean
	for {
		idx := indexCreateTable(rest)
		if idx < 0 {
			break
		}
		rest = rest[idx+len("create table"):]

		tableName, afterName, err := parseIdentifier(rest)
		if err != nil {
			return nil, fmt.Errorf("parse table name: %w", err)
		}
		rest = afterName
		tableName = sanitizeIdentifier(baseIdentifier(tableName))

		open := strings.Index(rest, "(")
		if open < 0 {
			return nil, fmt.Errorf("missing '(' after CREATE TABLE %s", tableName)
		}
		rest = rest[open:]
		body, afterBody, err := extractParenthesized(rest)
		if err != nil {
			return nil, fmt.Errorf("parse table body for %s: %w", tableName, err)
		}
		rest = afterBody

		tbl, tblLinks, err := parseTableBody(tableName, body)
		if err != nil {
			return nil, err
		}
		tables = append(tables, tbl)
		for _, l := range tblLinks {
			key := fmt.Sprintf("%s.%s>%s.%s", l.FromTable, l.FromColumn, l.ToTable, l.ToColumn)
			if seenLinks[key] {
				continue
			}
			seenLinks[key] = true
			links = append(links, l)
		}
	}

	return &ast.Program{Tables: tables, Links: links}, nil
}

// ToERDN formats an ERDN AST as textual erdn-lang source.
func ToERDN(prog *ast.Program) string {
	var b strings.Builder
	for i, t := range prog.Tables {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "table %s (\n", t.Name)
		for _, c := range t.Columns {
			fmt.Fprintf(&b, "  %s %s", c.Name, parser.FormatType(c))
			if mods := formatModifiers(c.Modifiers); mods != "" {
				b.WriteByte(' ')
				b.WriteString(mods)
			}
			b.WriteByte('\n')
		}
		b.WriteString(")")
	}

	if len(prog.Links) > 0 {
		b.WriteString("\n\n")
		for i, l := range prog.Links {
			if i > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "link %s %s.%s to %s %s.%s",
				cardToText(l.FromCardinality), l.FromTable, l.FromColumn,
				cardToText(l.ToCardinality), l.ToTable, l.ToColumn)
		}
	}

	b.WriteByte('\n')
	return b.String()
}

func parseTableBody(tableName, body string) (*ast.Table, []*ast.Link, error) {
	parts := splitTopLevelComma(body)
	tbl := &ast.Table{Name: tableName}

	var links []*ast.Link
	pkCols := map[string]bool{}

	for _, raw := range parts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		lower := strings.ToLower(part)

		if strings.HasPrefix(lower, "primary key") {
			for _, c := range parseColumnList(part) {
				pkCols[sanitizeIdentifier(baseIdentifier(c))] = true
			}
			continue
		}
		if strings.HasPrefix(lower, "constraint") && strings.Contains(lower, " primary key") {
			for _, c := range parseColumnList(part) {
				pkCols[sanitizeIdentifier(baseIdentifier(c))] = true
			}
			continue
		}

		if strings.HasPrefix(lower, "foreign key") {
			localCols := parseColumnList(part)
			refTable, refCols := parseFKReference(part)
			refTable = sanitizeIdentifier(baseIdentifier(refTable))
			n := len(localCols)
			if len(refCols) < n {
				n = len(refCols)
			}
			for i := 0; i < n; i++ {
				links = append(links, &ast.Link{
					FromTable:       sanitizeIdentifier(baseIdentifier(refTable)),
					FromColumn:      sanitizeIdentifier(baseIdentifier(refCols[i])),
					ToTable:         tableName,
					ToColumn:        sanitizeIdentifier(baseIdentifier(localCols[i])),
					FromCardinality: ast.CardOne,
					ToCardinality:   ast.CardMany,
				})
			}
			continue
		}
		if strings.HasPrefix(lower, "constraint") && strings.Contains(lower, " foreign key") {
			localCols := parseColumnList(part)
			refTable, refCols := parseFKReference(part)
			refTable = sanitizeIdentifier(baseIdentifier(refTable))
			n := len(localCols)
			if len(refCols) < n {
				n = len(refCols)
			}
			for i := 0; i < n; i++ {
				links = append(links, &ast.Link{
					FromTable:       sanitizeIdentifier(baseIdentifier(refTable)),
					FromColumn:      sanitizeIdentifier(baseIdentifier(refCols[i])),
					ToTable:         tableName,
					ToColumn:        sanitizeIdentifier(baseIdentifier(localCols[i])),
					FromCardinality: ast.CardOne,
					ToCardinality:   ast.CardMany,
				})
			}
			continue
		}

		col, inlineRef := parseColumnDef(part)
		if col == nil {
			continue
		}
		tbl.Columns = append(tbl.Columns, col)
		if inlineRef != nil {
			links = append(links, &ast.Link{
				FromTable:       inlineRef.table,
				FromColumn:      inlineRef.column,
				ToTable:         tableName,
				ToColumn:        col.Name,
				FromCardinality: ast.CardOne,
				ToCardinality:   ast.CardMany,
			})
		}
	}

	for _, c := range tbl.Columns {
		if pkCols[c.Name] {
			c.Modifiers = append(c.Modifiers, ast.Modifier{Kind: ast.ModPrimaryKey})
		}
	}

	return tbl, links, nil
}

type ref struct {
	table  string
	column string
}

func parseColumnDef(part string) (*ast.Column, *ref) {
	name, rest, err := parseIdentifier(part)
	if err != nil {
		return nil, nil
	}
	name = sanitizeIdentifier(baseIdentifier(name))
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil, nil
	}

	typeExpr, modifiersExpr := splitTypeAndModifiers(rest)
	col := &ast.Column{Name: name}
	populateType(col, typeExpr)

	lower := strings.ToLower(modifiersExpr)
	if strings.Contains(lower, "primary key") {
		col.Modifiers = append(col.Modifiers, ast.Modifier{Kind: ast.ModPrimaryKey})
	}
	if strings.Contains(lower, "not null") {
		col.Modifiers = append(col.Modifiers, ast.Modifier{Kind: ast.ModNotNull})
	} else if strings.Contains(lower, " null") {
		col.Modifiers = append(col.Modifiers, ast.Modifier{Kind: ast.ModNullable})
	}
	if strings.Contains(lower, "auto_increment") ||
		strings.Contains(lower, " identity") ||
		strings.Contains(lower, "generated always as identity") {
		col.Modifiers = append(col.Modifiers, ast.Modifier{Kind: ast.ModAutoIncrement})
	}
	if strings.Contains(lower, " unique") || strings.Contains(lower, " index") {
		col.Modifiers = append(col.Modifiers, ast.Modifier{Kind: ast.ModIndexed})
	}

	if def := parseDefault(modifiersExpr); def != "" {
		col.Modifiers = append(col.Modifiers, ast.Modifier{Kind: ast.ModDefault, Value: def})
	}

	if strings.Contains(lower, "references ") {
		tbl, cols := parseFKReference(part)
		if tbl != "" && len(cols) > 0 {
			return col, &ref{
				table:  sanitizeIdentifier(baseIdentifier(tbl)),
				column: sanitizeIdentifier(baseIdentifier(cols[0])),
			}
		}
	}

	return col, nil
}

func parseDefault(s string) string {
	lower := strings.ToLower(s)
	i := strings.Index(lower, " default ")
	if i < 0 {
		return ""
	}
	after := strings.TrimSpace(s[i+len(" default "):])
	if after == "" {
		return ""
	}

	if after[0] == '\'' {
		end := strings.Index(after[1:], "'")
		if end >= 0 {
			return `"` + after[1:1+end] + `"`
		}
	}
	if after[0] == '"' {
		end := strings.Index(after[1:], `"`)
		if end >= 0 {
			return `"` + after[1:1+end] + `"`
		}
	}
	if strings.HasPrefix(strings.ToUpper(after), "CURRENT_TIMESTAMP") {
		return "NOW()"
	}
	if strings.HasPrefix(strings.ToUpper(after), "NOW()") {
		return "NOW()"
	}

	for i := 0; i < len(after); i++ {
		if unicode.IsSpace(rune(after[i])) || after[i] == ',' {
			return after[:i]
		}
	}
	return after
}

func populateType(col *ast.Column, typeExpr string) {
	typeExpr = strings.TrimSpace(typeExpr)
	if typeExpr == "" {
		col.Type = "text"
		return
	}
	open := strings.Index(typeExpr, "(")
	if open < 0 || !strings.HasSuffix(typeExpr, ")") {
		col.Type = strings.ToLower(strings.TrimSpace(typeExpr))
		return
	}
	col.Type = strings.ToLower(strings.TrimSpace(typeExpr[:open]))
	params := strings.TrimSpace(typeExpr[open+1 : len(typeExpr)-1])
	if params == "" {
		return
	}
	for _, p := range strings.Split(params, ",") {
		col.TypeParams = append(col.TypeParams, strings.TrimSpace(p))
	}
}

func splitTypeAndModifiers(s string) (string, string) {
	var b strings.Builder
	depth := 0
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		ch := r[i]
		if ch == '(' {
			depth++
		} else if ch == ')' && depth > 0 {
			depth--
		}
		if depth == 0 && unicode.IsSpace(ch) {
			remaining := strings.TrimSpace(string(r[i:]))
			lower := strings.ToLower(remaining)
			if startsConstraint(lower) {
				return strings.TrimSpace(b.String()), remaining
			}
		}
		b.WriteRune(ch)
	}
	return strings.TrimSpace(b.String()), ""
}

func startsConstraint(s string) bool {
	for _, k := range []string{
		"not null", "null", "default", "primary key", "unique", "references",
		"constraint", "auto_increment", "identity", "generated",
	} {
		if strings.HasPrefix(s, k) {
			return true
		}
	}
	return false
}

func parseFKReference(s string) (string, []string) {
	lower := strings.ToLower(s)
	i := strings.Index(lower, "references")
	if i < 0 {
		return "", nil
	}
	rest := strings.TrimSpace(s[i+len("references"):])
	table, rest, err := parseIdentifier(rest)
	if err != nil {
		return "", nil
	}
	cols := parseColumnList(rest)
	return table, cols
}

func parseColumnList(s string) []string {
	open := strings.Index(s, "(")
	if open < 0 {
		return nil
	}
	body, _, err := extractParenthesized(s[open:])
	if err != nil {
		return nil
	}
	parts := splitTopLevelComma(body)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, _, err := parseIdentifier(p)
		if err != nil {
			continue
		}
		out = append(out, name)
	}
	return out
}

func stripSQLComments(s string) string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func indexCreateTable(s string) int {
	return strings.Index(strings.ToLower(s), "create table")
}

func extractParenthesized(s string) (inside, rest string, err error) {
	if len(s) == 0 || s[0] != '(' {
		return "", "", fmt.Errorf("expected '('")
	}
	depth := 0
	inSingle := false
	inDouble := false
	r := []rune(s)
	for i, ch := range r {
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		}
		if inSingle || inDouble {
			continue
		}
		if ch == '(' {
			depth++
		}
		if ch == ')' {
			depth--
			if depth == 0 {
				return string(r[1:i]), string(r[i+1:]), nil
			}
		}
	}
	return "", "", fmt.Errorf("unterminated parenthesized block")
}

func splitTopLevelComma(s string) []string {
	var parts []string
	depth := 0
	inSingle := false
	inDouble := false
	start := 0
	r := []rune(s)
	for i, ch := range r {
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		}
		if inSingle || inDouble {
			continue
		}
		if ch == '(' {
			depth++
			continue
		}
		if ch == ')' {
			if depth > 0 {
				depth--
			}
			continue
		}
		if ch == ',' && depth == 0 {
			parts = append(parts, string(r[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, string(r[start:]))
	return parts
}

func parseIdentifier(s string) (ident, rest string, err error) {
	s = strings.TrimLeftFunc(s, unicode.IsSpace)
	if s == "" {
		return "", "", fmt.Errorf("empty identifier")
	}
	switch s[0] {
	case '`':
		end := strings.Index(s[1:], "`")
		if end < 0 {
			return "", "", fmt.Errorf("unterminated backtick identifier")
		}
		return s[1 : 1+end], s[2+end:], nil
	case '"':
		end := strings.Index(s[1:], `"`)
		if end < 0 {
			return "", "", fmt.Errorf("unterminated quoted identifier")
		}
		return s[1 : 1+end], s[2+end:], nil
	case '[':
		end := strings.Index(s[1:], "]")
		if end < 0 {
			return "", "", fmt.Errorf("unterminated bracket identifier")
		}
		return s[1 : 1+end], s[2+end:], nil
	default:
		i := 0
		for i < len(s) {
			ch := s[i]
			if unicode.IsSpace(rune(ch)) || ch == '(' || ch == ')' || ch == ',' {
				break
			}
			i++
		}
		if i == 0 {
			return "", "", fmt.Errorf("invalid identifier")
		}
		return s[:i], s[i:], nil
	}
}

func baseIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		return strings.TrimSpace(s[idx+1:])
	}
	return s
}

func sanitizeIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "_"
	}
	var out []rune
	for i, r := range []rune(s) {
		ok := unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		if i == 0 && !(unicode.IsLetter(r) || r == '_') {
			ok = false
		}
		if ok {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	return strings.Trim(strings.ReplaceAll(string(out), "__", "_"), "_")
}

func cardToText(c ast.Cardinality) string {
	if c == ast.CardMany {
		return "many"
	}
	return "one"
}

func formatModifiers(mods []ast.Modifier) string {
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		switch m.Kind {
		case ast.ModPrimaryKey:
			out = append(out, "primary-key")
		case ast.ModNullable:
			out = append(out, "nullable")
		case ast.ModNotNull:
			out = append(out, "not-null")
		case ast.ModAutoIncrement:
			out = append(out, "auto-increment")
		case ast.ModIndexed:
			out = append(out, "indexed")
		case ast.ModDefault:
			out = append(out, "default("+m.Value+")")
		}
	}
	return strings.Join(out, " ")
}
