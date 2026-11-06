package playback

import (
	"context"
	"net/http"
	"path/filepath"
	"sync"
)

// Player plays replays back in one data dir. Its methods may be called from several goroutines at once.
type Player struct {
	config     Config
	httpClient *http.Client

	// dataDir is the absolute path of the data dir.
	dataDir string

	// prepareMutex lets one Prepare run at a time, so a download is not run twice at once.
	prepareMutex sync.Mutex
}

// NewPlayer returns a Player for the data dir dataDir. The analytics widget need not be installed there: each playback
// gets the embedded one (see MakeWriteDir).
func NewPlayer(dataDir string, options ...Option) (*Player, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}

	absoluteDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return nil, err
	}

	return &Player{
		config:     config,
		httpClient: &http.Client{Transport: config.Transport},
		dataDir:    absoluteDataDir,
	}, nil
}

// DataDir returns the absolute path of the data dir.
func (p *Player) DataDir() string {
	return p.dataDir
}

// FindWidgetDir returns the subfolder of widgets (a LuaUI/Widgets/) that holds widgetFile, and whether there is one.
// BAR loads user widgets only one folder level deep.
func FindWidgetDir(widgets string, widgetFile string) (string, bool, error) {
	matches, err := filepath.Glob(filepath.Join(widgets, "*", widgetFile))
	if err != nil || len(matches) == 0 {
		return "", false, err
	}
	return filepath.Dir(matches[0]), true, nil
}

// Prepare reads the replay file filePath and finds or downloads its engine, game and map. Calls run one at a time.
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
