//go:build js && wasm

// Command wasm is the WASM entry-point that exposes ERDN compilation to JavaScript.
// It registers a global "compileToSVG" function that accepts an ERDN source string
// and returns either the rendered SVG or an error message.
package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/headercat/erdn-lang/internal/parser"
	"github.com/headercat/erdn-lang/internal/render"
	"github.com/headercat/erdn-lang/internal/semantic"
)

func compileToSVG(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"error": "compileToSVG: missing source argument",
			"svg":   "",
		}
	}

	src := args[0].String()
	if strings.TrimSpace(src) == "" {
		return map[string]interface{}{
			"error": "",
			"svg":   "",
		}
	}

	prog, err := parser.ParseString(src)
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("Parse error: %v", err),
			"svg":   "",
		}
	}

	errs := semantic.Validate(prog)
	if len(errs) > 0 {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		return map[string]interface{}{
			"error": strings.Join(msgs, "\n"),
			"svg":   "",
		}
	}

	svg := render.GenerateSVG(prog)
	return map[string]interface{}{
		"error": "",
		"svg":   svg,
	}
}

func main() {
	js.Global().Set("compileToSVG", js.FuncOf(compileToSVG))

	// Block forever so the Go runtime stays alive for JS callbacks.
	select {}
}
