package render

import (
	"bytes"
	"fmt"
	"image/png"
	"math"

	gofpdf "github.com/go-pdf/fpdf"

	"github.com/headercat/erdn-lang/internal/ast"
)

// pxToPt converts a pixel (at 96 DPI) to PDF points (at 72 DPI).
const pxToPt = 72.0 / 96.0

// GeneratePDF renders prog directly to a PDF document and returns the encoded bytes.
// It rasterizes the SVG to a PNG and embeds it in a page sized to match the image.
func GeneratePDF(prog *ast.Program) ([]byte, error) {
	pngData, err := GeneratePNG(prog)
	if err != nil {
		return nil, fmt.Errorf("rasterizing for PDF: %w", err)
	}
	return pngToPDF(pngData)
}

// pngToPDF wraps a PNG image inside a single-page PDF sized to the image dimensions.
func pngToPDF(pngData []byte) ([]byte, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("decoding PNG config: %w", err)
	}

	wPt := math.Round(float64(cfg.Width)*pxToPt*100) / 100
	hPt := math.Round(float64(cfg.Height)*pxToPt*100) / 100

	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "pt",
		Size:           gofpdf.SizeType{Wd: wPt, Ht: hPt},
	})
	pdf.SetMargins(0, 0, 0)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPageFormat("P", gofpdf.SizeType{Wd: wPt, Ht: hPt})

	pdf.RegisterImageOptionsReader(
		"diagram.png",
		gofpdf.ImageOptions{ImageType: "PNG"},
		bytes.NewReader(pngData),
	)
	pdf.ImageOptions("diagram.png", 0, 0, wPt, hPt, false, gofpdf.ImageOptions{}, 0, "")

	if err := pdf.Error(); err != nil {
		return nil, fmt.Errorf("building PDF: %w", err)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("encoding PDF: %w", err)
	}
	return buf.Bytes(), nil
}
