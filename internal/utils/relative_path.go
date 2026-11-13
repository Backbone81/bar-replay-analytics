package utils

import (
	"path/filepath"
	"strings"
)

func RelativePath(basePath string, targetPath string) string {
	relativePath, err := filepath.Rel(basePath, targetPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return targetPath
	}
	return relativePath
}
