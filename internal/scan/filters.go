package scan

import (
	"path/filepath"
	"strings"
)

// Filters decides which paths the walker keeps.
//
// Include is the allow list; Exclude is the deny list. Both accept Go's
// filepath.Match glob patterns and are evaluated against:
//   - the full absolute path, and
//   - the basename.
//
// Matching is case-insensitive on Windows (the caller is expected to have
// normalised patterns to lowercase; see NewFilters).
type Filters struct {
	Include []string
	Exclude []string
}

// NewFilters returns a Filters with patterns trimmed of whitespace. Empty
// patterns are dropped.
func NewFilters(include, exclude []string) Filters {
	return Filters{
		Include: clean(include),
		Exclude: clean(exclude),
	}
}

func clean(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// AllowPath returns true when the file at absPath satisfies the include list
// (or the include list is empty) and does not match the exclude list.
func (f Filters) AllowPath(absPath string) bool {
	base := filepath.Base(absPath)
	if len(f.Exclude) > 0 {
		for _, p := range f.Exclude {
			if matchAny(p, absPath, base) {
				return false
			}
		}
	}
	if len(f.Include) == 0 {
		return true
	}
	for _, p := range f.Include {
		if matchAny(p, absPath, base) {
			return true
		}
	}
	return false
}

func matchAny(pattern, abs, base string) bool {
	if ok, _ := filepath.Match(pattern, abs); ok {
		return true
	}
	if ok, _ := filepath.Match(pattern, base); ok {
		return true
	}
	return false
}
