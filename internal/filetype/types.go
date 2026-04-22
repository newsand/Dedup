// Package filetype classifies discovered files into the FileType categories
// understood by the pipeline (image, pdf, unknown).
//
// v1.0 recognises a small set of extensions and, for ambiguous cases, peeks
// the first 512 bytes of the file to confirm the content type.
package filetype

import "deduplicator/internal/model"

// imageExtensions is the authoritative list of image extensions recognised by
// v1.0. All entries are lowercase and include the leading dot.
var imageExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".bmp":  {},
	".tif":  {},
	".tiff": {},
	".webp": {},
	".heic": {},
}

// pdfExtensions is trivial but kept as a set for symmetry with images.
var pdfExtensions = map[string]struct{}{
	".pdf": {},
}

// ClassifyExtension returns the type implied by the extension alone. Returns
// FileTypeUnknown when the extension is not recognised.
func ClassifyExtension(ext string) model.FileType {
	if _, ok := imageExtensions[ext]; ok {
		return model.FileTypeImage
	}
	if _, ok := pdfExtensions[ext]; ok {
		return model.FileTypePDF
	}
	return model.FileTypeUnknown
}
