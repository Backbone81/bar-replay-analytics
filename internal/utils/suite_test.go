package utils_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

const (
	// testEngineVersion is the engine of the tests.
	testEngineVersion = "2026.07.04"

	// testMapName is the map of the tests.
	testMapName = "Argent Strata 1.1"

	// testMapFileName is the archive of testMapName in maps/.
	testMapFileName = "argent_strata_1.1.sd7"

	// testGameType is the game of the tests.
	testGameType = "Beyond All Reason test-31720-0f97aa5"

	// testGamePackage is the md5 of the rapid package of testGameType.
	testGamePackage = "0123456789abcdef0123456789abcdef"

	// testRapidRepo is the folder of the rapid repo in rapid/.
	testRapidRepo = "repos-cdn.beyondallreason.dev"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Utils Suite")
}

// writeFile writes content to filePath, creating its folder.
func writeFile(filePath string, content string) {
	Expect(os.MkdirAll(filepath.Dir(filePath), 0o750)).To(Succeed())
	Expect(os.WriteFile(filePath, []byte(content), 0o600)).To(Succeed())
}

// newDataDir returns a new empty data dir.
func newDataDir() string {
	return GinkgoT().TempDir()
}

// addEngineFiles adds the folder engine/folderName to dataDir with the files fileNames, which are not executable.
func addEngineFiles(dataDir string, folderName string, fileNames ...string) {
	Expect(os.MkdirAll(filepath.Join(utils.EngineDir(dataDir), folderName), 0o750)).To(Succeed())
	for _, fileName := range fileNames {
		writeFile(filepath.Join(utils.EngineDir(dataDir), folderName, fileName), "")
	}
}

// addMapFile adds the map archive fileName to the maps/ of dataDir.
func addMapFile(dataDir string, fileName string) {
	writeFile(filepath.Join(utils.MapsDir(dataDir), fileName), "")
}

// addGame adds the game name with the rapid package md5 to dataDir: a versions.gz that lists it and its package.
func addGame(dataDir string, md5 string, name string) {
	writeRapidVersions(dataDir, "byar:test,"+md5+",,"+name)
	addPackage(dataDir, md5)
}

// writeRapidVersions writes the versions.gz of the BAR rapid repo in dataDir with lines.
func writeRapidVersions(dataDir string, lines ...string) {
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	Expect(writer.Write([]byte(strings.Join(lines, "\n") + "\n"))).Error().NotTo(HaveOccurred())
	Expect(writer.Close()).To(Succeed())
	writeFile(rapidVersionsFile(dataDir), buffer.String())
}

// rapidVersionsFile returns the path of the versions.gz of the BAR rapid repo in dataDir.
func rapidVersionsFile(dataDir string) string {
	return filepath.Join(utils.RapidDir(dataDir), testRapidRepo, "byar", "versions.gz")
}

// addPackage adds the rapid package file of md5 to dataDir.
func addPackage(dataDir string, md5 string) {
	writeFile(filepath.Join(utils.PackagesDir(dataDir), md5+".sdp"), "")
}

// fileMode returns the permission bits of filePath.
func fileMode(filePath string) os.FileMode {
	info, err := os.Stat(filePath)
	Expect(err).NotTo(HaveOccurred())
	return info.Mode().Perm()
}

// fileNames returns the names of the entries of dir.
func fileNames(dir string) []string {
	entries, err := os.ReadDir(dir)
	Expect(err).NotTo(HaveOccurred())
	var result []string
	for _, entry := range entries {
		result = append(result, entry.Name())
	}
	return result
}

// errTestRead is the error of a reader that fails in a test.
var errTestRead = errors.New("test read error")
