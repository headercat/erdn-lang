package render

import (
	"strings"
	"testing"

	"github.com/headercat/erdn-lang/internal/parser"
)

func generateSVG(t *testing.T, src string) string {
	t.Helper()
	prog, err := parser.ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return GenerateSVG(prog)
}

func TestSVGContainsTableNode(t *testing.T) {
	svg := generateSVG(t, `table users (
  id bigint primary-key
  name varchar(255)
)`)
	if !strings.Contains(svg, "users") {
		t.Error("expected 'users' in SVG output")
	}
	if !strings.Contains(svg, "id") {
		t.Error("expected 'id' column in SVG output")
	}
	if !strings.Contains(svg, "bigint") {
		t.Error("expected 'bigint' type in SVG output")
	}
}

func TestSVGContainsEdge(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	if !strings.Contains(svg, "<path") {
		t.Error("expected <path element in SVG output")
	}
}

func TestSVGContainsComment(t *testing.T) {
	svg := generateSVG(t, `# my table
table t (id bigint)`)
	if !strings.Contains(svg, "my table") {
		t.Error("expected table comment in SVG output")
	}
}

func TestSVGModifiers(t *testing.T) {
	svg := generateSVG(t, `table t (
  id bigint primary-key auto-increment
  name varchar(255) not-null default("hi")
)`)
	if !strings.Contains(svg, "PK") {
		t.Error("expected PK modifier in SVG")
	}
	if !strings.Contains(svg, "AI") {
		t.Error("expected AI modifier in SVG")
	}
	if !strings.Contains(svg, "NN") {
		t.Error("expected NN modifier in SVG")
	}
}

func TestSVGColumnComment(t *testing.T) {
	svg := generateSVG(t, `table t (
  # column doc
  id bigint
)`)
	if !strings.Contains(svg, "column doc") {
		t.Error("expected column comment in SVG output")
	}
}

func TestSVGLinkComment(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
# connects a to b
link one a.id to many b.a_id`)
	if !strings.Contains(svg, "connects a to b") {
		t.Error("expected link comment in SVG output")
	}
}

func TestSVGCJK(t *testing.T) {
	svg := generateSVG(t, `# 用户表
table 用户 (
  # 用户ID
  id bigint primary-key
)`)
	if !strings.Contains(svg, "用户") {
		t.Error("expected CJK table name in SVG output")
	}
	if !strings.Contains(svg, "用户ID") {
		t.Error("expected CJK column comment in SVG output")
	}
}

func TestSVGSelfReferentialLink(t *testing.T) {
	svg := generateSVG(t, `table categories (
  id bigint primary-key
  parent_id bigint nullable
)
# self-reference
link one categories.id to many categories.parent_id`)
	if !strings.Contains(svg, "<path") {
		t.Error("expected <path element for self-referential link")
	}
	if !strings.Contains(svg, "self-reference") {
		t.Error("expected self-reference comment in SVG output")
	}
}

func TestSVGDoubleSlashCommentNotRendered(t *testing.T) {
	svg := generateSVG(t, `// this must not appear
table t (
  // column note excluded
  id bigint
)
// link note excluded
table u (ref bigint)
link one t.id to many u.ref`)
	if strings.Contains(svg, "must not appear") {
		t.Error("// comment must not appear in SVG output")
	}
	if strings.Contains(svg, "column note excluded") {
		t.Error("// column comment must not appear in SVG output")
	}
	if strings.Contains(svg, "link note excluded") {
		t.Error("// link comment must not appear in SVG output")
	}
}

func TestSVGMultiLineTableComment(t *testing.T) {
	svg := generateSVG(t, `# first line
# second line
table t (id bigint)`)
	if !strings.Contains(svg, "first line") {
		t.Error("expected first comment line in SVG output")
	}
	if !strings.Contains(svg, "second line") {
		t.Error("expected second comment line in SVG output")
	}
}

func TestSVGMultipleTables(t *testing.T) {
	svg := generateSVG(t, `table foo (x int)
table bar (y int)`)
	if !strings.Contains(svg, "foo") || !strings.Contains(svg, "bar") {
		t.Error("expected both tables in SVG output")
	}
}
