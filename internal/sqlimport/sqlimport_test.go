package sqlimport

import (
	"strings"
	"testing"

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
