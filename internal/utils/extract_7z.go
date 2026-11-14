package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bodgit/sevenzip"
)

// Extract7z unpacks the 7z archive archiveFilePath into the folder destinationDir, replacing it if it already exists.
// It unpacks into a temporary folder <folder name>.part-<random> next to dir and renames that once every entry is
// unpacked, so dir is never partly unpacked.
func Extract7z(ctx context.Context, archiveFilePath string, destinationDir string) error {
	tempDir, err := extract7zToTempDir(ctx, archiveFilePath, destinationDir)
	if err != nil {
		return fmt.Errorf("unpacking %q into %q: %w", archiveFilePath, destinationDir, err)
	}

	// A folder can only be renamed onto a missing one. An existing dir is removed first. A running program in it keeps
	// its files until it ends.
	if err := os.RemoveAll(destinationDir); err != nil {
		return errors.Join(err, os.RemoveAll(tempDir))
	}
	if err := os.Rename(tempDir, destinationDir); err != nil {
		return errors.Join(err, os.RemoveAll(tempDir))
	}
	return nil
}

// extract7zToTempDir unpacks the 7z archive archiveFilePath into a new temporary folder next to destinationDir,
// readable by everyone, and returns its path. It removes the temporary folder if anything fails.
func extract7zToTempDir(ctx context.Context, archiveFilePath string, destinationDir string) (_ string, err error) {
	// A local variable, not the result: a return sets the result before the deferred cleanup reads it.
	tempDir, err := os.MkdirTemp(filepath.Dir(destinationDir), filepath.Base(destinationDir)+".part-*")
	if err != nil {
		return "", err
	}
	// Registered first, so it runs last and sees every error, also the one of closing the archive.
	defer func() {
		if err != nil {
			err = errors.Join(err, os.RemoveAll(tempDir))
		}
	}()

	//nolint:gosec // This folder does not contain sensitive data.
	if err = os.Chmod(tempDir, 0o755); err != nil {
		return "", err
	}

	reader, err := sevenzip.OpenReader(archiveFilePath)
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, reader.Close())
	}()

	for _, file := range reader.File {
		if err = ctx.Err(); err != nil {
			return "", err
		}
		if err = extract7zFile(file, tempDir); err != nil {
			return "", err
		}
	}

	return tempDir, nil
}

// extract7zFile unpacks the archive entry file into destinationDir. It rejects entries that leave dir and anything but
// folders and regular files, as the archive may come from the internet. destinationDir is the temporary folder of
// extract7zToTempDir, which is removed as a whole if anything fails, so the file is written directly, without a
// temporary file of its own.
func extract7zFile(file *sevenzip.File, destinationDir string) (err error) {
	destination := filepath.FromSlash(file.Name)
	if !filepath.IsLocal(destination) {
		return fmt.Errorf("%q leaves the destination folder", file.Name)
	}

	destination = filepath.Join(destinationDir, destination)
	if file.Mode().IsDir() {
		//nolint:gosec // This folder does not contain sensitive data.
		return os.MkdirAll(destination, 0o755)
	}
	if !file.Mode().IsRegular() {
		return fmt.Errorf("%q: unsupported file type %s", file.Name, file.Mode().Type())
	}

	//nolint:gosec // This folder does not contain sensitive data.
	if err = os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, reader.Close())
	}()

	//nolint:gosec // The entry stays inside dir, see above.
	writer, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE, file.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() {
		// Close can report a failed write.
		err = errors.Join(err, writer.Close())
	}()

	_, err = io.Copy(writer, reader)
	return err
}
