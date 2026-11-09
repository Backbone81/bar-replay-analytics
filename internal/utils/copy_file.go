package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
)

func CopyFile(sourceFilePath string, destinationFilePath string) (err error) {
	sourceFile, err := os.Open(sourceFilePath) //nolint:gosec // The caller is responsible for the file paths.
	if err != nil {
		return fmt.Errorf("opening file %q: %w", sourceFilePath, err)
	}
	defer func() {
		err = errors.Join(err, sourceFile.Close())
	}()

	//nolint:gosec // This file does not contain sensitive data
	destinationFile, err := os.OpenFile(
		destinationFilePath,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		0o644,
	)
	if err != nil {
		return fmt.Errorf("opening file %q: %w", destinationFilePath, err)
	}
	defer func() {
		err = errors.Join(err, destinationFile.Close())
	}()

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		return fmt.Errorf("copying file %q to %q: %w", sourceFilePath, destinationFilePath, err)
	}
	return nil
}
