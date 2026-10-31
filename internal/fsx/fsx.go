// Package fsx holds file helpers shared by the commands that write files.
package fsx

import (
	"os"
	"path/filepath"
)

// ReplaceFile writes data to a new temp file next to path and renames it
// over path, so a symbolic link at path is replaced instead of followed and
// readers never see a half-written file.
func ReplaceFile(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".trama-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
