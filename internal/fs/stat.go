// Package fs contains thin wrappers around os / io primitives the rest of the
// codebase uses. The indirection makes these operations easy to swap out in
// tests and keeps the platform-specific bits (long-path normalisation on
// Windows, EXDEV fallback on Linux) in one place.
package fs

import (
	"os"
	"time"
)

// MTime returns the modification time of path in UTC, truncated to whole
// seconds. Truncation matches the resolution we want across filesystems
// (ext4, NTFS, FAT, macOS APFS) and guarantees a stable ordering key for the
// canonical-selection stage.
func MTime(path string) (time.Time, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime().UTC().Truncate(time.Second), nil
}
