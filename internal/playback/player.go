package playback

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
)

// AnalyticsWidgetFile is the file name of the analytics widget of bar-replay-analytics, which logs the playback.
const AnalyticsWidgetFile = "dbg_replay_analytics.lua"

// analyticsWidgetFolder is the folder in LuaUI/Widgets/ the analytics widget goes into if it is not installed in the
// data dir: the name of the repository it comes from.
const analyticsWidgetFolder = "bar-replay-analytics"

// widgetsDir is the folder of the user widgets in a data dir, as BAR names it (WIDGET_DIRNAME in luaui/barwidgets.lua).
// Each playback's write dir gets the analytics widget and a link to its events folder under the same path.
var widgetsDir = filepath.Join("LuaUI", "Widgets")

// Player plays replays back in one data dir. Its methods may be called from several goroutines at once.
type Player struct {
	config     Config
	httpClient *http.Client

	// dataDir is the absolute path of the data dir.
	dataDir string

	// analyticsDir is the folder in the data dir's LuaUI/Widgets/ which holds the installed analytics widget, or
	// LuaUI/Widgets/bar-replay-analytics/ if none is installed. Its events/ holds the events and log files.
	analyticsDir string

	// analyticsInstalled is whether the analytics widget is installed in analyticsDir.
	analyticsInstalled bool

	// prepareMutex lets one Prepare run at a time, so a download is not run twice at once.
	prepareMutex sync.Mutex
}

// NewPlayer returns a Player for the data dir dataDir. The analytics widget need not be installed there: each playback
// gets the embedded one (see Play).
func NewPlayer(dataDir string, options ...Option) (*Player, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}

	absoluteDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}

	widgets := filepath.Join(absoluteDataDir, widgetsDir)
	analyticsDir, found, err := FindWidgetDir(widgets, AnalyticsWidgetFile)
	if err != nil {
		return nil, err
	}
	if !found {
		analyticsDir = filepath.Join(widgets, analyticsWidgetFolder)
	}

	return &Player{
		config:             config,
		httpClient:         &http.Client{Transport: config.Transport},
		dataDir:            absoluteDataDir,
		analyticsDir:       analyticsDir,
		analyticsInstalled: found,
	}, nil
}

// DataDir returns the absolute path of the data dir.
func (p *Player) DataDir() string {
	return p.dataDir
}

// AnalyticsWidgetDir returns the folder in the data dir's LuaUI/Widgets/ which holds the installed analytics widget,
// and whether one is installed. If none is, it returns LuaUI/Widgets/bar-replay-analytics/, where the events go.
func (p *Player) AnalyticsWidgetDir() (string, bool) {
	return p.analyticsDir, p.analyticsInstalled
}

// EventsDir returns the folder the events and log files go to: events/ in the analytics widget's folder (see
// AnalyticsWidgetDir).
func (p *Player) EventsDir() string {
	return filepath.Join(p.analyticsDir, "events")
}

// FindWidgetDir returns the folder in widgets (a LuaUI/Widgets/) which holds widgetFile, and whether there is one.
// BAR loads user widgets one folder level deep, so that is where it looks.
func FindWidgetDir(widgets string, widgetFile string) (string, bool, error) {
	matches, err := filepath.Glob(filepath.Join(widgets, "*", widgetFile))
	if err != nil || len(matches) == 0 {
		return "", false, err
	}
	return filepath.Dir(matches[0]), true, nil
}

// Prepare reads the replay file filePath, and finds or downloads its engine, game and map. Only one Prepare of the
// Player runs at a time.
func (p *Player) Prepare(ctx context.Context, filePath string) (*Replay, error) {
	absoluteFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}
	replay, err := readReplay(absoluteFilePath)
	if err != nil {
		return nil, err
	}

	p.prepareMutex.Lock()
	defer p.prepareMutex.Unlock()

	if replay.Engine, err = p.findOrDownloadEngine(ctx, replay.EngineVersion); err != nil {
		return nil, err
	}
	if replay.GamePackage, err = p.findOrDownloadGame(ctx, replay.GameType, replay.Engine); err != nil {
		return nil, err
	}
	if replay.MapArchive, err = p.findOrDownloadMap(ctx, replay.MapName, replay.Engine); err != nil {
		return nil, err
	}
	return replay, nil
}

// reportDownload calls Config.OnDownload, if set.
func (p *Player) reportDownload(description string) {
	if p.config.OnDownload != nil {
		p.config.OnDownload(description)
	}
}

// reportProgress calls Config.OnProgress, if set.
func (p *Player) reportProgress(progress Progress) {
	if p.config.OnProgress != nil {
		p.config.OnProgress(progress)
	}
}
