package playback

import (
	"net/http"
	"time"
)

// Config configures a Player. NewPlayer starts from DefaultConfig and applies its options to it.
type Config struct {
	// Transport executes the engine downloads.
	Transport http.RoundTripper

	// EngineCDNURL is the URL an engine is downloaded from first, with {platform} (runtime.GOOS) and {tag} (the
	// engine version, URL-escaped) replaced.
	EngineCDNURL string

	// EngineReleaseURL is the URL a RecoilEngine release tag (such as 2026.07.04) is downloaded from if EngineCDNURL
	// fails, with the same placeholders. Empty downloads from EngineCDNURL only.
	EngineReleaseURL string

	// PRDownloaderEnv is added to the environment of pr-downloader, which downloads games and maps.
	PRDownloaderEnv []string

	// SevenZip is the 7-Zip command line tool which unpacks engines. Empty selects the first of 7z, 7zz and 7za on
	// PATH.
	SevenZip string

	// StallTimeout is how long the events file (or, while the engine loads, its log) may stop growing before Play
	// stops the engine.
	StallTimeout time.Duration

	// StopTimeout is how long Play waits for the engine to exit after asking it to, before it kills the engine.
	StopTimeout time.Duration

	// WorkerThreadCount is the engine's WorkerThreadCount setting for a playback: the number of threads of its thread
	// pool, the main thread included. Zero leaves the engine's default, one thread per logical CPU.
	WorkerThreadCount int

	// ProgressInterval is how often Play calls OnProgress.
	ProgressInterval time.Duration

	// OnProgress, if set, is called by Play every ProgressInterval while the engine runs. It is called from the
	// goroutines of all running Play calls.
	OnProgress func(progress Progress)

	// OnDownload, if set, is called by Prepare when a download starts, with what is downloaded: the URL of an engine,
	// or "map <name>" and "game <name>" for pr-downloader.
	OnDownload func(description string)
}

// DefaultConfig is the configuration of a Player for BAR's servers.
var DefaultConfig = Config{
	Transport: http.DefaultTransport,

	// BAR's CDN, where the BAR lobby downloads engines from (BYAR-Chobby gui_steam_coop_handler.lua,
	// GetEngineDownloadUrl).
	EngineCDNURL: "https://files-cdn.beyondallreason.dev/download?category=engine_{platform}64&springname={tag}",
	// RecoilEngine's GitHub releases, which BAR's CDN is fed from.
	EngineReleaseURL: "https://github.com/beyond-all-reason/RecoilEngine/releases/download/{tag}/" +
		"recoil_{tag}_amd64-{platform}.7z",
	// The settings of the BAR launcher (BYAR-Chobby dist_cfg/config.json): BAR's servers, not SpringRTS's.
	PRDownloaderEnv: []string{
		"PRD_HTTP_SEARCH_URL=https://files-cdn.beyondallreason.dev/find",
		"PRD_RAPID_REPO_MASTER=https://repos-cdn.beyondallreason.dev/repos.gz",
		"PRD_RAPID_USE_STREAMER=false",
	},
	StallTimeout: time.Minute,
	StopTimeout:  10 * time.Second,

	// The engine's default of one thread per logical CPU is slower: a 1v1 replay of 27 minutes took 71 s on 32 CPUs
	// and used 5.2 cores, mostly waking and parking idle threads, against 43 s and 1.4 cores with 2 threads (50 s
	// with 1, 40 s with 4).
	WorkerThreadCount: 2,

	ProgressInterval: 5 * time.Second,
}
