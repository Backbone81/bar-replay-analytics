package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileExists returns whether filePath exists and is not a folder.
func FileExists(filePath string) bool {
	stat, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

// DirExists returns whether dir exists and is a folder.
func DirExists(dir string) bool {
	stat, err := os.Stat(dir)
	if err != nil {
		return false
	}
	return stat.IsDir()
}

// ExecutableExists returns whether the executable filePath exists as a file, with or without the .exe of Windows.
func ExecutableExists(filePath string) bool {
	return FileExists(filePath) || FileExists(filePath+".exe")
}

// WriteFile writes data to filePath with the permissions perm, replacing it, like os.WriteFile. It goes through
// WriteFileFrom, so filePath is never incomplete.
func WriteFile(filePath string, data []byte, perm os.FileMode) error {
	return WriteFileFrom(filePath, bytes.NewReader(data), perm)
}

// WriteFileFrom writes everything reader returns to filePath with the permissions perm, replacing it. It writes to a
// temporary file <file name>.part-<random> next to filePath and renames that, so filePath is never incomplete.
func WriteFileFrom(filePath string, reader io.Reader, perm os.FileMode) error {
	tempFilePath, err := writeTempFileFrom(filePath, reader, perm)
	if err != nil {
		return fmt.Errorf("writing file %q: %w", filePath, err)
	}
	if err := os.Rename(tempFilePath, filePath); err != nil {
		return errors.Join(err, os.Remove(tempFilePath))
	}
	return nil
}

// writeTempFileFrom writes everything reader returns to a new temporary file next to filePath with the permissions
// perm, and returns its path. It removes the temporary file if anything fails.
//
//nolint:nonamedreturns // Named returns are required because of defer cleanups
func writeTempFileFrom(filePath string, reader io.Reader, perm os.FileMode) (tempFilePath string, err error) {
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), filepath.Base(filePath)+".part-*")
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, tempFile.Close())
		if err != nil {
			err = errors.Join(err, os.Remove(tempFile.Name()))
			tempFilePath = ""
		}
	}()

	if _, err = io.Copy(tempFile, reader); err != nil {
		return "", err
	}

	// CreateTemp creates the file readable by its owner only.

	if err = os.Chmod(tempFile.Name(), perm); err != nil {
		return "", err
	}
	return tempFile.Name(), nil
}

// CopyFile copies the file sourceFilePath to destinationFilePath, readable by everyone, replacing it. It goes through
// WriteFileFrom, so destinationFilePath is never incomplete.
func CopyFile(sourceFilePath string, destinationFilePath string) (err error) {
	sourceFile, err := os.Open(sourceFilePath) //nolint:gosec // The caller is responsible for the file paths.
	if err != nil {
		return fmt.Errorf("opening file %q: %w", sourceFilePath, err)
	}
	defer func() {
		err = errors.Join(err, sourceFile.Close())
	}()

	return WriteFileFrom(destinationFilePath, sourceFile, 0o644)
}
