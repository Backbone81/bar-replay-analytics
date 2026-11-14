package player_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	replayplayer "github.com/backbone81/bar-replay-analytics-go/internal/replay/player"
	"github.com/backbone81/bar-replay-analytics-go/internal/replay/player/widget"
	"github.com/backbone81/bar-replay-analytics-go/internal/utils"
)

// The suite's test binary is also the fake engine and the fake pr-downloader, as the tests of os/exec do it: the tests
// link it into a fake engine folder as spring-headless and pr-downloader, and TestMain tells by the name it was started
// under what it is to be. The fakes read what to do from a behaviour file and write what they were called with to a
// call file, both next to the replay (spring-headless) or in the data dir (pr-downloader).

const (
	// testEngineVersion is the engine of the test replays.
	testEngineVersion = "2026.07.04"

	// testMapName is the map of the test replays.
	testMapName = "Argent Strata 1.1"

	// testMapFileName is the archive of testMapName in maps/.
	testMapFileName = "argent_strata_1.1.sd7"

	// testGameType is the game of the test replays.
	testGameType = "Beyond All Reason test-31720-0f97aa5"

	// testGamePackage is the md5 of the rapid package of testGameType.
	testGamePackage = "0123456789abcdef0123456789abcdef"

	// testRapidRepo is the folder of the rapid repo in rapid/.
	testRapidRepo = "repos-cdn.beyondallreason.dev"

	// testGameTime is the game time of the test replays in seconds.
	testGameTime = 1607

	// springHeadlessBehaviourSuffix, springHeadlessCallSuffix and springHeadlessSignalSuffix replace a replay's
	// extension for the fake spring-headless's files next to the replay.
	springHeadlessBehaviourSuffix = ".spring-headless-behaviour.json"
	springHeadlessCallSuffix      = ".spring-headless-call.json"
	springHeadlessSignalSuffix    = ".spring-headless-signal"

	// prDownloaderBehaviourFile and prDownloaderCallsFile are the fake pr-downloader's files in the data dir. The calls
	// file has one JSON line per call.
	prDownloaderBehaviourFile = "pr-downloader-behaviour.json"
	prDownloaderCallsFile     = "pr-downloader-calls.jsonl"

	// offlineEngineDownloadURL is the engine download URL of the AssetManagers of the tests, unless a test sets its own:
	// nothing listens on port 1, so a missing engine fails at once instead of being downloaded from BAR's CDN.
	offlineEngineDownloadURL = "http://127.0.0.1:1/{{.EngineVersion}}.7z"

	// fakeExitCodeFailure is the exit code of a fake that could not do what its behaviour file says.
	fakeExitCodeFailure = 99
)

// springHeadlessBehaviour is what the fake spring-headless does.
type springHeadlessBehaviour struct {
	// ExitCode is the exit code it exits with.
	ExitCode int `json:"exitCode"`

	// Output is what it writes to stdout.
	Output string `json:"output"`

	// EventsFile is what it writes as the replay's events file into the write dir, where the analytics widget writes
	// it. If empty, it writes no events file.
	EventsFile string `json:"eventsFile"`

	// WaitForFile is a file it waits for before it exits, for at most 10 s.
	WaitForFile string `json:"waitForFile"`

	// WaitForSignal makes it wait for SIGTERM and exit with exit code 1 then.
	WaitForSignal bool `json:"waitForSignal"`
}

// springHeadlessCall is what the fake spring-headless was called with.
type springHeadlessCall struct {
	// Args are its arguments without the program name.
	Args []string `json:"args"`

	// Dir is its working dir.
	Dir string `json:"dir"`

	// SpringDataDir is its SPRING_DATADIR.
	SpringDataDir string `json:"springDataDir"`

	// WriteDirFiles are the files in the write dir (--write-dir) as it started, relative to it and slash separated.
	WriteDirFiles []string `json:"writeDirFiles"`

	// SpringSettings is the content of the config file (--config).
	SpringSettings string `json:"springSettings"`

	// WidgetConfig is the content of the widget config file in the write dir, empty if there is none.
	WidgetConfig string `json:"widgetConfig"`
}

// prDownloaderBehaviour is what the fake pr-downloader does.
type prDownloaderBehaviour struct {
	// ExitCode is the exit code it exits with.
	ExitCode int `json:"exitCode"`

	// Output is what it writes to stdout.
	Output string `json:"output"`

	// Sleep is how long it takes.
	Sleep time.Duration `json:"sleep"`
}

// prDownloaderCall is what the fake pr-downloader was called with.
type prDownloaderCall struct {
	// Args are its arguments without the program name.
	Args []string `json:"args"`

	// Dir is its working dir.
	Dir string `json:"dir"`

	// Env are its environment variables that start with PRD_.
	Env []string `json:"env"`
}

func TestMain(m *testing.M) {
	switch strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe") {
	case "spring-headless":
		os.Exit(runFakeSpringHeadless())
	case "pr-downloader":
		os.Exit(runFakePrDownloader())
	}
	os.Exit(m.Run())
}

func TestReplayPlayer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ReplayPlayer Suite")
}

// runFakeSpringHeadless is the fake spring-headless: it records its call next to the replay (its last argument) and
// does what the behaviour file there says. It returns its exit code.
func runFakeSpringHeadless() int {
	exitCode, err := fakeSpringHeadless()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fake spring-headless:", err)
		return fakeExitCodeFailure
	}
	return exitCode
}

// fakeSpringHeadless does the work of runFakeSpringHeadless.
func fakeSpringHeadless() (int, error) {
	args := os.Args[1:]
	if len(args) == 0 {
		return 0, errors.New("no replay given")
	}
	replayFilePath := args[len(args)-1]
	replayBase := strings.TrimSuffix(replayFilePath, filepath.Ext(replayFilePath))

	var behaviour springHeadlessBehaviour
	if err := readOptionalJSON(replayBase+springHeadlessBehaviourSuffix, &behaviour); err != nil {
		return 0, err
	}

	// The signal handler is installed before the call is recorded, so a test that waits for the call file can send
	// the signal right away.
	signals := make(chan os.Signal, 1)
	if behaviour.WaitForSignal {
		signal.Notify(signals, syscall.SIGTERM)
	}

	call, err := newSpringHeadlessCall(args)
	if err != nil {
		return 0, err
	}
	if err := writeJSON(replayBase+springHeadlessCallSuffix, call); err != nil {
		return 0, err
	}

	//nolint:forbidigo
	fmt.Print(behaviour.Output)
	if behaviour.EventsFile != "" {
		// The widget writes the events file relative to the working directory, which is the write dir.
		eventsFilePath := filepath.Join(
			utils.AnalyticsWidgetEventsDir("."),
			filepath.Base(utils.DemoEventsFilePath(replayFilePath)),
		)
		if err := writeFileForFake(eventsFilePath, []byte(behaviour.EventsFile)); err != nil {
			return 0, err
		}
	}
	if behaviour.WaitForFile != "" {
		if err := waitForFile(behaviour.WaitForFile, 10*time.Second); err != nil {
			return 0, err
		}
	}
	if behaviour.WaitForSignal {
		received := <-signals
		if err := writeFileForFake(replayBase+springHeadlessSignalSuffix, []byte(received.String())); err != nil {
			return 0, err
		}
		return 1, nil
	}
	return behaviour.ExitCode, nil
}

// newSpringHeadlessCall returns the call of the fake spring-headless with args.
func newSpringHeadlessCall(args []string) (*springHeadlessCall, error) {
	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	call := springHeadlessCall{
		Args:          args,
		Dir:           dir,
		SpringDataDir: os.Getenv("SPRING_DATADIR"),
	}
	if writeDir := argumentValue(args, "--write-dir"); writeDir != "" {
		call.WriteDirFiles, err = relativeFiles(writeDir)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(utils.WidgetHandlerConfigFilePath(writeDir))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		call.WidgetConfig = string(data)
	}
	if config := argumentValue(args, "--config"); config != "" {
		data, err := os.ReadFile(config)
		if err != nil {
			return nil, err
		}
		call.SpringSettings = string(data)
	}
	return &call, nil
}

// runFakePrDownloader is the fake pr-downloader: it appends its call to the calls file in the data dir
// (--filesystem-writepath) and does what the behaviour file there says. It returns its exit code.
func runFakePrDownloader() int {
	exitCode, err := fakePrDownloader()
	if err != nil {
		fmt.Fprintln(os.Stderr, "fake pr-downloader:", err)
		return fakeExitCodeFailure
	}
	return exitCode
}

// fakePrDownloader does the work of runFakePrDownloader.
func fakePrDownloader() (int, error) {
	args := os.Args[1:]
	dataDir := argumentValue(args, "--filesystem-writepath")
	if dataDir == "" {
		return 0, errors.New("no --filesystem-writepath given")
	}

	var behaviour prDownloaderBehaviour
	if err := readOptionalJSON(filepath.Join(dataDir, prDownloaderBehaviourFile), &behaviour); err != nil {
		return 0, err
	}

	dir, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	call := prDownloaderCall{
		Args: args,
		Dir:  dir,
	}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "PRD_") {
			call.Env = append(call.Env, env)
		}
	}
	if err := appendJSONLine(filepath.Join(dataDir, prDownloaderCallsFile), call); err != nil {
		return 0, err
	}

	time.Sleep(behaviour.Sleep)
	//nolint:forbidigo
	fmt.Print(behaviour.Output)
	return behaviour.ExitCode, nil
}

// argumentValue returns the argument after name in args, or "" if there is none.
func argumentValue(args []string, name string) string {
	index := slices.Index(args, name)
	if index < 0 || index+1 >= len(args) {
		return ""
	}
	return args[index+1]
}

// relativeFiles returns the files in dir and its subfolders, relative to dir and slash separated.
func relativeFiles(dir string) ([]string, error) {
	var result []string
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relativePath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		result = append(result, filepath.ToSlash(relativePath))
		return nil
	})
	return result, err
}

// readOptionalJSON unmarshals the JSON file filePath into value. A missing file leaves value as it is.
func readOptionalJSON(filePath string, value any) error {
	data, err := os.ReadFile(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

// writeJSON writes value as JSON to filePath.
func writeJSON(filePath string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeFileForFake(filePath, data)
}

// appendJSONLine appends value as one JSON line to filePath.
func appendJSONLine(filePath string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	_, err = file.Write(append(data, '\n'))
	return errors.Join(err, file.Close())
}

// writeFileForFake writes content to filePath, creating its folder. It is writeFile for the fakes, which cannot use
// Gomega.
func writeFileForFake(filePath string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0o750); err != nil {
		return err
	}
	return os.WriteFile(filePath, content, 0o600)
}

// waitForFile waits until filePath exists, for at most timeout.
func waitForFile(filePath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if utils.FileExists(filePath) {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("%s did not appear within %s", filePath, timeout)
}

// writeFile writes content to filePath, creating its folder.
func writeFile(filePath string, content string) {
	Expect(writeFileForFake(filePath, []byte(content))).To(Succeed())
}

// readFile returns the content of filePath.
func readFile(filePath string) string {
	data, err := os.ReadFile(filePath)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

// newDataDir returns a new empty data dir.
func newDataDir() string {
	return GinkgoT().TempDir()
}

// newDataDirWithAssets returns a new data dir that has the engine (the fakes), map and game of the test replays.
func newDataDirWithAssets() string {
	dataDir := newDataDir()
	addFakeEngine(dataDir, testEngineVersion)
	addMapFile(dataDir, testMapFileName)
	addGame(dataDir, testGamePackage, testGameType)
	return dataDir
}

// addFakeEngine adds the engine engineVersion to dataDir with the test binary as its spring-headless and
// pr-downloader.
//
//nolint:unparam
func addFakeEngine(dataDir string, engineVersion string) {
	testBinary, err := os.Executable()
	Expect(err).NotTo(HaveOccurred())
	Expect(os.MkdirAll(utils.EngineVersionDir(dataDir, engineVersion), 0o750)).To(Succeed())
	Expect(os.Symlink(testBinary, utils.SpringHeadlessFilePath(dataDir, engineVersion))).To(Succeed())
	Expect(os.Symlink(testBinary, utils.PrDownloaderFilePath(dataDir, engineVersion))).To(Succeed())
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

// replaySibling returns the path of the file next to the replay replayFilePath that has suffix instead of the
// replay's extension.
func replaySibling(replayFilePath string, suffix string) string {
	return strings.TrimSuffix(replayFilePath, filepath.Ext(replayFilePath)) + suffix
}

// eventsFileContent returns an events file of formatVersion that ended for reason, or has no end if reason is "".
func eventsFileContent(formatVersion int, reason events.EndReason) string {
	file := map[string]any{
		"formatVersion": formatVersion,
	}
	if reason != "" {
		file["end"] = events.End{Frame: 48210, Reason: reason}
	}
	data, err := json.Marshal(file)
	Expect(err).NotTo(HaveOccurred())
	return string(data)
}

// currentEventsFileContent returns a complete events file of the embedded widget's format version.
func currentEventsFileContent() string {
	return eventsFileContent(widget.FormatVersion, events.EndReasonGameOver)
}

// writeSpringHeadlessBehaviour sets what the fake spring-headless does for the replay replayFilePath.
func writeSpringHeadlessBehaviour(replayFilePath string, behaviour springHeadlessBehaviour) {
	Expect(writeJSON(replaySibling(replayFilePath, springHeadlessBehaviourSuffix), behaviour)).To(Succeed())
}

// readSpringHeadlessCall returns what the fake spring-headless was called with for the replay replayFilePath.
func readSpringHeadlessCall(replayFilePath string) springHeadlessCall {
	var call springHeadlessCall
	Expect(json.Unmarshal([]byte(readFile(replaySibling(replayFilePath, springHeadlessCallSuffix))), &call)).
		To(Succeed())
	return call
}

// springHeadlessCalled returns whether the fake spring-headless was called for the replay replayFilePath.
func springHeadlessCalled(replayFilePath string) bool {
	return utils.FileExists(replaySibling(replayFilePath, springHeadlessCallSuffix))
}

// writePrDownloaderBehaviour sets what the fake pr-downloader does in dataDir.
func writePrDownloaderBehaviour(dataDir string, behaviour prDownloaderBehaviour) {
	Expect(writeJSON(filepath.Join(dataDir, prDownloaderBehaviourFile), behaviour)).To(Succeed())
}

// readPrDownloaderCalls returns the calls of the fake pr-downloader in dataDir, in order.
func readPrDownloaderCalls(dataDir string) []prDownloaderCall {
	data, err := os.ReadFile(filepath.Join(dataDir, prDownloaderCallsFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	Expect(err).NotTo(HaveOccurred())

	var calls []prDownloaderCall
	for line := range strings.Lines(string(data)) {
		var call prDownloaderCall
		Expect(json.Unmarshal([]byte(line), &call)).To(Succeed())
		calls = append(calls, call)
	}
	return calls
}

// testAssetManagerConfig returns DefaultAssetManagerConfig for dataDir that downloads no engine (see
// offlineEngineDownloadURL).
func testAssetManagerConfig(dataDir string) replayplayer.AssetManagerConfig {
	config := replayplayer.DefaultAssetManagerConfig
	config.DataDir = dataDir
	config.EngineDownloadURLTemplates = []string{offlineEngineDownloadURL}
	return config
}

// newAssetManager returns an AssetManager for dataDir that downloads no engine (see offlineEngineDownloadURL).
func newAssetManager(dataDir string) *replayplayer.AssetManager {
	assetManager, err := replayplayer.NewAssetManager(testAssetManagerConfig(dataDir))
	Expect(err).NotTo(HaveOccurred())
	return assetManager
}

// newAssetManagerWithEngineDownloads returns an AssetManager for dataDir that downloads engines from
// engineDownloadURLTemplates.
func newAssetManagerWithEngineDownloads(
	dataDir string,
	engineDownloadURLTemplates ...string,
) *replayplayer.AssetManager {
	config := testAssetManagerConfig(dataDir)
	config.EngineDownloadURLTemplates = engineDownloadURLTemplates
	assetManager, err := replayplayer.NewAssetManager(config)
	Expect(err).NotTo(HaveOccurred())
	return assetManager
}

// validConfig returns DefaultConfig with an asset manager for a new data dir, which Validate accepts.
func validConfig() replayplayer.Config {
	config := replayplayer.DefaultConfig
	config.AssetManager = newAssetManager(newDataDir())
	return config
}

// newPlayer returns a Player for dataDir with options.
func newPlayer(dataDir string, options ...replayplayer.Option) *replayplayer.Player {
	player, err := replayplayer.New(append([]replayplayer.Option{
		replayplayer.WithDataDir(dataDir),
		replayplayer.WithAssetManager(newAssetManager(dataDir)),
	}, options...)...)
	Expect(err).NotTo(HaveOccurred())
	return player
}

// newPreparedReplay writes a test replay as fileName into the demos/ of dataDir, prepares it with player and returns
// its ReplayInfo.
func newPreparedReplay(
	ctx context.Context,
	player *replayplayer.Player,
	dataDir string,
	fileName string, //nolint:unparam
) *replayplayer.ReplayInfo {
	replayInfo, err := replayplayer.ReadReplayInfo(writeTestReplay(demosFile(dataDir, fileName), testMapName))
	Expect(err).NotTo(HaveOccurred())
	Expect(player.Prepare(ctx, replayInfo)).To(Succeed())
	return replayInfo
}

// runPlayerInBackground runs player on replayInfo in a goroutine and returns the channel its error arrives on.
func runPlayerInBackground(
	ctx context.Context,
	player *replayplayer.Player,
	replayInfo *replayplayer.ReplayInfo,
) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- player.Run(ctx, replayInfo)
	}()
	return result
}

// newBatchPlayer returns a BatchPlayer for dataDir with options.
func newBatchPlayer(dataDir string, options ...replayplayer.Option) *replayplayer.BatchPlayer {
	batchPlayer, err := replayplayer.NewBatchPlayer(append([]replayplayer.Option{
		replayplayer.WithDataDir(dataDir),
		replayplayer.WithAssetManager(newAssetManager(dataDir)),
	}, options...)...)
	Expect(err).NotTo(HaveOccurred())
	return batchPlayer
}

// runBatchPlayerInBackground runs batchPlayer on paths in a goroutine and returns the channel its error arrives on.
func runBatchPlayerInBackground(
	ctx context.Context,
	batchPlayer *replayplayer.BatchPlayer,
	paths ...string,
) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- batchPlayer.Run(ctx, paths...)
	}()
	return result
}

// writePlayableReplay writes a test replay as fileName into the demos/ of dataDir on which the fake spring-headless
// writes a current events file, and returns its path.
func writePlayableReplay(dataDir string, fileName string, mapName string) string {
	replayFilePath := writeTestReplay(demosFile(dataDir, fileName), mapName)
	writeSpringHeadlessBehaviour(replayFilePath, springHeadlessBehaviour{EventsFile: currentEventsFileContent()})
	return replayFilePath
}

// ensureMapConcurrently calls EnsureMap of assetManager count times at once and returns their errors.
func ensureMapConcurrently(
	ctx context.Context,
	assetManager *replayplayer.AssetManager,
	engineVersion string,
	mapName string,
	count int,
) []error {
	errs := make([]error, count)
	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			errs[i] = assetManager.EnsureMap(ctx, engineVersion, mapName)
		})
	}
	wg.Wait()
	return errs
}

// copyTestdata copies the file name from testdata/ to dir and returns the copy's path.
func copyTestdata(name string, dir string) string {
	filePath := filepath.Join(dir, name)
	Expect(utils.CopyFile(filepath.Join("testdata", name), filePath)).To(Succeed())
	return filePath
}

// engineServer serves the engine archive testdata/engine.7z for requests under /engine/, a response that breaks off
// under /truncated/ and 404 for everything else. It records the request URIs.
type engineServer struct {
	mutex       sync.Mutex
	requestURIs []string
}

// ServeHTTP implements http.Handler.
func (s *engineServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.mutex.Lock()
	s.requestURIs = append(s.requestURIs, request.RequestURI)
	s.mutex.Unlock()

	switch {
	case strings.HasPrefix(request.URL.Path, "/engine/"):
		http.ServeFile(writer, request, filepath.Join("testdata", "engine.7z"))
	case strings.HasPrefix(request.URL.Path, "/truncated/"):
		writer.Header().Set("Content-Length", "1000")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("only a part"))
	default:
		http.NotFound(writer, request)
	}
}

// RequestURIs returns the request URIs the server received, in order.
func (s *engineServer) RequestURIs() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return slices.Clone(s.requestURIs)
}

// newEngineServer starts an engineServer, which the spec closes when it ends.
func newEngineServer() (*engineServer, *httptest.Server) {
	handler := &engineServer{}
	server := httptest.NewServer(handler)
	DeferCleanup(server.Close)
	return handler, server
}

// fileMode returns the permissions of filePath.
func fileMode(filePath string) os.FileMode {
	info, err := os.Stat(filePath)
	Expect(err).NotTo(HaveOccurred())
	return info.Mode().Perm()
}

// writeDirWidgetFiles returns the paths of the files of the embedded analytics widget in a write dir, relative to it
// and slash separated.
func writeDirWidgetFiles() []string {
	var result []string
	Expect(fs.WalkDir(widget.Template, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		result = append(result, filepath.ToSlash(filepath.Join(utils.AnalyticsWidgetDir("."), name)))
		return nil
	})).To(Succeed())
	return result
}
