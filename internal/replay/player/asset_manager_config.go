package player

import (
	"errors"
	"log/slog"
)

// AssetManagerConfig configures an AssetManager. Start from DefaultAssetManagerConfig and change what is needed.
type AssetManagerConfig struct {
	// Logger receives the downloads.
	Logger *slog.Logger

	// DataDir is the BAR data dir the engines, maps and games are looked up in and downloaded to.
	DataDir string

	// EngineDownloadURLTemplates are the URLs of an engine's release build, tried in order until one download works.
	// Each is a text/template template of EngineDownloadURLData.
	EngineDownloadURLTemplates []string
}

// EngineDownloadURLData is what an EngineDownloadURLTemplates template is executed with. Both values are plain
// identifiers that need no escaping in a URL: EnsureEngineVersion accepts only release tags.
type EngineDownloadURLData struct {
	// EngineVersion is the engine tag, e.g. 2026.07.04.
	EngineVersion string

	// OS is the operating system as Go names it (runtime.GOOS), e.g. linux or windows.
	OS string
}

// DefaultAssetManagerConfig is the AssetManagerConfig to start from. Its engine downloads try BAR's CDN first, which
// the BAR lobby uses, then the RecoilEngine release on GitHub.
var DefaultAssetManagerConfig = AssetManagerConfig{
	Logger: slog.New(slog.DiscardHandler),

	DataDir: "data",
	//nolint:lll
	EngineDownloadURLTemplates: []string{
		"https://files-cdn.beyondallreason.dev/download?category=engine_{{.OS}}64&springname={{.EngineVersion}}",
		"https://github.com/beyond-all-reason/RecoilEngine/releases/download/{{.EngineVersion}}/recoil_{{.EngineVersion}}_amd64-{{.OS}}.7z",
	},
}

// Validate returns an error if the AssetManagerConfig cannot be used.
func (c AssetManagerConfig) Validate() error {
	if c.Logger == nil {
		return errors.New("the logger is required")
	}
	if len(c.EngineDownloadURLTemplates) == 0 {
		return errors.New("at least one engine download URL template is required")
	}
	return nil
}
