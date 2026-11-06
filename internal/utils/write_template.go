package utils

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// WriteTemplate writes every file of fsys into the folder dir, keeping their paths in fsys and creating the folders
// they need. It never overwrites a file: it stops at the first one that exists and returns an error wrapping
// fs.ErrExist.
func WriteTemplate(log *slog.Logger, fsys fs.FS, dir string) error {
	return fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		filePath := filepath.Join(dir, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(filePath, 0o755) //nolint:gosec // The dir does not contain sensitive data
		}

		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}

		log.Info("Creating file", "path", filePath)
		if err := writeTemplateFile(filePath, content); err != nil {
			return err
		}
		return nil
	})
}

// writeTemplateFile creates filePath with content. It returns an error if the file already exists.
func writeTemplateFile(filePath string, content []byte) error {
	//nolint:gosec // The caller chooses the folder; the names come from the embedded files.
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer file.Close() //nolint:errcheck

	if _, err := file.Write(content); err != nil {
		return fmt.Errorf("writing file %q: %w", filePath, err)
	}
	return nil
}
