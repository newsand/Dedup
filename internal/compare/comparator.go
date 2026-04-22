// Package compare turns a stream of DiscoveredFile entries into
// DuplicateGroups and uniques, routing each file through a Comparator chosen
// by FileType.
//
// The MVP uses BLAKE3 on the binary content for every supported file type.
// Splitting images and PDFs into their own Comparator implementations is
// deliberate: it keeps the extension points open for v2.0 (visual similarity,
// PDF-text compare, etc.) without changing the pipeline or the CLI.
package compare

import (
	"context"

	"deduplicator/internal/model"
)

// Comparator computes a duplicity key for a supported FileType.
//
// Implementations must:
//   - be deterministic: same content -> same key;
//   - be independent of file name / path / mtime;
//   - not mutate the filesystem.
type Comparator interface {
	Supports() []model.FileType
	Key(ctx context.Context, f model.DiscoveredFile) (string, error)
}

// Comparable pairs a file with the key produced by its Comparator.
type Comparable struct {
	File model.DiscoveredFile
	Key  string
}
