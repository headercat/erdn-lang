package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/headercat/erdn-lang/internal/ast"
)

var (
	pngFontOnce    sync.Once
	pngRegularFont *opentype.Font
	pngBoldFont    *opentype.Font
)

func initPNGFonts() {
	pngFontOnce.Do(func() {
		pngRegularFont, _ = opentype.Parse(goregular.TTF)
		pngBoldFont, _ = opentype.Parse(gobold.TTF)
	})
}

func pngFontFace(bold bool, size float64) font.Face {
	initPNGFonts()
	f := pngRegularFont
	if bold {
		f = pngBoldFont
	}
	if f == nil {
		return nil
	}
	face, _ := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 96})
	return face
}

// GeneratePNG renders prog directly to a PNG image and returns the encoded bytes.
// Text is rendered using embedded Go fonts; no external tools are required.
func GeneratePNG(prog *ast.Program) ([]byte, error) {
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

	iw, ih := int(math.Ceil(cw)), int(math.Ceil(ch))
	img := image.NewNRGBA(image.Rect(0, 0, iw, ih))
	pngFillRect(img, 0, 0, float64(iw), float64(ih), pngHexColor("#F4F6F8"))

	// Links are drawn first so table boxes appear on top.
	for _, lnk := range prog.Links {
		pngRenderLink(img, lnk, ltMap)
	}
	for _, lt := range layouts {
		pngRenderTable(img, lt)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}

func pngRenderTable(img *image.NRGBA, lt *svgTableLayout) {
	x, y, w, h := lt.x, lt.y, lt.width, lt.height

	// White background, then outer border.
	pngFillRect(img, x, y, w, h, color.NRGBA{255, 255, 255, 255})
	pngStrokeRect(img, x, y, w, h, pngHexColor("#BDC3C7"))

	curY := y

	// ── Header row ──────────────────────────────────────────────────────────
	pngFillRect(img, x, curY, w, svgHdrH, pngHexColor("#2C3E50"))
	pngDrawText(img, x+w/2, curY+svgHdrH/2, "middle", color.NRGBA{255, 255, 255, 255}, svgHdrFontSz, true, lt.tbl.Name)
	curY += svgHdrH

	// ── Per-line table-comment subtitle rows ─────────────────────────────────
	for _, comment := range lt.tbl.Comments {
		pngFillRect(img, x, curY, w, svgCommentH, pngHexColor("#34495E"))
		pngDrawText(img, x+w/2, curY+svgCommentH/2, "middle", pngHexColor("#BDC3C7"), svgSubFontSz, false, comment)
		curY += svgCommentH
	}

	// ── Sub-header row ───────────────────────────────────────────────────────
	pngFillRect(img, x, curY, w, svgSubHdrH, pngHexColor("#ECF0F1"))
	cx := x
	for ci, cw := range lt.colWidths {
		if ci > 0 {
			pngDrawVLine(img, cx, curY, y+h, pngHexColor("#BDC3C7"))
		}
		pngDrawText(img, cx+cw/2, curY+svgSubHdrH/2, "middle", pngHexColor("#7F8C8D"), svgSubFontSz, true, svgColHeaders[ci])
		cx += cw
	}
	curY += svgSubHdrH

	// ── Data rows ────────────────────────────────────────────────────────────
	for ri, col := range lt.tbl.Columns {
		if ri > 0 {
			pngDrawHLine(img, x, x+w, curY, pngHexColor("#ECF0F1"))
		}
		rowBG := color.NRGBA{255, 255, 255, 255}
		if ri%2 == 1 {
			rowBG = pngHexColor("#F8F9FA")
		}
		pngFillRect(img, x, curY, w, svgRowH, rowBG)

		texts := svgCellTexts(col)
		cx = x
		for ci, cw := range lt.colWidths {
			txt := texts[ci]
			switch ci {
			case 0:
				pngDrawText(img, cx+svgPadH, curY+svgRowH/2, "start", pngHexColor("#2C3E50"), svgFontSz, true, txt)
			case 1:
				pngDrawText(img, cx+svgPadH, curY+svgRowH/2, "start", pngHexColor("#555555"), svgFontSz, false, txt)
			case 2:
				pngRenderKeyCell(img, cx+cw/2, curY+svgRowH/2, col)
			case 3:
				pngRenderNullCell(img, cx+cw/2, curY+svgRowH/2, col)
			case 4:
				pngDrawText(img, cx+svgPadH, curY+svgRowH/2, "start", pngHexColor("#8E44AD"), svgFontSz, false, txt)
			case 5:
				pngDrawText(img, cx+svgPadH, curY+svgRowH/2, "start", pngHexColor("#7F8C8D"), svgFontSz, false, txt)
			}
			cx += cw
		}
		curY += svgRowH
	}
}

// pngRenderKeyCell draws PK / AI / IDX badges each in their own colour.
func pngRenderKeyCell(img *image.NRGBA, cx, cy float64, col *ast.Column) {
	type badge struct {
		text string
		clr  color.NRGBA
	}
	var badges []badge
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModPrimaryKey:
			badges = append(badges, badge{"PK", pngHexColor("#C0392B")})
		case ast.ModAutoIncrement:
			badges = append(badges, badge{"AI", pngHexColor("#2980B9")})
		case ast.ModIndexed:
			badges = append(badges, badge{"IDX", pngHexColor("#27AE60")})
		}
	}
	if len(badges) == 0 {
		return
	}
	face := pngFontFace(true, svgFontSz)
	if face == nil {
		return
	}
	defer face.Close()

	const gap = 4.0
	totalAdv := fixed.Int26_6(0)
	for i, b := range badges {
		if i > 0 {
			totalAdv += fixed.I(int(gap))
		}
		totalAdv += (&font.Drawer{Face: face}).MeasureString(b.text)
	}

	metrics := face.Metrics()
	baselineY := fixed.Int26_6(cy*64) + (metrics.Ascent-metrics.Descent)/2
	penX := fixed.Int26_6(cx*64) - totalAdv/2
	for i, b := range badges {
		if i > 0 {
			penX += fixed.I(int(gap))
		}
		adv := (&font.Drawer{Face: face}).MeasureString(b.text)
		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(b.clr),
			Face: face,
			Dot:  fixed.Point26_6{X: penX, Y: baselineY},
		}
		d.DrawString(b.text)
		penX += adv
	}
}

func pngRenderNullCell(img *image.NRGBA, cx, cy float64, col *ast.Column) {
	for _, mod := range col.Modifiers {
		switch mod.Kind {
		case ast.ModNotNull:
			pngDrawText(img, cx, cy, "middle", pngHexColor("#E67E22"), svgFontSz, true, "NN")
			return
		case ast.ModNullable:
			pngDrawText(img, cx, cy, "middle", pngHexColor("#95A5A6"), svgFontSz, false, "NULL")
			return
		}
	}
}

func pngRenderLink(img *image.NRGBA, lnk *ast.Link, ltMap map[string]*svgTableLayout) {
	src := ltMap[lnk.FromTable]
	dst := ltMap[lnk.ToTable]
	if src == nil || dst == nil {
		return
	}
	sy, syOK := src.portY[lnk.FromColumn]
	dy, dyOK := dst.portY[lnk.ToColumn]
	if !syOK || !dyOK {
		return
	}

	lineC := pngHexColor("#95A5A6")
	fromLbl := cardLabel(lnk.FromCardinality)
	toLbl := cardLabel(lnk.ToCardinality)

	// ── Self-referential link: rectangular loop on the right side ────────────
	if lnk.FromTable == lnk.ToTable {
		sx := src.x + src.width
		loopOffset := math.Max(50, math.Abs(dy-sy)*0.3+35)
		loopX := sx + loopOffset
		pngDrawHLine(img, sx, loopX, sy, lineC)
		pngDrawVLine(img, loopX, sy, dy, lineC)
		pngDrawHLine(img, sx, loopX, dy, lineC)
		pngDrawText(img, sx+14, sy-6, "middle", pngHexColor("#546E7A"), 10, true, fromLbl)
		pngDrawText(img, sx+14, dy-6, "middle", pngHexColor("#546E7A"), 10, true, toLbl)
		if len(lnk.Comments) > 0 {
			pngDrawText(img, loopX, (sy+dy)/2, "middle", pngHexColor("#546E7A"), svgSubFontSz, false, strings.Join(lnk.Comments, " "))
		}
		return
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
	pngDrawHLine(img, sx, midX, sy, lineC)
	pngDrawVLine(img, midX, sy, dy, lineC)
	pngDrawHLine(img, midX, dx, dy, lineC)

	const labelOff = 14.0
	var flx, tlx float64
	if sx <= dx {
		flx = sx + labelOff
		tlx = dx - labelOff
	} else {
		flx = sx - labelOff
		tlx = dx + labelOff
	}
	pngDrawText(img, flx, sy-6, "middle", pngHexColor("#546E7A"), 10, true, fromLbl)
	pngDrawText(img, tlx, dy-6, "middle", pngHexColor("#546E7A"), 10, true, toLbl)

	if len(lnk.Comments) > 0 {
		const bh = 16.0
		commentY := (sy + dy) / 2
		if math.Abs(sy-dy) < 2*bh {
			commentY = math.Min(sy, dy) - bh - 4
		}
		pngDrawText(img, midX, commentY, "middle", pngHexColor("#546E7A"), svgSubFontSz, false, strings.Join(lnk.Comments, " "))
	}
}

// pngDrawText draws text on img.
// anchor: "start", "middle", or "end".
// y is the vertical centre (dominant-baseline="central").
func pngDrawText(img *image.NRGBA, x, y float64, anchor string, clr color.NRGBA, size float64, bold bool, text string) {
	if text == "" {
		return
	}
	face := pngFontFace(bold, size)
	if face == nil {
		return
	}
	defer face.Close()

	d := &font.Drawer{Dst: img, Src: image.NewUniform(clr), Face: face}

	textW := d.MeasureString(text)
	fx := fixed.Int26_6(x * 64)
	switch anchor {
	case "middle":
		fx -= textW / 2
	case "end":
		fx -= textW
	}

	metrics := face.Metrics()
	fy := fixed.Int26_6(y*64) + (metrics.Ascent-metrics.Descent)/2

	d.Dot = fixed.Point26_6{X: fx, Y: fy}
	d.DrawString(text)
}

func pngFillRect(img *image.NRGBA, x, y, w, h float64, c color.NRGBA) {
	x0 := int(math.Round(x))
	y0 := int(math.Round(y))
	x1 := int(math.Round(x + w))
	y1 := int(math.Round(y + h))
	b := img.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			img.SetNRGBA(px, py, c)
		}
	}
}

func pngStrokeRect(img *image.NRGBA, x, y, w, h float64, c color.NRGBA) {
	x1, y1 := x+w, y+h
	pngDrawHLine(img, x, x1, y, c)
	pngDrawHLine(img, x, x1, y1, c)
	pngDrawVLine(img, x, y, y1, c)
	pngDrawVLine(img, x1, y, y1, c)
}

func pngDrawHLine(img *image.NRGBA, x0, x1, y float64, c color.NRGBA) {
	iy := int(math.Round(y))
	ix0 := int(math.Round(x0))
	ix1 := int(math.Round(x1))
	if ix0 > ix1 {
		ix0, ix1 = ix1, ix0
	}
	b := img.Bounds()
	if iy < b.Min.Y || iy >= b.Max.Y {
		return
	}
	if ix0 < b.Min.X {
		ix0 = b.Min.X
	}
	if ix1 >= b.Max.X {
		ix1 = b.Max.X - 1
	}
	for px := ix0; px <= ix1; px++ {
		img.SetNRGBA(px, iy, c)
	}
}

func pngDrawVLine(img *image.NRGBA, x, y0, y1 float64, c color.NRGBA) {
	ix := int(math.Round(x))
	iy0 := int(math.Round(y0))
	iy1 := int(math.Round(y1))
	if iy0 > iy1 {
		iy0, iy1 = iy1, iy0
	}
	b := img.Bounds()
	if ix < b.Min.X || ix >= b.Max.X {
		return
	}
	if iy0 < b.Min.Y {
		iy0 = b.Min.Y
	}
	if iy1 >= b.Max.Y {
		iy1 = b.Max.Y - 1
	}
	for py := iy0; py <= iy1; py++ {
		img.SetNRGBA(ix, py, c)
	}
}

// pngHexColor parses a 6-digit hex colour string (e.g. "#2C3E50") into color.NRGBA.
func pngHexColor(hex string) color.NRGBA {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return color.NRGBA{A: 255}
	}
	return color.NRGBA{
		R: pngHexByte(hex[0], hex[1]),
		G: pngHexByte(hex[2], hex[3]),
		B: pngHexByte(hex[4], hex[5]),
		A: 255,
	}
}

func pngHexByte(hi, lo byte) uint8 {
	return pngHexNibble(hi)<<4 | pngHexNibble(lo)
}

func pngHexNibble(c byte) uint8 {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}
