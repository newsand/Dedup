package compare

import (
	"context"

	"deduplicator/internal/hash"
	"deduplicator/internal/model"
)

// ImageComparator computes a BLAKE3 key for image files.
type ImageComparator struct{}

func NewImageComparator() *ImageComparator { return &ImageComparator{} }

func (ImageComparator) Supports() []model.FileType { return []model.FileType{model.FileTypeImage} }

func (ImageComparator) Key(ctx context.Context, f model.DiscoveredFile) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	hex, _, err := hash.FileBLAKE3(f.AbsolutePath)
	return hex, err
}
