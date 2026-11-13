package utils

import "path/filepath"

func DemosDir(dataDir string) string {
	return filepath.Join(dataDir, "demos")
}
