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
	mathFontsMu  sync.Mutex
	mathFonts    *ttf.Fonts
	mathFontsErr error
)

// MathFonts returns a *ttf.Fonts populated with Noto Sans Math.
// On the first call it downloads the font from the Noto fonts GitHub
// release, caches it in the user's cache directory, and parses it.
// Subsequent successful calls return the cached value; if a previous
// call failed (e.g. transient network outage) the next call retries
// rather than serving the cached error forever.
func MathFonts() (*ttf.Fonts, error) {
	mathFontsMu.Lock()
	defer mathFontsMu.Unlock()
	if mathFonts != nil {
		return mathFonts, nil
	}
	mathFonts, mathFontsErr = loadMathFonts()
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
		// Cached TTF is truncated/corrupted (e.g. interrupted download).
		// Wipe it and try one fresh download before giving up.
		_ = os.Remove(fontPath)
		if dlErr := downloadAndExtract(cacheDir); dlErr != nil {
			return nil, fmt.Errorf("math font: re-download after parse error %v: %w", err, dlErr)
		}
		data, err = os.ReadFile(fontPath)
		if err != nil {
			return nil, fmt.Errorf("math font: re-read: %w", err)
		}
		ft, err = sfnt.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("math font: parse: %w", err)
		}
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

	if err := writeZip(zipPath, resp.Body); err != nil {
		return err
	}

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
		if err := writeFile(outPath, rc); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("font %s not found in zip", notoMathTTF)
}

// writeZip writes r to path, making sure both the copy error and the
// close error propagate via deferred close.
func writeZip(path string, r io.Reader) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close zip: %w", closeErr)
		}
	}()
	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("write zip: %w", err)
	}
	return nil
}

// writeFile is writeZip's sibling for the extracted TTF.
func writeFile(path string, r io.Reader) (err error) {
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create ttf: %w", err)
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("close ttf: %w", closeErr)
		}
	}()
	if _, err := io.Copy(out, r); err != nil {
		return fmt.Errorf("write ttf: %w", err)
	}
	return nil
}
