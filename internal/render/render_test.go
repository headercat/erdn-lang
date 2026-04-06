package render

import (
	"fmt"
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

// Link color and crow's foot notation tests.

func TestSVGLinkOneToManyGreen(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	if !strings.Contains(svg, "#27AE60") {
		t.Error("expected green (#27AE60) stroke for one-to-many link")
	}
}

func TestSVGLinkOneToOneBlue(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to one b.a_id`)
	if !strings.Contains(svg, "#3498DB") {
		t.Error("expected blue (#3498DB) stroke for one-to-one link")
	}
}

func TestSVGLinkManyToManyRed(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link many a.id to many b.a_id`)
	if !strings.Contains(svg, "#E74C3C") {
		t.Error("expected red (#E74C3C) stroke for many-to-many link")
	}
}

// TestSVGLinkNoTextCardinality verifies that the old text labels ("1" / "N")
// are no longer emitted and only SVG line elements are used for decorators.
func TestSVGLinkNoTextCardinality(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	// Old text labels must not appear.
	if strings.Contains(svg, `>1<`) || strings.Contains(svg, `>N<`) {
		t.Error("old text cardinality labels must not appear in SVG output")
	}
}

// TestSVGCrowsFootLinesPresent verifies that crow's foot <line> elements are
// present when a many-cardinality endpoint is rendered.
func TestSVGCrowsFootLinesPresent(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	// At least one <line> element must be present for the crow's foot symbol.
	if !strings.Contains(svg, "<line ") {
		t.Error("expected <line> elements for crow's foot symbol in SVG output")
	}
}

// TestSVGCrowsFootOpenFork verifies that the "many" crow's foot is an open fork
// (two diverging lines, no crossbar). A crossbar turns the fork into a closed
// triangle that looks like an unknown arrowhead.
func TestSVGCrowsFootOpenFork(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)

	// All link-related elements use stroke-width="1.5":
	//   1 connector <path>, 2 crow's-foot fork lines, 1 "one" bar → total 4.
	// If a crossbar were present the total would be 5 (1 path + 3 + 1).
	linkLines := strings.Count(svg, `stroke-width="1.5"`)
	if linkLines < 4 {
		t.Errorf("expected at least 4 stroke-width=1.5 elements, got %d", linkLines)
	}
	if linkLines > 4 {
		t.Errorf("got %d stroke-width=1.5 elements; expected exactly 4 (path + 2 fork + 1 bar) – crossbar may have been added", linkLines)
	}
}

// TestSVGLinkCommentNoBadgeArrowMarker verifies that the old arrowhead marker
// is no longer referenced (we draw symbols inline instead).
func TestSVGLinkCommentNoBadgeArrowMarker(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	if strings.Contains(svg, `marker-end`) {
		t.Error("marker-end must not appear; cardinality is now drawn inline")
	}
	if strings.Contains(svg, `id="arrow"`) {
		t.Error("arrow marker definition must not appear in SVG defs")
	}
}

// TestSVGSelfLoopCanvasNotClipped verifies that the SVG canvas width is large
// enough to contain the self-referential loop, which extends past the table's
// right edge by at least 50 px.
func TestSVGSelfLoopCanvasNotClipped(t *testing.T) {
	svg := generateSVG(t, `table categories (
  id bigint primary-key
  parent_id bigint nullable
)
link one categories.id to many categories.parent_id`)

	// Parse the width attribute from the <svg> element.
	widthPrefix := `width="`
	wi := strings.Index(svg, widthPrefix)
	if wi < 0 {
		t.Fatal("could not find width attribute in SVG")
	}
	wi += len(widthPrefix)
	we := strings.Index(svg[wi:], `"`)
	if we < 0 {
		t.Fatal("could not parse width attribute in SVG")
	}
	var canvasWidth float64
	if _, err := fmt.Sscanf(svg[wi:wi+we], "%f", &canvasWidth); err != nil {
		t.Fatalf("could not parse canvas width: %v", err)
	}

	// The self-referential loop must reach at least loopOffset=50 px past the
	// table right edge, and then the badge (if any) extends further.
	// The canvas must accommodate that plus svgMargin.
	// A minimum safe threshold: table right edge + loopOffset + svgMargin.
	// We don't know exact table width, but we do know the loop extends at least
	// 50 px past the table and must fit within the canvas with a margin.
	// Extract the path data to find the actual loopX.
	pi := strings.Index(svg, `<path d="M `)
	if pi < 0 {
		t.Fatal("could not find self-loop path in SVG")
	}
	// The path is: M sx,sy H loopX V dy H sx
	// Read the H loopX segment.
	pathSnip := svg[pi+len(`<path d="M `):]
	var sx, sy, loopX float64
	if _, err := fmt.Sscanf(pathSnip, "%f,%f H %f", &sx, &sy, &loopX); err != nil {
		t.Fatalf("could not parse self-loop path: %v", err)
	}

	minRequired := loopX + svgMargin
	if canvasWidth < minRequired-1 { // 1 px tolerance for %.2f path / %.0f canvas rounding
		t.Errorf("canvas width %.0f is too narrow: self-loop reaches x=%.0f, need at least %.0f (loopX + margin)",
			canvasWidth, loopX, minRequired)
	}
}

// TestSVGRegularLinkBadgeNotClipped verifies the canvas is wide enough for a
// link comment badge centered at the midpoint of the connector.
func TestSVGRegularLinkBadgeNotClipped(t *testing.T) {
	svg := generateSVG(t, `table a (id bigint)
table b (a_id bigint)
# a very long comment that makes the badge wide
link one a.id to many b.a_id`)

	widthPrefix := `width="`
	wi := strings.Index(svg, widthPrefix)
	if wi < 0 {
		t.Fatal("could not find width attribute in SVG")
	}
	wi += len(widthPrefix)
	we := strings.Index(svg[wi:], `"`)
	if we < 0 {
		t.Fatal("could not parse width attribute in SVG")
	}
	var canvasWidth float64
	if _, err := fmt.Sscanf(svg[wi:wi+we], "%f", &canvasWidth); err != nil {
		t.Fatalf("could not parse canvas width: %v", err)
	}

	if canvasWidth < 1 {
		t.Error("canvas width must be positive")
	}
	// The badge rect starts at x - w/2; ensure no rect has a negative x.
	if strings.Contains(svg, `x="-`) {
		t.Error("badge rect has negative x coordinate (clipped on left)")
	}
}
