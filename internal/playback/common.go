package playback

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)

// executableName returns the file name of the program name on this platform: with .exe on Windows.
func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// safeNamePattern matches the names Prepare accepts from a replay: no path separator or control character, and an
// alphanumeric start, so no option ("-x") or relative path (".."). It allows "2026.07.04",
// "Beyond All Reason test-31720-0f97aa5" and "Argent Strata 1.1".
var safeNamePattern = regexp.MustCompile(`^[A-Za-z0-9][^/\\:\x00-\x1f]*$`)

// validateReplayName returns an error if name, read from a replay, does not match safeNamePattern.
func validateReplayName(name string) error {
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("%q is not a plain name", name)
	}
	return nil
}

// filePermissions are the permissions of the files the package writes; none is confidential.
const filePermissions = 0o644

// dirPermissions are the permissions of the directories the package creates.
const dirPermissions = 0o755

// copyFileAtomically copies the file sourceFilePath to destinationFilePath, replacing it if it exists. It writes a
// temporary file next to it and renames that, so destinationFilePath is never incomplete. A missing source is an
// error that wraps fs.ErrNotExist.
func copyFileAtomically(sourceFilePath string, destinationFilePath string) error {
	if _, err := os.Stat(sourceFilePath); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(destinationFilePath), filepath.Base(destinationFilePath)+".tmp-*")
	if err != nil {
		return err
	}
	tempFilePath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return errors.Join(err, os.Remove(tempFilePath))
	}
	if err := copyFile(sourceFilePath, tempFilePath); err != nil {
		return errors.Join(err, os.Remove(tempFilePath))
	}
	if err := os.Chmod(tempFilePath, filePermissions); err != nil {
		return errors.Join(err, os.Remove(tempFilePath))
	}
	if err := os.Rename(tempFilePath, destinationFilePath); err != nil {
		return errors.Join(err, os.Remove(tempFilePath))
	}
	return nil
}

// copyFile copies the file sourceFilePath to destinationFilePath, which it creates or truncates.
func copyFile(sourceFilePath string, destinationFilePath string) error {
	sourceFile, err := os.Open(sourceFilePath) //nolint:gosec // The data dir's files are what the package works with.
	if err != nil {
		return err
	}
	defer sourceFile.Close() //nolint:errcheck

	//nolint:gosec // target is in a folder of the package; see filePermissions.
	targetFile, err := os.OpenFile(destinationFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePermissions)
	if err != nil {
		return err
	}
	_, err = io.Copy(targetFile, sourceFile)
	return errors.Join(err, targetFile.Close())
}
