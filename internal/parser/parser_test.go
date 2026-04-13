package parser

import (
	"testing"

	"github.com/headercat/erdn-lang/internal/ast"
)

func TestParseSimpleTable(t *testing.T) {
	src := `table users (
  id bigint primary-key auto-increment
  name varchar(255) not-null
)`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(prog.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(prog.Tables))
	}
	tbl := prog.Tables[0]
	if tbl.Name != "users" {
		t.Errorf("expected table name 'users', got %q", tbl.Name)
	}
	if len(tbl.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(tbl.Columns))
	}
	col := tbl.Columns[0]
	if col.Name != "id" || col.Type != "bigint" {
		t.Errorf("unexpected column: %+v", col)
	}
	if len(col.Modifiers) != 2 {
		t.Errorf("expected 2 modifiers, got %d", len(col.Modifiers))
	}
}

func TestParseTableWithBraces(t *testing.T) {
	src := `table things {
  val int
}`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(prog.Tables) != 1 {
		t.Fatalf("expected 1 table")
	}
}

func TestParseColumnTypeParams(t *testing.T) {
	src := `table t (
  price decimal(10,2)
)`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	col := prog.Tables[0].Columns[0]
	if len(col.TypeParams) != 2 || col.TypeParams[0] != "10" || col.TypeParams[1] != "2" {
		t.Errorf("unexpected type params: %v", col.TypeParams)
	}
}

func TestParseDefaultModifier(t *testing.T) {
	src := `table t (
  title varchar(255) default("Untitled")
)`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	col := prog.Tables[0].Columns[0]
	if len(col.Modifiers) != 1 || col.Modifiers[0].Kind != ast.ModDefault {
		t.Errorf("expected default modifier")
	}
	if col.Modifiers[0].Value != `"Untitled"` {
		t.Errorf(`expected default value '"Untitled"', got %q`, col.Modifiers[0].Value)
	}
}

func TestParseDefaultFunctionCall(t *testing.T) {
	src := `table t (
  created_at timestamp default(NOW())
)`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	col := prog.Tables[0].Columns[0]
	if len(col.Modifiers) != 1 || col.Modifiers[0].Kind != ast.ModDefault {
		t.Errorf("expected default modifier")
	}
	if col.Modifiers[0].Value != "NOW()" {
		t.Errorf("expected default value 'NOW()', got %q", col.Modifiers[0].Value)
	}
}

func TestParseLink(t *testing.T) {
	src := `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(prog.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(prog.Links))
	}
	lnk := prog.Links[0]
	if lnk.FromTable != "a" || lnk.FromColumn != "id" {
		t.Errorf("unexpected from: %s.%s", lnk.FromTable, lnk.FromColumn)
	}
	if lnk.ToTable != "b" || lnk.ToColumn != "a_id" {
		t.Errorf("unexpected to: %s.%s", lnk.ToTable, lnk.ToColumn)
	}
	if lnk.FromCardinality != ast.CardOne || lnk.ToCardinality != ast.CardMany {
		t.Errorf("unexpected cardinalities")
	}
}

func TestParseHashComments(t *testing.T) {
	src := `# table comment
table users (
  # column comment
  id bigint
)`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	tbl := prog.Tables[0]
	if len(tbl.Comments) != 1 || tbl.Comments[0] != "table comment" {
		t.Errorf("expected table comment, got %v", tbl.Comments)
	}
	col := tbl.Columns[0]
	if len(col.Comments) != 1 || col.Comments[0] != "column comment" {
		t.Errorf("expected column comment, got %v", col.Comments)
	}
}

func TestParseLinkComment(t *testing.T) {
	src := `table a (id bigint)
table b (a_id bigint)
# relationship comment
link one a.id to many b.a_id`
	prog, err := ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	lnk := prog.Links[0]
	if len(lnk.Comments) != 1 || lnk.Comments[0] != "relationship comment" {
		t.Errorf("expected link comment, got %v", lnk.Comments)
	}
}
