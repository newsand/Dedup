package filetype

import (
	"bytes"
	"io"
	"os"

	"deduplicator/internal/model"
)

// sniffBytes is the number of bytes inspected when the extension alone is
// inconclusive (e.g. unusual extensions on correct content).
const sniffBytes = 512

// Detect returns the FileType for a path. The extension is always tried first
// — it is cheap and correct in the vast majority of real-world cases. If the
// extension is unknown, Detect sniffs up to sniffBytes from the file and maps
// known magic bytes to a type.
func Detect(path, ext string) model.FileType {
	if ft := ClassifyExtension(ext); ft != model.FileTypeUnknown {
		return ft
	}
	return sniff(path)
}

// sniff is the file-content fallback. Errors are swallowed and treated as
// FileTypeUnknown; we do not want scanning to abort because one file is
// unreadable.
func sniff(path string) model.FileType {
	f, err := os.Open(path)
	if err != nil {
		return model.FileTypeUnknown
	}
	defer f.Close()

	buf := make([]byte, sniffBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return model.FileTypeUnknown
	}
	return magic(buf[:n])
}

// magic maps leading bytes to a FileType. We only encode the byte patterns
// that correspond to the extensions we already ship; exotic formats stay
// FileTypeUnknown on purpose.
func magic(b []byte) model.FileType {
	switch {
	case bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}):
		return model.FileTypeImage // JPEG
	case bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return model.FileTypeImage // PNG
	case bytes.HasPrefix(b, []byte{'G', 'I', 'F', '8'}):
		return model.FileTypeImage // GIF
	case bytes.HasPrefix(b, []byte{'B', 'M'}):
		return model.FileTypeImage // BMP
	case len(b) >= 12 && bytes.Equal(b[0:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return model.FileTypeImage // WEBP
	case bytes.HasPrefix(b, []byte{'I', 'I', 0x2A, 0x00}),
		bytes.HasPrefix(b, []byte{'M', 'M', 0x00, 0x2A}):
		return model.FileTypeImage // TIFF
	case bytes.HasPrefix(b, []byte{'%', 'P', 'D', 'F', '-'}):
		return model.FileTypePDF
	}
	return model.FileTypeUnknown
}
