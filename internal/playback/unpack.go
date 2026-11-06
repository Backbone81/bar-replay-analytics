package playback

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bodgit/sevenzip"
)

// Unpack7z unpacks the 7z archive archivePath into destination, which it creates. Files keep their stored
// permissions, so the engine stays executable. It rejects names that leave destination and entries other than files
// and folders, such as links; engine archives have neither. Cancelling ctx stops it between files.
func Unpack7z(ctx context.Context, archivePath string, destination string) error {
	archive, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close() //nolint:errcheck

	if err := os.MkdirAll(destination, dirPermissions); err != nil {
		return err
	}
	// The entries of a solid archive are unpacked fastest in the order they are stored in.
	for _, file := range archive.File {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := unpack7zFile(file, destination); err != nil {
			return fmt.Errorf("unpacking %s from %s: %w", file.Name, archivePath, err)
		}
	}
	return nil
}

// unpack7zFile unpacks the archive entry file into the folder destination.
func unpack7zFile(file *sevenzip.File, destination string) error {
	name := filepath.FromSlash(file.Name)
	if !filepath.IsLocal(name) {
		return errors.New("the name leaves the destination folder")
	}
	target := filepath.Join(destination, name)

	mode := file.Mode()
	switch {
	case mode.IsDir():
		return os.MkdirAll(target, dirPermissions)
	case !mode.IsRegular():
		return fmt.Errorf("unsupported file type %s", mode.Type())
	}

	if err := os.MkdirAll(filepath.Dir(target), dirPermissions); err != nil {
		return err
	}
	permissions := mode.Perm()
	if permissions == 0 {
		// An archive made on Windows may store no Unix permissions.
		permissions = filePermissions
	}
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close() //nolint:errcheck

	//nolint:gosec // target is checked to be inside destination, a folder of the package.
	writer, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions|0o200)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, reader)
	return errors.Join(err, writer.Close())
}
