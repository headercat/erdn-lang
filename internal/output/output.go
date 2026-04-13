package output

import "os"

// RenderSVG writes SVG content directly to outPath.
func RenderSVG(svgContent, outPath string) error {
	return os.WriteFile(outPath, []byte(svgContent), 0644)
}
