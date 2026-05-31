package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// tokenFilePerm is the permission of files written by atomicWrite (current user only).
const tokenFilePerm os.FileMode = 0o600

// atomicWrite writes data to path atomically: it writes to a temporary file in the
// same directory, sets its permission to 0600, and renames it into place. A
// concurrent writer can at worst lose its write but never observe a partial file.
// The parent directory is created with 0700 if it does not exist.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, socketDirPerm); err != nil {
		return fmt.Errorf("create the directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".ghtkn-tmp-*")
	if err != nil {
		return fmt.Errorf("create a temporary file: %w", err)
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(tokenFilePerm); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("set the temporary file permission: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write to the temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close the temporary file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename the temporary file: %w", err)
	}
	return nil
}
