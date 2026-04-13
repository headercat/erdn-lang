package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/headercat/erdn-lang/internal/ast"
	"github.com/headercat/erdn-lang/internal/output"
	"github.com/headercat/erdn-lang/internal/parser"
	"github.com/headercat/erdn-lang/internal/render"
	"github.com/headercat/erdn-lang/internal/semantic"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "render":
		runRender(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `erdn - erdn-lang schema toolchain

Usage:
  erdn render <schema.erdn> [--out <file>]
  erdn validate <schema.erdn>`)
}

// loadAndValidate parses and semantically validates a schema file.
func loadAndValidate(schemaFile string) (*ast.Program, error) {
	data, err := os.ReadFile(schemaFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", schemaFile, err)
	}
	prog, err := parser.ParseString(string(data))
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	errs := semantic.Validate(prog)
	if len(errs) > 0 {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		return nil, fmt.Errorf("validation errors:\n%s", strings.Join(msgs, "\n"))
	}
	return prog, nil
}

func runRender(args []string) {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	outFlag := fs.String("out", "", "output file path (default: <schema>.svg)")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "render: missing schema file")
		os.Exit(1)
	}
	schemaFile := fs.Arg(0)

	prog, err := loadAndValidate(schemaFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	outPath := *outFlag
	if outPath == "" {
		base := strings.TrimSuffix(schemaFile, filepath.Ext(schemaFile))
		outPath = base + ".svg"
	}

	if err := output.RenderSVG(render.GenerateSVG(prog), outPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("rendered %s\n", outPath)
}

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "validate: missing schema file")
		os.Exit(1)
	}
	schemaFile := fs.Arg(0)

	data, err := os.ReadFile(schemaFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading %s: %v\n", schemaFile, err)
		os.Exit(1)
	}
	prog, err := parser.ParseString(string(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	errs := semantic.Validate(prog)
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, e)
		}
		os.Exit(1)
	}
	fmt.Println("OK")
}
