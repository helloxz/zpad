// Package atomicfile writes a file through a temporary file in the same
// directory, then renames it into place, so a crash cannot leave a
// half-written file.
package atomicfile

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var writerPool = sync.Pool{
	New: func() any { return bufio.NewWriterSize(nil, 32<<10) },
}

// Write writes data to dir/name with mode 0600, for settings and session
// files.
func Write(dir, name string, data []byte) error {
	return WriteMode(dir, name, data, 0o600)
}

// WriteMode writes data to dir/name with mode.
func WriteMode(dir, name string, data []byte, mode os.FileMode) error {
	return WriteTo(dir, name, mode, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

// WriteTo writes what fn writes to dir/name with mode, buffered in 32 KB
// blocks through a temporary file, so that a large text is written without a
// copy of it in memory or a system call per rope chunk.
func WriteTo(dir, name string, mode os.FileMode, fn func(io.Writer) error) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+name+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if tmpName != "" {
			_ = os.Remove(tmpName)
		}
	}()

	bw := writerPool.Get().(*bufio.Writer)
	bw.Reset(tmp)
	err = fn(bw)
	if err == nil {
		err = bw.Flush()
	}
	bw.Reset(nil)
	writerPool.Put(bw)
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(dir, name)); err != nil {
		return err
	}
	tmpName = ""
	return nil
}

// WriteFile writes data to path, keeping the file's directory and using
// mode, or the mode of the file already there.
func WriteFile(path string, data []byte, mode os.FileMode) error {
	return WriteFileTo(path, mode, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
}

// WriteFileTo streams what fn writes to path atomically, keeping the mode of
// the file already there or using mode for a new file.
func WriteFileTo(path string, mode os.FileMode, fn func(io.Writer) error) error {
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return WriteTo(filepath.Dir(path), filepath.Base(path), mode, fn)
}
