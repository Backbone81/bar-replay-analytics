package playback

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
)

// executableName returns the file name of the engine's program name on this platform: with .exe on Windows.
func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// safeNamePattern describes the names Prepare accepts from a replay, which end up in paths and command lines: no path
// separator, and no start like an option or a relative path. It allows "2026.07.04",
// "Beyond All Reason test-31720-0f97aa5" and "Argent Strata 1.1".
var safeNamePattern = regexp.MustCompile(`^[A-Za-z0-9][^/\\:\x00-\x1f]*$`)

// validateReplayName returns an error if name, read from a replay, is not safe to use in a path or as a command
// argument.
func validateReplayName(name string) error {
	if !safeNamePattern.MatchString(name) {
		return fmt.Errorf("%q is not a plain name", name)
	}
	return nil
}

// filePermissions are the permissions of the files the package writes: nothing in them is confidential.
const filePermissions = 0o644

// dirPermissions are the permissions of the directories the package creates.
const dirPermissions = 0o755

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
