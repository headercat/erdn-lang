package sqlimport

import (
	"strings"
	"testing"

	"github.com/headercat/erdn-lang/internal/ast"
	"github.com/headercat/erdn-lang/internal/semantic"
)

func TestParseDDL_BasicTableAndForeignKey(t *testing.T) {
	sql := `
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE posts (
  id BIGINT PRIMARY KEY,
  author_id BIGINT NOT NULL,
  title VARCHAR(255) NOT NULL,
  CONSTRAINT fk_posts_author FOREIGN KEY (author_id) REFERENCES users(id)
);`

	prog, err := ParseDDL(sql)
	if err != nil {
		t.Fatalf("ParseDDL error: %v", err)
	}
	if len(prog.Tables) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(prog.Tables))
	}
	if len(prog.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(prog.Links))
	}
	link := prog.Links[0]
	if link.FromTable != "users" {
		t.Fatalf("unexpected from table: %q", link.FromTable)
	}
	if link.FromColumn != "id" {
		t.Fatalf("unexpected from column: %q", link.FromColumn)
	}
	if link.ToTable != "posts" {
		t.Fatalf("unexpected to table: %q", link.ToTable)
	}
	if link.ToColumn != "author_id" {
		t.Fatalf("unexpected to column: %q", link.ToColumn)
	}
	if link.FromCardinality != ast.CardOne {
		t.Fatalf("unexpected from cardinality: %v", link.FromCardinality)
	}
	if link.ToCardinality != ast.CardMany {
		t.Fatalf("unexpected to cardinality: %v", link.ToCardinality)
	}
	if errs := semantic.Validate(prog); len(errs) > 0 {
		t.Fatalf("semantic errors: %v", errs)
	}

	erdn := ToERDN(prog)
	if !strings.Contains(erdn, "table users (") {
		t.Fatalf("missing users table in ERDN:\n%s", erdn)
	}
	if !strings.Contains(erdn, "link one users.id to many posts.author_id") {
		t.Fatalf("missing expected link in ERDN:\n%s", erdn)
	}
}

func TestParseDDL_QuotedAndNonLatinIdentifiers(t *testing.T) {
	sql := `
CREATE TABLE "用户" (
  "编号" BIGINT PRIMARY KEY,
  "名称" VARCHAR(128) NOT NULL
);`

	prog, err := ParseDDL(sql)
	if err != nil {
		t.Fatalf("ParseDDL error: %v", err)
	}
	if len(prog.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(prog.Tables))
	}
	if prog.Tables[0].Name != "用户" {
		t.Fatalf("expected table name 用户, got %q", prog.Tables[0].Name)
	}
	if len(prog.Tables[0].Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(prog.Tables[0].Columns))
	}
}

func TestParseDDL_InlineReference(t *testing.T) {
	sql := `
CREATE TABLE customers (
  id BIGINT PRIMARY KEY
);
CREATE TABLE orders (
  id BIGINT PRIMARY KEY,
  customer_id BIGINT REFERENCES customers(id)
);`

	prog, err := ParseDDL(sql)
	if err != nil {
		t.Fatalf("ParseDDL error: %v", err)
	}
	if len(prog.Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(prog.Links))
	}
}
