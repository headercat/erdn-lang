package render

import (
	"fmt"
	"math"
	"strings"

	"github.com/headercat/erdn-lang/internal/ast"
	"github.com/headercat/erdn-lang/internal/parser"
)

// SVG layout constants.
const (
	svgFont      = "sans-serif"
	svgFontSz    = 12.0
	svgHdrFontSz = 13.0
	svgSubFontSz = 10.0
	svgPadH      = 10.0  // horizontal cell padding
	svgPadV      = 6.0   // vertical cell padding
	svgRowH      = 24.0  // data row height  (12 + 2×6)
	svgHdrH      = 26.0  // table header row height
	svgSubHdrH   = 22.0  // sub-header / comment-subtitle row height
	svgTblGapH   = 120.0 // horizontal gap between tables (space for link routing)
	svgTblGapV   = 80.0  // vertical gap between tables
	svgMargin    = 30.0  // outer canvas margin
)

var (
	svgColHeaders = [6]string{"column", "type", "key", "null", "default", "comment"}
	svgColMinW    = [6]float64{80, 70, 44, 44, 60, 80}
)

type svgTableLayout struct {
	tbl       *ast.Table
	x, y      float64
	width     float64
	height    float64
	colWidths [6]float64
	portY     map[string]float64 // column name → absolute y-centre
}

// GenerateSVG converts an AST Program into a self-contained SVG ER diagram.
// No external tools are required. Unicode and CJK text are fully supported.
func GenerateSVG(prog *ast.Program) string {
	layouts := buildSVGLayouts(prog)
	ltMap := make(map[string]*svgTableLayout, len(layouts))
	for _, lt := range layouts {
		ltMap[lt.tbl.Name] = lt
	}

	var cw, ch float64
	for _, lt := range layouts {
		if r := lt.x + lt.width + svgMargin; r > cw {
			cw = r
		}
		if b := lt.y + lt.height + svgMargin; b > ch {
			ch = b
		}
	}
	if cw < 1 {
		cw = 1
	}
	if ch < 1 {
		ch = 1
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	fmt.Fprintf(&sb, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f">`+"\n", cw, ch)
	sb.WriteString(svgDefsBlock())
	fmt.Fprintf(&sb, `<rect width="%.0f" height="%.0f" fill="#F4F6F8"/>`+"\n", cw, ch)

	// Links are drawn first so table boxes appear on top.
	for _, lnk := range prog.Links {
		sb.WriteString(renderSVGLink(lnk, ltMap))
	}
	for _, lt := range layouts {
		sb.WriteString(renderSVGTable(lt))
	}

	sb.WriteString("</svg>\n")
	return sb.String()
}

// buildSVGLayouts computes positions and dimensions for every table node.
func buildSVGLayouts(prog *ast.Program) []*svgTableLayout {
	n := len(prog.Tables)
	if n == 0 {
		return nil
	}

	layouts := make([]*svgTableLayout, n)
	for i, tbl := range prog.Tables {
		layouts[i] = measureSVGTable(tbl)
	}

	// Choose grid column count based on schema size.
	cols := 3
	if n == 1 {
		cols = 1
	} else if n <= 4 {
		cols = 2
	}
	rows := (n + cols - 1) / cols

	maxColW := make([]float64, cols)
	maxRowH := make([]float64, rows)
	for i, lt := range layouts {
		c, r := i%cols, i/cols
		if lt.width > maxColW[c] {
			maxColW[c] = lt.width
		}
		if lt.height > maxRowH[r] {
			maxRowH[r] = lt.height
		}
	}

	colX := make([]float64, cols)
	colX[0] = svgMargin
	for c := 1; c < cols; c++ {
		colX[c] = colX[c-1] + maxColW[c-1] + svgTblGapH
	}
	rowY := make([]float64, rows)
	rowY[0] = svgMargin
	for r := 1; r < rows; r++ {
		rowY[r] = rowY[r-1] + maxRowH[r-1] + svgTblGapV
	}

	for i, lt := range layouts {
		c, r := i%cols, i/cols
		lt.x = colX[c]
		lt.y = rowY[r]

		// Port Y = y-centre of the column's data row (absolute).
		dataTop := lt.y + svgHdrH + svgSubHdrH
		if len(lt.tbl.Comments) > 0 {
			dataTop += svgSubHdrH // comment subtitle row
		}
		lt.portY = make(map[string]float64, len(lt.tbl.Columns))
		for j, col := range lt.tbl.Columns {
			lt.portY[col.Name] = dataTop + float64(j)*svgRowH + svgRowH/2
		}
	}

	return layouts
}

// measureSVGTable computes the pixel dimensions of a single table node.
func measureSVGTable(tbl *ast.Table) *svgTableLayout {
	lt := &svgTableLayout{tbl: tbl}

	// Initialise sub-column widths from header labels.
	for ci, hdr := range svgColHeaders {
		w := svgTextWidth(hdr, svgSubFontSz) + 2*svgPadH
		if w < svgColMinW[ci] {
			w = svgColMinW[ci]
		}
		lt.colWidths[ci] = w
	}

	// Grow each sub-column to fit its data.
	for _, col := range tbl.Columns {
		for ci, txt := range svgCellTexts(col) {
			w := svgTextWidth(txt, svgFontSz) + 2*svgPadH
			if w > lt.colWidths[ci] {
				lt.colWidths[ci] = w
			}
		}
	}

	totalCW := 0.0
	for _, cw := range lt.colWidths {
		totalCW += cw
	}

	// Ensure the table is at least as wide as its header / comment text.
	minW := svgTextWidth(tbl.Name, svgHdrFontSz) + 2*svgPadH
	if len(tbl.Comments) > 0 {
		cw := svgTextWidth(strings.Join(tbl.Comments, " "), svgSubFontSz) + 2*svgPadH
		if cw > minW {
			minW = cw
		}
	}
	lt.width = math.Max(totalCW, minW)
	if lt.width > totalCW {
		lt.colWidths[5] += lt.width - totalCW // expand comment column
	}

	lt.height = svgHdrH + svgSubHdrH + float64(len(tbl.Columns))*svgRowH
	if len(tbl.Comments) > 0 {
		lt.height += svgSubHdrH
	}
	return lt
}

// svgCellTexts returns the text for each of the 6 sub-columns of a column row.
func svgCellTexts(col *ast.Column) [6]string {
	var t [6]string
	t[0] = col.Name
	t[1] = parser.FormatType(col)
	t[2] = svgKeyText(col)
	t[3] = svgNullText(col)
	t[4] = svgDefaultText(col)
	t[5] = strings.Join(col.Comments, " ")
	return t
}

func svgKeyText(col *ast.Column) string {
	var parts []string
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModPrimaryKey:
			parts = append(parts, "PK")
		case ast.ModAutoIncrement:
			parts = append(parts, "AI")
		case ast.ModIndexed:
			parts = append(parts, "IDX")
		}
	}
	return strings.Join(parts, " ")
}

func svgNullText(col *ast.Column) string {
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModNotNull:
			return "NN"
		case ast.ModNullable:
			return "NULL"
		}
	}
	return ""
}

func svgDefaultText(col *ast.Column) string {
	for _, mod := range col.Modifiers {
		if mod.Kind == ast.ModDefault {
			return mod.Value
		}
	}
	return ""
}

// renderSVGTable emits the SVG markup for one table node.
func renderSVGTable(lt *svgTableLayout) string {
	var sb strings.Builder
	x, y, w, h := lt.x, lt.y, lt.width, lt.height

	sb.WriteString("<g>\n")

	// Outer border.
	fmt.Fprintf(&sb, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="none" stroke="#BDC3C7" stroke-width="1"/>`+"\n",
		x, y, w, h)

	curY := y

	// ── Header row ──────────────────────────────────────────────────────────
	fmt.Fprintf(&sb, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="#2C3E50"/>`+"\n",
		x, curY, w, svgHdrH)
	svgWriteText(&sb, x+w/2, curY+svgHdrH/2, "middle", "white", svgHdrFontSz, "bold", "normal", lt.tbl.Name)
	curY += svgHdrH

	// ── Optional table-comment subtitle ─────────────────────────────────────
	if len(lt.tbl.Comments) > 0 {
		comment := strings.Join(lt.tbl.Comments, " ")
		fmt.Fprintf(&sb, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="#34495E"/>`+"\n",
			x, curY, w, svgSubHdrH)
		svgWriteText(&sb, x+w/2, curY+svgSubHdrH/2, "middle", "#BDC3C7", svgSubFontSz, "normal", "italic", comment)
		curY += svgSubHdrH
	}

	// ── Sub-header row ───────────────────────────────────────────────────────
	fmt.Fprintf(&sb, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="#ECF0F1"/>`+"\n",
		x, curY, w, svgSubHdrH)
	cx := x
	for ci, cw := range lt.colWidths {
		if ci > 0 {
			// Vertical divider line that spans from the sub-header down to the bottom.
			fmt.Fprintf(&sb, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#BDC3C7" stroke-width="0.5"/>`+"\n",
				cx, curY, cx, y+h)
		}
		svgWriteText(&sb, cx+cw/2, curY+svgSubHdrH/2, "middle", "#7F8C8D", svgSubFontSz, "bold", "normal", svgColHeaders[ci])
		cx += cw
	}
	curY += svgSubHdrH

	// ── Data rows ────────────────────────────────────────────────────────────
	for ri, col := range lt.tbl.Columns {
		if ri > 0 {
			fmt.Fprintf(&sb, `  <line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f" stroke="#ECF0F1" stroke-width="0.5"/>`+"\n",
				x, curY, x+w, curY)
		}
		rowBG := "white"
		if ri%2 == 1 {
			rowBG = "#F8F9FA"
		}
		fmt.Fprintf(&sb, `  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" fill="%s"/>`+"\n",
			x, curY, w, svgRowH, rowBG)

		texts := svgCellTexts(col)
		cx = x
		for ci, cw := range lt.colWidths {
			txt := texts[ci]
			switch ci {
			case 0:
				svgWriteText(&sb, cx+svgPadH, curY+svgRowH/2, "start", "#2C3E50", svgFontSz, "bold", "normal", txt)
			case 1:
				svgWriteText(&sb, cx+svgPadH, curY+svgRowH/2, "start", "#555555", svgFontSz, "normal", "normal", txt)
			case 2:
				svgWriteKeyCell(&sb, cx+cw/2, curY+svgRowH/2, col)
			case 3:
				svgWriteNullCell(&sb, cx+cw/2, curY+svgRowH/2, col)
			case 4:
				svgWriteText(&sb, cx+svgPadH, curY+svgRowH/2, "start", "#8E44AD", svgFontSz, "normal", "italic", txt)
			case 5:
				svgWriteText(&sb, cx+svgPadH, curY+svgRowH/2, "start", "#7F8C8D", svgFontSz, "normal", "italic", txt)
			}
			cx += cw
		}
		curY += svgRowH
	}

	sb.WriteString("</g>\n")
	return sb.String()
}

func svgWriteKeyCell(sb *strings.Builder, cx, cy float64, col *ast.Column) {
	type badge struct{ text, color string }
	var badges []badge
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModPrimaryKey:
			badges = append(badges, badge{"PK", "#C0392B"})
		case ast.ModAutoIncrement:
			badges = append(badges, badge{"AI", "#2980B9"})
		case ast.ModIndexed:
			badges = append(badges, badge{"IDX", "#27AE60"})
		}
	}
	if len(badges) == 0 {
		return
	}
	fmt.Fprintf(sb, `  <text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="central" font-family="%s" font-size="%.0f" font-weight="bold">`,
		cx, cy, svgFont, svgFontSz)
	for i, b := range badges {
		if i > 0 {
			fmt.Fprintf(sb, `<tspan fill="#aaa"> </tspan>`)
		}
		fmt.Fprintf(sb, `<tspan fill="%s">%s</tspan>`, b.color, svgEscapeText(b.text))
	}
	sb.WriteString("</text>\n")
}

func svgWriteNullCell(sb *strings.Builder, cx, cy float64, col *ast.Column) {
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModNotNull:
			fmt.Fprintf(sb, `  <text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="central" fill="#E67E22" font-family="%s" font-size="%.0f" font-weight="bold">NN</text>`+"\n",
				cx, cy, svgFont, svgFontSz)
			return
		case ast.ModNullable:
			fmt.Fprintf(sb, `  <text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="central" fill="#95A5A6" font-family="%s" font-size="%.0f">NULL</text>`+"\n",
				cx, cy, svgFont, svgFontSz)
			return
		}
	}
}

// renderSVGLink draws an orthogonal (right-angle) connector between two column
// ports. Each connector has three segments: horizontal → vertical → horizontal.
// Link comments are shown as a labelled badge at the midpoint of the connector.
func renderSVGLink(lnk *ast.Link, ltMap map[string]*svgTableLayout) string {
	src := ltMap[lnk.FromTable]
	dst := ltMap[lnk.ToTable]
	if src == nil || dst == nil {
		return ""
	}
	sy, syOK := src.portY[lnk.FromColumn]
	dy, dyOK := dst.portY[lnk.ToColumn]
	if !syOK || !dyOK {
		return ""
	}

	fromLabel := cardLabel(lnk.FromCardinality)
	toLabel := cardLabel(lnk.ToCardinality)

	var sb strings.Builder

	// ── Self-referential link: rectangular loop on the right side ────────────
	if lnk.FromTable == lnk.ToTable {
		sx := src.x + src.width
		loopX := sx + math.Max(40, math.Abs(dy-sy)*0.3+30)
		path := fmt.Sprintf("M %.2f,%.2f H %.2f V %.2f H %.2f", sx, sy, loopX, dy, sx)
		fmt.Fprintf(&sb, `<path d="%s" fill="none" stroke="#95A5A6" stroke-width="1.5" marker-end="url(#arrow)"/>`+"\n", path)
		svgWriteCardLabels(&sb, sx+14, sy, sx+14, dy, fromLabel, toLabel)
		svgWriteLinkComment(&sb, loopX+6, (sy+dy)/2, lnk)
		return sb.String()
	}

	// ── Regular link: H → V → H orthogonal path ─────────────────────────────
	var sx, dx float64
	if src.x+src.width/2 <= dst.x+dst.width/2 {
		sx = src.x + src.width
		dx = dst.x
	} else {
		sx = src.x
		dx = dst.x + dst.width
	}
	midX := (sx + dx) / 2
	path := fmt.Sprintf("M %.2f,%.2f H %.2f V %.2f H %.2f", sx, sy, midX, dy, dx)
	fmt.Fprintf(&sb, `<path d="%s" fill="none" stroke="#95A5A6" stroke-width="1.5" marker-end="url(#arrow)"/>`+"\n", path)

	const labelOff = 14.0
	var flx, tlx float64
	if sx <= dx {
		flx = sx + labelOff
		tlx = dx - labelOff
	} else {
		flx = sx - labelOff
		tlx = dx + labelOff
	}
	svgWriteCardLabels(&sb, flx, sy, tlx, dy, fromLabel, toLabel)

	// Comment badge sits at the midpoint of the vertical segment.
	svgWriteLinkComment(&sb, midX, (sy+dy)/2, lnk)

	return sb.String()
}

// svgWriteCardLabels emits cardinality labels ("1" / "N") near each endpoint.
func svgWriteCardLabels(sb *strings.Builder, flx, fy, tlx, ty float64, fromLabel, toLabel string) {
	fmt.Fprintf(sb, `<text x="%.2f" y="%.2f" text-anchor="middle" fill="#546E7A" font-family="%s" font-size="10" font-weight="bold">%s</text>`+"\n",
		flx, fy-6, svgFont, fromLabel)
	fmt.Fprintf(sb, `<text x="%.2f" y="%.2f" text-anchor="middle" fill="#546E7A" font-family="%s" font-size="10" font-weight="bold">%s</text>`+"\n",
		tlx, ty-6, svgFont, toLabel)
}

// svgWriteLinkComment renders the link's comment text as a pill-shaped badge.
func svgWriteLinkComment(sb *strings.Builder, x, y float64, lnk *ast.Link) {
	if len(lnk.Comments) == 0 {
		return
	}
	comment := strings.Join(lnk.Comments, " ")
	w := svgTextWidth(comment, svgSubFontSz) + 12
	const bh = 16.0
	fmt.Fprintf(sb, `<rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="3" fill="white" stroke="#BDC3C7" stroke-width="0.5"/>`+"\n",
		x-w/2, y-bh/2, w, bh)
	fmt.Fprintf(sb, `<text x="%.2f" y="%.2f" text-anchor="middle" dominant-baseline="central" fill="#546E7A" font-family="%s" font-size="%.0f" font-style="italic">%s</text>`+"\n",
		x, y, svgFont, svgSubFontSz, svgEscapeText(comment))
}

// svgDefsBlock returns the SVG <defs> section (arrowhead marker).
func svgDefsBlock() string {
	return `<defs>
  <marker id="arrow" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">
    <polygon points="0 0, 10 3.5, 0 7" fill="#95A5A6"/>
  </marker>
</defs>` + "\n"
}

// svgWriteText emits a <text> element with the given properties.
func svgWriteText(sb *strings.Builder, x, y float64, anchor, fill string, size float64, weight, style, text string) {
	fmt.Fprintf(sb, `  <text x="%.2f" y="%.2f" text-anchor="%s" dominant-baseline="central" fill="%s" font-family="%s" font-size="%.0f" font-weight="%s" font-style="%s">%s</text>`+"\n",
		x, y, anchor, fill, svgFont, size, weight, style, svgEscapeText(text))
}

// svgEscapeText escapes XML special characters for use in SVG text content.
func svgEscapeText(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// svgTextWidth estimates the rendered pixel width of s at the given font size.
// Wide characters (CJK, fullwidth) are counted at ~0.95× font size;
// all other characters at ~0.55× font size.
func svgTextWidth(s string, size float64) float64 {
	w := 0.0
	for _, r := range s {
		if svgIsWideRune(r) {
			w += size * 0.95
		} else {
			w += size * 0.55
		}
	}
	return w
}

// svgIsWideRune reports whether r occupies double width (CJK, Hangul, etc.).
func svgIsWideRune(r rune) bool {
	return (r >= 0x1100 && r <= 0x115F) || // Hangul Jamo
		(r >= 0x2E80 && r <= 0x303F) || // CJK Radicals / Kangxi
		(r >= 0x3040 && r <= 0xA4CF) || // Kana, CJK Unified Ideographs
		(r >= 0xAC00 && r <= 0xD7AF) || // Hangul Syllables
		(r >= 0xF900 && r <= 0xFAFF) || // CJK Compatibility Ideographs
		(r >= 0xFE10 && r <= 0xFE19) || // Vertical Forms
		(r >= 0xFE30 && r <= 0xFE6F) || // CJK Compatibility Forms
		(r >= 0xFF00 && r <= 0xFF60) || // Fullwidth Latin / Katakana
		(r >= 0xFFE0 && r <= 0xFFE6) // Fullwidth Signs
}
