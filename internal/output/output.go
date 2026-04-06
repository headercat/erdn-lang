package output

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RenderPNG writes the DOT content to outPath as a PNG using Graphviz dot.
func RenderPNG(dotContent, outPath string) error {
	return renderWithFormat(dotContent, outPath, "png")
}

// RenderPDF writes the DOT content to outPath as a PDF using Graphviz dot.
func RenderPDF(dotContent, outPath string) error {
	return renderWithFormat(dotContent, outPath, "pdf")
}

// RenderDOT writes the DOT content directly to outPath.
func RenderDOT(dotContent, outPath string) error {
	return os.WriteFile(outPath, []byte(dotContent), 0644)
}

func renderWithFormat(dotContent, outPath, format string) error {
	if _, err := exec.LookPath("dot"); err != nil {
		return fmt.Errorf("graphviz 'dot' command not found: install graphviz to render images")
	}
	cmd := exec.Command("dot", fmt.Sprintf("-T%s", format), "-o", outPath)
	cmd.Stdin = strings.NewReader(dotContent)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("dot command failed: %w\n%s", err, string(out))
	}
	return nil
}
