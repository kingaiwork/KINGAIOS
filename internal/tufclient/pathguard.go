package tufclient

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func validateTrustedRootPath(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("trusted TUF root path must be absolute")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect trusted TUF root: %w", err)
	}
	if !st.Mode().IsRegular() {
		return errors.New("trusted TUF root must be a regular file; symlinks and special files are refused")
	}
	if st.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("trusted TUF root is writable by group/other (mode %04o)", st.Mode().Perm())
	}
	dir, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("inspect trusted TUF root directory: %w", err)
	}
	if !dir.IsDir() || dir.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("trusted TUF root directory is not a protected directory (mode %04o)", dir.Mode().Perm())
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("TUF state directory must be absolute")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("TUF state path %s must be a real directory", path)
	}
	if st.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("TUF state directory %s is writable by group/other (mode %04o)", path, st.Mode().Perm())
	}
	return nil
}
