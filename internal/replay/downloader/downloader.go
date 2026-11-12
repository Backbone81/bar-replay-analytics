package downloader

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
	"github.com/backbone81/bar-replay-analytics-go/internal/replay/index"
)

type Downloader struct {
	config Config
}

func New(options ...Option) (*Downloader, error) {
	config := DefaultConfig
	for _, option := range options {
		option(&config)
	}
	if config.Client == nil {
		config.Client = barlive.NewClient(config.Logger)
	}
	if config.Index == nil {
		index, err := index.Open(filepath.Join(config.Dir, "index.json"))
		if err != nil {
			return nil, err
		}
		config.Index = index
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return &Downloader{
		config: config,
	}, nil
}

func (d *Downloader) Run(ctx context.Context) error {
	d.config.Logger.Info("Fetching current season")
	season, err := d.config.Client.LeaderboardSeason(ctx, 0)
	if err != nil {
		return err
	}

	var matchedGameTypes int
	for _, gameType := range season.GameTypes {
		if gameType != d.config.GameType {
			continue
		}

		matchedGameTypes++
		if err := d.fetchLeaderboard(ctx, d.config.Logger, season); err != nil {
			return err
		}
	}
	if matchedGameTypes == 0 {
		return errors.New("the requested game type did not match any game type available in the season")
	}
	return nil
}

func (d *Downloader) Close() error {
	if d.config.DryRun {
		// Exit early to not modify the index during dry-run.
		return nil
	}
	return d.config.Index.Save()
}

func (d *Downloader) fetchLeaderboard(
	ctx context.Context,
	logger *slog.Logger,
	season barlive.LeaderboardSeason,
) error {
	preset, found := d.config.GameType.ReplayPreset()
	if !found {
		return fmt.Errorf("no preset for game type %q", d.config.GameType)
	}

	logger = logger.With(
		"season", season.Season,
		"game-type", d.config.GameType,
	)
	logger.Info("Fetching leaderboard")
	leaderboard, err := d.config.Client.Leaderboard(ctx, season.Season, d.config.GameType)
	if err != nil {
		return err
	}

	var joinedErrs error
	for _, player := range leaderboard.Players[:min(d.config.Players, len(leaderboard.Players))] {
		if err := d.fetchPlayerReplays(ctx, logger, player, preset); err != nil {
			// We only abort the current player on errors. Downloads might be so far into the past that replays listed
			// by the API are not available for download anymore (more than 12 months old).
			logger.Error(err.Error())
			joinedErrs = errors.Join(joinedErrs, err)

			// If the context was canceled, don't try the next player.
			if ctx.Err() != nil {
				return ctx.Err()
			}
		}
	}
	return joinedErrs
}

func (d *Downloader) fetchPlayerReplays(
	ctx context.Context,
	logger *slog.Logger,
	player barlive.LeaderboardPlayer,
	preset barlive.ReplayPreset,
) error {
	query := barlive.ReplaysQuery{
		Page:          1,
		Limit:         barlive.MaxReplaysLimit,
		Preset:        []barlive.ReplayPreset{preset},
		EndedNormally: new(true),
		HasBots:       new(false),
		Players:       []string{player.Name},
		Maps:          d.config.Maps,
		Date: &barlive.DateRange{
			From: time.Now().UTC().Add(-time.Duration(d.config.Days) * 24 * time.Hour),
			To:   time.Now().UTC(),
		},
	}
	logger = logger.With(
		"player", player.Name,
		"rating", fmt.Sprintf("%.1f", player.Rating),
	)
	for {
		count, err := d.fetchPlayerReplaysPage(ctx, logger, query)
		if err != nil {
			return err
		}
		if count < query.Limit {
			break
		}
		query.Page++
	}
	return nil
}

func (d *Downloader) fetchPlayerReplaysPage(
	ctx context.Context,
	logger *slog.Logger,
	query barlive.ReplaysQuery,
) (int, error) {
	logger = logger.With(
		"page", query.Page,
	)
	logger.Info("Fetching replays")
	page, err := d.config.Client.Replays(ctx, query)
	if err != nil {
		return 0, err
	}

	for _, replay := range page.Data {
		if !d.config.GameType.Matches(&replay) {
			continue
		}
		if err := d.fetchPlayerReplay(ctx, logger, replay); err != nil {
			return 0, err
		}
	}
	return len(page.Data), nil
}

func (d *Downloader) fetchPlayerReplay(
	ctx context.Context,
	logger *slog.Logger,
	replaySummary barlive.ReplaySummary,
) error {
	logger = logger.With(
		"id", replaySummary.ID,
		"date", replaySummary.StartTime.Format("02.01.2006"),
		"duration", replaySummary.Duration(),
		"map", replaySummary.Map.ScriptName,
	)

	// We use the replay index to reduce our work of verifying if the replay is already on disk.
	fileName, found := d.config.Index.Get(replaySummary.ID)
	if found {
		// We found the replay in the index, but the corresponding file might be missing from disk. We therefore check
		// if the file is indeed on disk.
		if _, err := os.Stat(filepath.Join(d.config.Dir, fileName)); err == nil {
			logger.Info("Replay already available on disk")
			return nil
		}
		logger.Info("Replay listed in index but file is missing. Removing from index.")
		d.config.Index.Remove(replaySummary.ID)
	}

	// The replay might be missing from the replay index, but the file is already on disk. We need the replay file name
	// to check if the file is already there.
	replay, err := d.config.Client.Replay(ctx, replaySummary.ID)
	if err != nil {
		return err
	}

	if err := barlive.ValidateDemoFileName(replay.FileName); err != nil {
		// The file name did contain unexpected characters. Reject it.
		return err
	}
	if _, err := os.Stat(filepath.Join(d.config.Dir, replay.FileName)); err == nil {
		logger.Info("Replay file already on disk but missing from index. Adding to index.")
		d.config.Index.Add(replay.ID, replay.FileName)
		return nil
	}

	if d.config.DryRun {
		// Exit early for dry runs which should not download anything
		logger.Info("Would download replay", "file", replay.FileName)
		return nil
	}

	logger.Info("Downloading replay", "file", replay.FileName)
	//nolint:gosec // This folder does not contain confidential data.
	if err := os.MkdirAll(d.config.Dir, 0o755); err != nil {
		return err
	}
	if _, err := d.config.Client.DownloadDemoIntoDir(ctx, replay, d.config.Dir); err != nil {
		return err
	}
	d.config.Index.Add(replay.ID, replay.FileName)
	return nil
}
