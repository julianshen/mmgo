package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/julianshen/mmgo/pkg/config"
	svg "github.com/julianshen/mmgo/pkg/output/svg"
	flag "github.com/spf13/pflag"
)

func main() {
	var (
		input      string
		output     string
		theme      string
		bgColor    string
		configFile string
		quiet      bool
	)

	flag.StringVarP(&input, "input", "i", "", "Input file (.mmd) or - for stdin")
	flag.StringVarP(&output, "output", "o", "", "Output file (format inferred from extension; defaults to stdout)")
	flag.StringVarP(&theme, "theme", "t", "", "Mermaid theme (default, dark, forest, neutral)")
	flag.StringVarP(&bgColor, "backgroundColor", "b", "", "Background color")
	flag.StringVarP(&configFile, "configFile", "c", "", "Path to JSON config file")
	flag.BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: mmdc [flags]\n\nRender Mermaid diagrams to SVG.\n\nFlags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if err := run(input, output, theme, bgColor, configFile, quiet); err != nil {
		fmt.Fprintf(os.Stderr, "mmdc: %v\n", err)
		os.Exit(1)
	}
}

func run(input, output, theme, bgColor, configFile string, quiet bool) error {
	if input == "" {
		return fmt.Errorf("no input specified (use -i <file> or -i - for stdin)")
	}

	if output != "" {
		ext := strings.ToLower(filepath.Ext(output))
		if ext != ".svg" && ext != "" {
			return fmt.Errorf("%s output not yet supported (only .svg)", ext)
		}
	}

	var r io.Reader
	if input == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(input)
		if err != nil {
			return fmt.Errorf("open input: %w", err)
		}
		defer f.Close()
		r = f
	}

	opts := &svg.Options{}

	if configFile != "" {
		cfg, err := config.LoadFile(configFile)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		opts.Theme = cfg.Theme
	}

	if theme != "" {
		opts.Theme = config.ThemeName(theme)
	}

	svgBytes, err := svg.Render(r, opts)
	if err != nil {
		return err
	}

	if output == "" {
		_, err = os.Stdout.Write(svgBytes)
		return err
	}

	if err := os.WriteFile(output, svgBytes, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if !quiet {
		fmt.Fprintf(os.Stderr, "Wrote %s\n", output)
	}
	return nil
}
