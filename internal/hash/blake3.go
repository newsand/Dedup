// Package hash computes BLAKE3 digests of files, streamed through a 64 KiB
// buffer. The package is intentionally tiny — it is the hot path and we want
// every allocation accounted for.
package hash

import (
	"encoding/hex"
	"io"
	"os"

	"github.com/zeebo/blake3"
)

// bufSize is the copy buffer used by FileBLAKE3.
const bufSize = 64 * 1024

// FileBLAKE3 hashes the entire contents of path and returns (hex, size, err).
// The file is opened with O_RDONLY and closed before returning.
func FileBLAKE3(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	return HashReader(f)
}

// HashReader hashes from r until io.EOF and returns (hex, bytesRead, err).
// The function does not close r. Callers that need to hash an open file should
// prefer FileBLAKE3 to avoid having to manage the descriptor themselves.
func HashReader(r io.Reader) (string, int64, error) {
	h := blake3.New()
	buf := make([]byte, bufSize)
	n, err := io.CopyBuffer(h, r, buf)
	if err != nil {
		return "", n, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
