// Package newfiles writes embedded files into a folder without overwriting anything: a file that exists already is
// left as it is and reported, as it may hold the user's changes.
package newfiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// filePermissions are the permissions of the files Write creates: nothing in them is confidential.
const filePermissions = 0o644

// dirPermissions are the permissions of the folders Write creates.
const dirPermissions = 0o755

// ExistError is returned by Write for the files it left alone because they exist already.
type ExistError struct {
	// FilePaths are the files that exist already.
	FilePaths []string
}

// Error lists the files that exist already.
func (e *ExistError) Error() string {
	return "not overwriting existing files: " + strings.Join(e.FilePaths, ", ")
}

// Write writes every file of fsys into the folder dir, keeping their paths in fsys and creating the folders they need.
// It returns the files it created.
//
// Write never overwrites a file: it creates the files that are missing and leaves the others alone. If any of them
// existed, it returns an *ExistError listing them, after creating the missing ones.
func Write(fsys fs.FS, dir string) ([]string, error) {
	var created []string
	var existing []string
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		filePath := filepath.Join(dir, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(filePath, dirPermissions)
		}
		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		if err := writeNewFile(filePath, content); err != nil {
			if errors.Is(err, fs.ErrExist) {
				existing = append(existing, filePath)
				return nil
			}
			return err
		}
		created = append(created, filePath)
		return nil
	})
	if err != nil {
		return created, err
	}
	if len(existing) > 0 {
		return created, &ExistError{FilePaths: existing}
	}
	return created, nil
}

// writeNewFile creates filePath with content. It returns an error wrapping fs.ErrExist if filePath exists, and removes
// the file again if writing it fails, so no partial file is left behind.
func writeNewFile(filePath string, content []byte) error {
	//nolint:gosec // The caller chooses the folder; the names come from the embedded files.
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePermissions)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		_ = os.Remove(filePath)
		return fmt.Errorf("writing %s: %w", filePath, err)
	}
	return nil
}
