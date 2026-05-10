package math

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-latex/latex/font/ttf"
	"golang.org/x/image/font/sfnt"
)

const (
	notoMathURL = "https://github.com/notofonts/math/releases/download/NotoSansMath-v3.000/NotoSansMath-v3.000.zip"
	notoMathZip = "NotoSansMath-v3.000.zip"
	notoMathTTF = "NotoSansMath/full/ttf/NotoSansMath-Regular.ttf"
)

var (
	mathFontsOnce sync.Once
	mathFonts     *ttf.Fonts
	mathFontsErr  error
)

// MathFonts returns a *ttf.Fonts populated with Noto Sans Math.
// On the first call it downloads the font from the Noto fonts GitHub
// release, caches it in the user's cache directory, and parses it.
// Subsequent calls return the cached value.
func MathFonts() (*ttf.Fonts, error) {
	mathFontsOnce.Do(func() {
		mathFonts, mathFontsErr = loadMathFonts()
	})
	return mathFonts, mathFontsErr
}

func loadMathFonts() (*ttf.Fonts, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	cacheDir = filepath.Join(cacheDir, "mmgo")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("math font: create cache dir: %w", err)
	}

	fontPath := filepath.Join(cacheDir, "NotoSansMath-Regular.ttf")

	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		if err := downloadAndExtract(cacheDir); err != nil {
			return nil, fmt.Errorf("math font: download: %w", err)
		}
	}

	data, err := os.ReadFile(fontPath)
	if err != nil {
		return nil, fmt.Errorf("math font: read: %w", err)
	}

	ft, err := sfnt.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("math font: parse: %w", err)
	}

	// Noto Sans Math is a single font file; use it for all styles.
	fonts := &ttf.Fonts{
		Default: ft,
		Rm:      ft,
		It:      ft,
		Bf:      ft,
		BfIt:    ft,
	}
	return fonts, nil
}

func downloadAndExtract(cacheDir string) error {
	zipPath := filepath.Join(cacheDir, notoMathZip)

	// Download the zip.
	resp, err := http.Get(notoMathURL)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", notoMathURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: status %d", notoMathURL, resp.StatusCode)
	}

	f, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return fmt.Errorf("write zip: %w", err)
	}
	f.Close()

	// Extract the TTF.
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()

	for _, zf := range zr.File {
		if zf.Name != notoMathTTF {
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return fmt.Errorf("open %s in zip: %w", zf.Name, err)
		}
		defer rc.Close()

		outPath := filepath.Join(cacheDir, "NotoSansMath-Regular.ttf")
		out, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("create ttf: %w", err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			return fmt.Errorf("write ttf: %w", err)
		}
		out.Close()
		return nil
	}

	return fmt.Errorf("font %s not found in zip", notoMathTTF)
}
