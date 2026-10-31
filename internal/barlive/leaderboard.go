package barlive

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// LeaderboardSeason is a season for which Teiserver has leaderboards.
type LeaderboardSeason struct {
	// Season is the number of the season.
	Season int `json:"season"`

	// GameTypes lists the game types with a leaderboard in this season.
	GameTypes []GameType `json:"game_types"` //nolint:tagliatelle // Teiserver's name
}

// Leaderboard ranks the best players of one game type in one season.
type Leaderboard struct {
	// Name is the game type.
	Name GameType `json:"name"`

	// Players holds at most the 100 best players ordered by Rating descending. Teiserver lists only players whose rating
	// changed in the 35 days before now (for the current season) or before the season's last rating change (for an
	// earlier season), so the list may be shorter.
	Players []LeaderboardPlayer `json:"players"`
}

// LeaderboardPlayer is a player on a Leaderboard.
type LeaderboardPlayer struct {
	// ID is the player's Teiserver account ID, which bar-db calls Player.UserID.
	ID int `json:"id"`

	// Name is the player's current account name, which ReplaysQuery.Players searches for.
	Name string `json:"name"`

	// Rating is the value the leaderboard is sorted by.
	Rating float64 `json:"rating"`
}

// LeaderboardSeasons returns the seasons for which Teiserver has leaderboards ordered ascending.
func (c *Client) LeaderboardSeasons(ctx context.Context) ([]LeaderboardSeason, error) {
	leaderboardURL, err := url.JoinPath(c.config.TeiserverURL, "/teiserver/api/public/leaderboard")
	if err != nil {
		return nil, err
	}

	var seasons []LeaderboardSeason
	if err := c.getJSON(ctx, leaderboardURL, &seasons); err != nil {
		return nil, err
	}
	return seasons, nil
}

// Leaderboards returns the leaderboard of each game type in season, which starts at 1.
func (c *Client) Leaderboards(ctx context.Context, season int) ([]Leaderboard, error) {
	if season <= 0 {
		return nil, fmt.Errorf("season: got %d, want >= 1", season)
	}

	seasonURL, err := url.JoinPath(c.config.TeiserverURL, "/teiserver/api/public/leaderboard/", strconv.Itoa(season))
	if err != nil {
		return nil, err
	}

	var leaderboards []Leaderboard
	if err := c.getJSON(ctx, seasonURL, &leaderboards); err != nil {
		return nil, err
	}
	return leaderboards, nil
}

// FindLeaderboard returns the leaderboard of gameType from leaderboards, or false if there is none.
func FindLeaderboard(leaderboards []Leaderboard, gameType GameType) (Leaderboard, bool) {
	for _, leaderboard := range leaderboards {
		if leaderboard.Name == gameType {
			return leaderboard, true
		}
	}
	return Leaderboard{}, false
}

// LeaderboardSeason returns the season with the number season, or the active season if season is 0. Teiserver lists
// the seasons from 1 up to the active one, so the active season is the last one.
func (c *Client) LeaderboardSeason(ctx context.Context, season int) (LeaderboardSeason, error) {
	if season < 0 {
		return LeaderboardSeason{}, fmt.Errorf("season: got %d, want >= 0", season)
	}

	seasons, err := c.LeaderboardSeasons(ctx)
	if err != nil {
		return LeaderboardSeason{}, err
	}
	if len(seasons) == 0 {
		return LeaderboardSeason{}, errors.New("teiserver lists no seasons")
	}
	if season == 0 {
		return seasons[len(seasons)-1], nil
	}
	for _, leaderboardSeason := range seasons {
		if leaderboardSeason.Season == season {
			return leaderboardSeason, nil
		}
	}
	return LeaderboardSeason{}, fmt.Errorf("season %d: teiserver lists seasons 1 to %d", season,
		seasons[len(seasons)-1].Season)
}

// Leaderboard returns the leaderboard of gameType in season, which starts at 1.
func (c *Client) Leaderboard(ctx context.Context, season int, gameType GameType) (Leaderboard, error) {
	leaderboards, err := c.Leaderboards(ctx, season)
	if err != nil {
		return Leaderboard{}, err
	}
	leaderboard, found := FindLeaderboard(leaderboards, gameType)
	if !found {
		return Leaderboard{}, fmt.Errorf("season %d has no leaderboard for game type %q", season, gameType)
	}
	return leaderboard, nil
}
