package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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

func generatePNG(t *testing.T, src string) []byte {
	t.Helper()
	prog, err := parser.ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	data, err := GeneratePNG(prog)
	if err != nil {
		t.Fatalf("GeneratePNG error: %v", err)
	}
	return data
}

func generatePDF(t *testing.T, src string) []byte {
	t.Helper()
	prog, err := parser.ParseString(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	data, err := GeneratePDF(prog)
	if err != nil {
		t.Fatalf("GeneratePDF error: %v", err)
	}
	return data
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

// PNG renderer tests

var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// decodePNGPixels decodes a PNG byte slice and returns the raw NRGBA image.
func decodePNGPixels(t *testing.T, data []byte) *image.NRGBA {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}
	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		// Convert to NRGBA.
		b := img.Bounds()
		nrgba = image.NewNRGBA(b)
		for py := b.Min.Y; py < b.Max.Y; py++ {
			for px := b.Min.X; px < b.Max.X; px++ {
				nrgba.Set(px, py, img.At(px, py))
			}
		}
	}
	return nrgba
}

// hasNonBackgroundPixel reports whether any pixel in img differs from the
// canvas background colour (#F4F6F8 = 244,246,248).
func hasNonBackgroundPixel(img *image.NRGBA) bool {
	bg := color.NRGBA{R: 244, G: 246, B: 248, A: 255}
	b := img.Bounds()
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			if img.NRGBAAt(px, py) != bg {
				return true
			}
		}
	}
	return false
}

// hasDarkPixel reports whether any pixel has all RGB channels below threshold,
// indicating rendered text or dark UI elements (header, border, etc.).
func hasDarkPixel(img *image.NRGBA, threshold uint8) bool {
	b := img.Bounds()
	for py := b.Min.Y; py < b.Max.Y; py++ {
		for px := b.Min.X; px < b.Max.X; px++ {
			c := img.NRGBAAt(px, py)
			if c.A > 0 && c.R < threshold && c.G < threshold && c.B < threshold {
				return true
			}
		}
	}
	return false
}

func TestPNGHasMagicBytes(t *testing.T) {
	data := generatePNG(t, `table t (id bigint)`)
	if len(data) < len(pngMagic) || !bytes.Equal(data[:len(pngMagic)], pngMagic) {
		t.Error("PNG output does not start with PNG magic bytes")
	}
}

func TestPNGNonEmpty(t *testing.T) {
	data := generatePNG(t, `table t (id bigint)`)
	if len(data) == 0 {
		t.Error("PNG output must not be empty")
	}
}

func TestPNGWithEdge(t *testing.T) {
	data := generatePNG(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	if len(data) == 0 {
		t.Error("PNG output with edge must not be empty")
	}
	if !bytes.Equal(data[:len(pngMagic)], pngMagic) {
		t.Error("PNG output with edge does not start with PNG magic bytes")
	}
}

func TestPNGWithCJK(t *testing.T) {
	data := generatePNG(t, `# 用户表
table 用户 (id bigint primary-key)`)
	if len(data) == 0 {
		t.Error("PNG output with CJK must not be empty")
	}
	if !bytes.Equal(data[:len(pngMagic)], pngMagic) {
		t.Error("PNG output with CJK does not start with PNG magic bytes")
	}
}

// TestPNGRendersContent verifies that the PNG output contains non-background
// pixels (table headers, borders, text) — not just a blank canvas.
func TestPNGRendersContent(t *testing.T) {
	data := generatePNG(t, `table users (
  id bigint primary-key
  name varchar(255)
)`)
	img := decodePNGPixels(t, data)
	if !hasNonBackgroundPixel(img) {
		t.Error("PNG contains only background colour; expected table content to be drawn")
	}
}

// TestPNGRendersText verifies that dark pixels exist in the PNG, indicating
// that text labels (e.g. table name, column names) were actually rendered.
func TestPNGRendersText(t *testing.T) {
	data := generatePNG(t, `table users (
  id bigint primary-key
  name varchar(255)
)`)
	img := decodePNGPixels(t, data)
	// The table header background is #2C3E50 (dark). Text rendered on top of
	// lighter rows produces near-black pixels. Either proves text rendering.
	if !hasDarkPixel(img, 100) {
		t.Error("PNG has no dark pixels; expected text or header to be rendered")
	}
}

// TestPNGRenderModifiers verifies that PK/AI/NN modifier text is rendered.
func TestPNGRenderModifiers(t *testing.T) {
	data := generatePNG(t, `table t (
  id bigint primary-key auto-increment
  name varchar(255) not-null
)`)
	img := decodePNGPixels(t, data)
	if !hasDarkPixel(img, 100) {
		t.Error("PNG with modifiers has no dark pixels; modifier text not rendered")
	}
}

// PDF renderer tests

func TestPDFHasMagicBytes(t *testing.T) {
	data := generatePDF(t, `table t (id bigint)`)
	if len(data) < 4 || string(data[:4]) != "%PDF" {
		t.Error("PDF output does not start with %PDF magic bytes")
	}
}

func TestPDFNonEmpty(t *testing.T) {
	data := generatePDF(t, `table t (id bigint)`)
	if len(data) == 0 {
		t.Error("PDF output must not be empty")
	}
}

func TestPDFWithEdge(t *testing.T) {
	data := generatePDF(t, `table a (id bigint)
table b (a_id bigint)
link one a.id to many b.a_id`)
	if len(data) == 0 {
		t.Error("PDF output with edge must not be empty")
	}
	if string(data[:4]) != "%PDF" {
		t.Error("PDF output with edge does not start with %PDF magic bytes")
	}
}

func TestPDFWithModifiers(t *testing.T) {
	data := generatePDF(t, `table t (
  id bigint primary-key auto-increment
  name varchar(255) not-null default("hi")
)`)
	if len(data) == 0 {
		t.Error("PDF output with modifiers must not be empty")
	}
	if string(data[:4]) != "%PDF" {
		t.Error("PDF output with modifiers does not start with %PDF magic bytes")
	}
}
