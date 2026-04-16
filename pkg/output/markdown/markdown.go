// Package markdown rewrites Mermaid code blocks in markdown files,
// replacing them with rendered image references.
package markdown

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/julianshen/mmgo/pkg/output"
	svgpkg "github.com/julianshen/mmgo/pkg/output/svg"
)

func Process(r io.Reader, outputDir, baseName, format string, opts *svgpkg.Options) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	var out strings.Builder
	var mermaidBuf strings.Builder
	inMermaid := false
	blockNum := 0

	for scanner.Scan() {
		line := scanner.Text()

		if inMermaid {
			if line == "```" {
				blockNum++
				content := mermaidBuf.String()
				mermaidBuf.Reset()
				inMermaid = false

				filename := fmt.Sprintf("%s-%d.%s", baseName, blockNum, format)
				if err := renderAndWrite(content, filepath.Join(outputDir, filename), format, opts); err != nil {
					return nil, fmt.Errorf("block %d: %w", blockNum, err)
				}
				fmt.Fprintf(&out, "![](%s)\n", filename)
				continue
			}
			mermaidBuf.WriteString(line)
			mermaidBuf.WriteByte('\n')
			continue
		}

		if strings.TrimSpace(line) == "```mermaid" {
			inMermaid = true
			mermaidBuf.Reset()
			continue
		}

		out.WriteString(line)
		out.WriteByte('\n')
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("markdown: read: %w", err)
	}
	if inMermaid {
		return nil, fmt.Errorf("markdown: unclosed mermaid code block")
	}
	return []byte(out.String()), nil
}

func renderAndWrite(mermaid, path, format string, opts *svgpkg.Options) error {
	svgBytes, err := svgpkg.Render(strings.NewReader(mermaid), opts)
	if err != nil {
		return err
	}
	outBytes, err := output.ConvertSVG(svgBytes, format)
	if err != nil {
		return err
	}
	return os.WriteFile(path, outBytes, 0o644)
}
