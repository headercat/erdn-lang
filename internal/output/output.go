package output

import "os"

// RenderSVG writes SVG content directly to outPath.
func RenderSVG(svgContent, outPath string) error {
	return os.WriteFile(outPath, []byte(svgContent), 0644)
}

// RenderDOT writes DOT content directly to outPath.
func RenderDOT(dotContent, outPath string) error {
	return os.WriteFile(outPath, []byte(dotContent), 0644)
}
