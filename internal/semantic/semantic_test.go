package semantic

import (
	"testing"

	"github.com/headercat/erdn-lang/internal/parser"
)

func validate(t *testing.T, src string) []error {
	t.Helper()
	prog, err := parser.ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return Validate(prog)
}

func TestNoDuplicateTables(t *testing.T) {
	errs := validate(t, `
table a (id bigint)
table a (id bigint)
`)
	if len(errs) == 0 {
		t.Error("expected duplicate table error")
	}
}

func TestNoDuplicateColumns(t *testing.T) {
	errs := validate(t, `
table t (
  id bigint
  id bigint
)
`)
	if len(errs) == 0 {
		t.Error("expected duplicate column error")
	}
}

func TestBadModifierCombination(t *testing.T) {
	errs := validate(t, `
table t (
  x int nullable not-null
)
`)
	if len(errs) == 0 {
		t.Error("expected nullable+not-null error")
	}
}

func TestAtMostOnePrimaryKey(t *testing.T) {
	errs := validate(t, `
table t (
  a int primary-key
  b int primary-key
)
`)
	if len(errs) == 0 {
		t.Error("expected multiple primary key error")
	}
}

func TestLinkMissingTable(t *testing.T) {
	errs := validate(t, `
table a (id bigint)
link one a.id to many b.id
`)
	if len(errs) == 0 {
		t.Error("expected missing table error")
	}
}

func TestLinkMissingColumn(t *testing.T) {
	errs := validate(t, `
table a (id bigint)
table b (x bigint)
link one a.id to many b.missing
`)
	if len(errs) == 0 {
		t.Error("expected missing column error")
	}
}

func TestValidSchema(t *testing.T) {
	errs := validate(t, `
table users (
  id bigint primary-key auto-increment
  name varchar(255) not-null
)
table posts (
  id bigint primary-key auto-increment
  author_id bigint not-null
)
link one users.id to many posts.author_id
`)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}
