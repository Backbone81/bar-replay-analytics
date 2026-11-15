package utils_test

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
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

	// testGameTime is the game time of the test replays in seconds.
	testGameTime = 1607
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

// copyTestdata copies the file name from testdata/ to dir and returns the copy's path.
func copyTestdata(name string, dir string) string {
	filePath := filepath.Join(dir, name)
	Expect(utils.CopyFile(filepath.Join("testdata", name), filePath)).To(Succeed())
	return filePath
}

// testSetupScript returns a start script with the map mapName and the game gameType, a player, a spectator and two AIs.
//
//nolint:unparam
func testSetupScript(mapName string, gameType string) string {
	return "[game]\n{\n" +
		"\tmapname=" + mapName + ";\n" +
		"\tgametype=" + gameType + ";\n" +
		"\t[player0]\n\t{\n\t\tname=Alice;\n\t\tspectator=0;\n\t}\n" +
		"\t[player1]\n\t{\n\t\tname=Watcher;\n\t\tspectator=1;\n\t}\n" +
		"\t[ai0]\n\t{\n\t\tname=BARb Hard;\n\t\tshortname=BARb;\n\t}\n" +
		"\t[ai1]\n\t{\n\t\tshortname=SimpleAI;\n\t}\n" +
		"}\n"
}

// writeReplay writes a replay file to filePath with engineVersion, setupScript, a stream of demoStreamSize bytes and a
// game time of gameTime seconds. It has no statistics: ReadReplayInfo reads only the header and the start script.
//
//nolint:unparam
func writeReplay(filePath string, engineVersion string, setupScript string, demoStreamSize int, gameTime int) {
	header := demo.DemoFileHeader{
		Version:            demo.DemofileVersion,
		HeaderSize:         int32(binary.Size(demo.DemoFileHeader{})),
		ScriptSize:         int32(len(setupScript)),
		DemoStreamSize:     int32(demoStreamSize),
		GameTime:           int32(gameTime),
		UnixTime:           uint64(time.Now().UTC().Unix()),
		PlayerStatElemSize: int32(binary.Size(demo.PlayerStatistics{})),
		TeamStatElemSize:   int32(binary.Size(demo.TeamStatistics{})),
		TeamStatPeriod:     15,
	}
	copy(header.Magic[:], demo.DemofileMagic)
	copy(header.VersionString[:], engineVersion)

	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	Expect(binary.Write(writer, demo.ByteOrder, header)).To(Succeed())
	Expect(writer.Write([]byte(setupScript))).Error().NotTo(HaveOccurred())
	Expect(writer.Write(make([]byte, demoStreamSize))).Error().NotTo(HaveOccurred())
	Expect(writer.Close()).To(Succeed())
	writeFile(filePath, buffer.String())
}

// writeTestReplay writes a playable test replay on mapName to filePath and returns filePath.
func writeTestReplay(filePath string, mapName string) string {
	writeReplay(filePath, testEngineVersion, testSetupScript(mapName, testGameType), 16, testGameTime)
	return filePath
}

// demosFile returns the path of fileName in the demos/ of dataDir.
func demosFile(dataDir string, fileName string) string {
	return filepath.Join(utils.DemosDir(dataDir), fileName)
}
