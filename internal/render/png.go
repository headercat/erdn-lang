package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strings"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"

	"github.com/headercat/erdn-lang/internal/ast"
)

// pngDPI is the pixel density used when rasterizing SVG user units to pixels.
// SVG user units are defined at 96 dpi, so 1 unit = 1 px at pngDPI=96.
const pngDPI = 96.0

// GeneratePNG renders prog directly to a PNG image and returns the encoded bytes.
func GeneratePNG(prog *ast.Program) ([]byte, error) {
	return svgToPNG(GenerateSVG(prog))
}

// svgToPNG rasterizes svgContent to a PNG image at 96 DPI (1:1 px mapping).
func svgToPNG(svgContent string) ([]byte, error) {
	icon, err := oksvg.ReadIconStream(strings.NewReader(svgContent), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, fmt.Errorf("parsing SVG: %w", err)
	}

	w := int(math.Ceil(icon.ViewBox.W))
	h := int(math.Ceil(icon.ViewBox.H))
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}

	icon.SetTarget(0, 0, float64(w), float64(h))

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	// Fill with white background before rasterizing.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Draw(dasher, 1.0)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}
