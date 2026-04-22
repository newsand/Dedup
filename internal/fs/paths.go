package fs

import (
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizeWindows canonicalises a path on Windows:
//   - backslashes for the separator;
//   - absolute;
//   - clean.
//
// On non-Windows systems the path is passed through filepath.Clean.
func NormalizeWindows(p string) string {
	if runtime.GOOS == "windows" {
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		return filepath.Clean(p)
	}
	return filepath.Clean(p)
}

// LongPathWin prefixes p with `\\?\` when running on Windows and the path is
// long enough to risk exceeding MAX_PATH. On other systems it returns p as-is.
func LongPathWin(p string) string {
	if runtime.GOOS != "windows" {
		return p
	}
	if len(p) < 248 {
		return p
	}
	if strings.HasPrefix(p, `\\?\`) {
		return p
	}
	return `\\?\` + filepath.Clean(p)
}
