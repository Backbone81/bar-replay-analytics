package playback

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
)

// AnalyticsWidgetFile is the file name of the analytics widget of bar-replay-analytics, which logs the playback.
const AnalyticsWidgetFile = "dbg_replay_analytics.lua"

// widgetsDir is the folder of the user widgets in a data dir, as BAR names it (WIDGET_DIRNAME in luaui/barwidgets.lua).
// The analytics widget's events folder is linked into each playback's write dir under the same path.
var widgetsDir = filepath.Join("LuaUI", "Widgets")

// Player plays replays back in one data dir. Its methods may be called from several goroutines at once.
type Player struct {
	config     Config
	httpClient *http.Client

	// dataDir is the absolute path of the data dir.
	dataDir string

	// analyticsDir is the folder in LuaUI/Widgets/ which holds the analytics widget.
	analyticsDir string

	// prepareMutex lets one Prepare run at a time, so a download is not run twice at once.
	prepareMutex sync.Mutex
}

// NewPlayer returns a Player for the data dir dataDir. It returns an error if the analytics widget is not installed
// there.
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
		return nil, fmt.Errorf("the analytics widget is not installed: no %s in a folder of %s; clone or link "+
			"bar-replay-analytics there", AnalyticsWidgetFile, widgets)
	}

	return &Player{
		config:       config,
		httpClient:   &http.Client{Transport: config.Transport},
		dataDir:      absoluteDataDir,
		analyticsDir: analyticsDir,
	}, nil
}

// DataDir returns the absolute path of the data dir.
func (p *Player) DataDir() string {
	return p.dataDir
}

// AnalyticsWidgetDir returns the folder in the data dir's LuaUI/Widgets/ which holds the analytics widget.
func (p *Player) AnalyticsWidgetDir() string {
	return p.analyticsDir
}

// EventsDir returns the folder the events and log files go to: events/ in the analytics widget's folder.
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
