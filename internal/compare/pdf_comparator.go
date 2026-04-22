package compare

import (
	"context"

	"deduplicator/internal/hash"
	"deduplicator/internal/model"
)

// PDFComparator computes a BLAKE3 key for PDF files.
type PDFComparator struct{}

func NewPDFComparator() *PDFComparator { return &PDFComparator{} }

func (PDFComparator) Supports() []model.FileType { return []model.FileType{model.FileTypePDF} }

func (PDFComparator) Key(ctx context.Context, f model.DiscoveredFile) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	hex, _, err := hash.FileBLAKE3(f.AbsolutePath)
	return hex, err
}
