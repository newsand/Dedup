// Package naming produces flattened, portable file names for the copy-unique
// action.
//
// The transformation is a deterministic pure function on (relative path,
// rules, peers) — same input, same output. Collisions are resolved by
// appending an incremental `_N` suffix in lexicographic order.
package naming

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// ToSnake converts s to a conservative snake_case form:
//
//   1. lowercase;
//   2. Unicode NFD, then drop combining marks (diacritics);
//   3. replace every rune that is not [a-z0-9] with '_';
//   4. collapse repeated '_';
//   5. trim leading/trailing '_'.
//
// The output is safe on both POSIX and Windows filesystems and on typical
// CLI usage (no spaces, no shell metacharacters).
func ToSnake(s string) string {
	s = strings.ToLower(s)
	// NFD so that "é" becomes "e" + combining acute; we then drop the
	// combining mark and keep the base letter.
	decomposed := norm.NFD.String(s)

	var b strings.Builder
	b.Grow(len(decomposed))
	lastUnderscore := true // suppress leading underscore
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := b.String()
	return strings.Trim(out, "_")
}
