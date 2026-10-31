package barlive

import (
	"context"
	"net/url"
	"strconv"
)

// The game types of the leaderboards, as Teiserver names them in LeaderboardSeason.GameTypes and Leaderboard.Name.
const (
	GameTypeDuel      = "Duel"
	GameTypeSmallTeam = "Small Team"
	GameTypeLargeTeam = "Large Team"
	GameTypeFFA       = "FFA"
)

// LeaderboardSeason is a season for which Teiserver has leaderboards.
type LeaderboardSeason struct {
	// Season is the number of the season.
	Season int `json:"season"`

	// GameTypes lists the game types with a leaderboard in this season, see the GameType constants.
	GameTypes []string `json:"game_types"` //nolint:tagliatelle // Teiserver's name
}

// Leaderboard ranks the best players of one game type in one season.
type Leaderboard struct {
	// Name is the game type, see the GameType constants.
	Name string `json:"name"`

	// Players holds the top 100 players ordered descending.
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
	leaderboardUrl, err := url.JoinPath(c.config.TeiserverURL, "/teiserver/api/public/leaderboard")
	if err != nil {
		return nil, err
	}

	var seasons []LeaderboardSeason
	if err := c.getJSON(ctx, leaderboardUrl, &seasons); err != nil {
		return nil, err
	}
	return seasons, nil
}

// Leaderboards returns the leaderboard of each game type in season.
func (c *Client) Leaderboards(ctx context.Context, season int) ([]Leaderboard, error) {
	seasonUrl, err := url.JoinPath(c.config.TeiserverURL, "/teiserver/api/public/leaderboard/", strconv.Itoa(season))
	if err != nil {
		return nil, err
	}

	var leaderboards []Leaderboard
	if err := c.getJSON(ctx, seasonUrl, &leaderboards); err != nil {
		return nil, err
	}
	return leaderboards, nil
}

// FindLeaderboard returns the leaderboard of gameType from leaderboards, or false if there is none.
func FindLeaderboard(leaderboards []Leaderboard, gameType string) (Leaderboard, bool) {
	for _, leaderboard := range leaderboards {
		if leaderboard.Name == gameType {
			return leaderboard, true
		}
	}
	return Leaderboard{}, false
}
