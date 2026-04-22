package fs

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// ErrDestExistsMismatch is returned when a destination already exists with
// different bytes than the source. The operation refuses to overwrite.
var ErrDestExistsMismatch = errors.New("destination already exists with different content")

// EnsureParent makes sure the parent directory of path exists (mkdir -p).
func EnsureParent(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// CopyFile copies src to dst via an intermediate `<dst>.tmp-<pid>` to keep
// dst atomic from a reader's point of view. If dst already exists, CopyFile
// compares bytes:
//
//   - identical content → returns (false, nil) to mean "already there"
//   - different content → returns (false, ErrDestExistsMismatch)
//
// On success, returns (true, nil).
func CopyFile(src, dst string) (copied bool, err error) {
	if err := EnsureParent(dst); err != nil {
		return false, err
	}
	if equal, exists, err := sameBytes(src, dst); err != nil {
		return false, err
	} else if exists {
		if equal {
			return false, nil
		}
		return false, ErrDestExistsMismatch
	}

	in, err := os.Open(src)
	if err != nil {
		return false, err
	}
	defer in.Close()

	tmp := dst + ".tmp-dedup"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return false, err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return false, err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		os.Remove(tmp)
		return false, err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return false, err
	}
	if err := os.Rename(tmp, dst); err != nil {
		os.Remove(tmp)
		return false, err
	}
	return true, nil
}

// Move renames src to dst. If the rename crosses devices (EXDEV), Move falls
// back to copy-then-delete. Return values mirror CopyFile:
//   - (true, nil)                    success
//   - (false, nil)                   dst already contained identical bytes;
//                                    src was removed to match "move" semantics
//   - (false, ErrDestExistsMismatch) dst existed with different content
func Move(src, dst string) (moved bool, err error) {
	if err := EnsureParent(dst); err != nil {
		return false, err
	}
	if equal, exists, err := sameBytes(src, dst); err != nil {
		return false, err
	} else if exists {
		if equal {
			return false, os.Remove(src)
		}
		return false, ErrDestExistsMismatch
	}

	err = os.Rename(src, dst)
	if err == nil {
		return true, nil
	}
	if !isCrossDevice(err) {
		return false, err
	}

	if _, err := CopyFile(src, dst); err != nil {
		return false, err
	}
	if err := os.Remove(src); err != nil {
		return false, err
	}
	return true, nil
}

// Delete removes path. It refuses to remove directories, always.
func Delete(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.IsDir() {
		return errors.New("refusing to delete a directory")
	}
	return os.Remove(path)
}

// sameBytes reports:
//   - equal:  both files exist and their content is identical
//   - exists: dst exists (regardless of equality)
//   - err:    non-"not-exists" I/O failures
func sameBytes(src, dst string) (equal, exists bool, err error) {
	dfi, derr := os.Stat(dst)
	if derr != nil {
		if os.IsNotExist(derr) {
			return false, false, nil
		}
		return false, false, derr
	}
	if dfi.IsDir() {
		return false, true, errors.New("destination is a directory")
	}
	exists = true

	sfi, serr := os.Stat(src)
	if serr != nil {
		return false, exists, serr
	}
	if sfi.Size() != dfi.Size() {
		return false, exists, nil
	}

	sf, err := os.Open(src)
	if err != nil {
		return false, exists, err
	}
	defer sf.Close()
	df, err := os.Open(dst)
	if err != nil {
		return false, exists, err
	}
	defer df.Close()

	const bufSize = 64 * 1024
	a := make([]byte, bufSize)
	b := make([]byte, bufSize)
	for {
		na, _ := io.ReadFull(sf, a)
		nb, _ := io.ReadFull(df, b)
		if na != nb || !bytes.Equal(a[:na], b[:nb]) {
			return false, exists, nil
		}
		if na == 0 {
			return true, exists, nil
		}
	}
}

// isCrossDevice reports whether err is the Linux EXDEV error that os.Rename
// returns when src and dst are on different filesystems.
func isCrossDevice(err error) bool {
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		if errno, ok := linkErr.Err.(syscall.Errno); ok {
			return errno == syscall.EXDEV
		}
	}
	return false
}
