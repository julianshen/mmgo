package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/julianshen/mmgo/pkg/config"
	"github.com/julianshen/mmgo/pkg/output"
	svg "github.com/julianshen/mmgo/pkg/output/svg"
	flag "github.com/spf13/pflag"
)

func main() {
	var opts cliOptions

	flag.StringVarP(&opts.Input, "input", "i", "", "Input file (.mmd) or - for stdin")
	flag.StringVarP(&opts.Output, "output", "o", "", "Output file (format inferred from extension; defaults to stdout)")
	flag.StringVarP(&opts.Theme, "theme", "t", "", "Mermaid theme (default, dark, forest, neutral)")
	flag.StringVarP(&opts.BackgroundColor, "backgroundColor", "b", "", "Background color")
	flag.StringVarP(&opts.ConfigFile, "configFile", "c", "", "Path to JSON config file")
	flag.BoolVarP(&opts.Quiet, "quiet", "q", false, "Suppress non-error output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mmgo [flags]\n\nRender Mermaid diagrams to SVG.\n\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "mmgo: %v\n", err)
		os.Exit(1)
	}
}

type cliOptions struct {
	Input           string
	Output          string
	Theme           string
	BackgroundColor string
	ConfigFile      string
	Quiet           bool
}

func run(opts cliOptions) error {
	if opts.Input == "" {
		return fmt.Errorf("no input specified (use -i <file> or -i - for stdin)")
	}

	outputFormat := "svg"
	if opts.Output != "" {
		ext := strings.ToLower(filepath.Ext(opts.Output))
		switch ext {
		case ".svg", "":
			outputFormat = "svg"
		case ".png":
			outputFormat = "png"
		case ".pdf":
			outputFormat = "pdf"
		default:
			return fmt.Errorf("%s output not yet supported", ext)
		}
	}

	var r io.Reader
	if opts.Input == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(opts.Input)
		if err != nil {
			return fmt.Errorf("open input: %w", err)
		}
		defer func() { _ = f.Close() }()
		r = f
	}

	svgOpts := &svg.Options{}

	if opts.ConfigFile != "" {
		cfg, err := config.LoadFile(opts.ConfigFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		svgOpts.Theme = cfg.Theme
	}

	if opts.Theme != "" {
		svgOpts.Theme = config.ThemeName(opts.Theme)
	}

	svgBytes, err := svg.Render(r, svgOpts)
	if err != nil {
		return err
	}

	outBytes, err := output.ConvertSVG(svgBytes, outputFormat)
	if err != nil {
		return err
	}

	if opts.Output == "" {
		_, err = os.Stdout.Write(outBytes)
		return err
	}

	if err := os.WriteFile(opts.Output, outBytes, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if !opts.Quiet {
		fmt.Fprintf(os.Stderr, "Wrote %s\n", opts.Output)
	}
	return nil
}
