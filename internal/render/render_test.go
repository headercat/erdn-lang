package render

import (
	"strings"
	"testing"

	"github.com/headercat/erdn-lang/internal/parser"
)

func generateDOT(t *testing.T, src string) string {
	t.Helper()
	prog, err := parser.ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return GenerateDOT(prog)
}

func TestDOTContainsTableNode(t *testing.T) {
	dot := generateDOT(t, `table users (
  id bigint primary-key
  name varchar(255)
)`)
	if !strings.Contains(dot, "users") {
		t.Error("expected 'users' in DOT output")
	}
	if !strings.Contains(dot, "id") {
		t.Error("expected 'id' column in DOT output")
	}
	if !strings.Contains(dot, "bigint") {
		t.Error("expected 'bigint' type in DOT output")
	}
}

func TestDOTContainsEdge(t *testing.T) {
	dot := generateDOT(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	if !strings.Contains(dot, "->") {
		t.Error("expected edge arrow in DOT output")
	}
	if !strings.Contains(dot, `"1"`) || !strings.Contains(dot, `"N"`) {
		t.Error("expected cardinality labels in DOT output")
	}
}

func TestDOTContainsComment(t *testing.T) {
	dot := generateDOT(t, `# my table
table t (id bigint)`)
	if !strings.Contains(dot, "my table") {
		t.Error("expected comment in DOT output")
	}
}

func TestDOTMultipleTables(t *testing.T) {
	dot := generateDOT(t, `table foo (x int)
table bar (y int)`)
	if !strings.Contains(dot, "foo") || !strings.Contains(dot, "bar") {
		t.Error("expected both tables in DOT output")
	}
}

func TestDOTModifiers(t *testing.T) {
	dot := generateDOT(t, `table t (
  id bigint primary-key auto-increment
  name varchar(255) not-null default("hi")
)`)
	if !strings.Contains(dot, "PK") {
		t.Error("expected PK modifier")
	}
	if !strings.Contains(dot, "AI") {
		t.Error("expected AI modifier")
	}
}
