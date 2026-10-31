package barlive

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"strconv"
	"time"
)

// MaxReplaysLimit is the largest ReplaysQuery.Limit bar-db accepts.
const MaxReplaysLimit = 100

// dateLayout is the format of dates in a ReplaysQuery.
const dateLayout = time.DateOnly

// ReplaysQuery selects replays. Every set field narrows the search; the zero value selects all replays.
//
// See src/model/rest-api/replays.ts and src/rest-api/routes/replays.ts in beyond-all-reason/bar-db.
type ReplaysQuery struct {
	// Page is the page to return, starting at 1. Zero selects bar-db's default of 1.
	Page int

	// Limit is the number of replays per page, at most MaxReplaysLimit. Zero selects bar-db's default of 24.
	Limit int

	// Preset selects replays of any of these presets.
	Preset []ReplayPreset

	// EndedNormally selects replays of games which ended normally (true) or were aborted (false).
	EndedNormally *bool

	// HasBots selects replays with (true) or without (false) an AI.
	HasBots *bool

	// Reported selects replays in which someone wrote "!report" in the chat (true) or nobody did (false).
	Reported *bool

	// Players selects replays in which all of these players played; spectators do not count. bar-db looks the names
	// up as current account names, see LeaderboardPlayer.Name, and finds a player's games under earlier names too. An
	// unknown name selects no replay.
	Players []string

	// Maps selects replays on any of these maps, by Map.ScriptName.
	Maps []string

	// Date selects replays of games which started in this range, or nil for any date.
	Date *DateRange

	// DurationRange selects replays of games whose Replay.DurationMs lies in this range, or nil for any duration.
	// bar-db names it durationRangeMins, as it receives minutes.
	DurationRange *DurationRange

	// TSRange selects replays in which every player's Player.TrueSkill lies in this range, or nil for any skill.
	// Current replays have no TrueSkill, so this selects only old ones.
	TSRange *TrueSkillRange

	// ComputeTotalResults makes bar-db count all selected replays into ReplaysPage.TotalResults.
	ComputeTotalResults bool
}

// DateRange is a range of days in UTC, both ends inclusive. Only the dates of From and To count, not their times or
// locations.
type DateRange struct {
	// From is the first day.
	From time.Time

	// To is the last day. If it is the zero time, the range is the day From.
	To time.Time
}

// DurationRange is a range of game durations, both ends inclusive. bar-db receives them in minutes, which may have
// fractions.
type DurationRange struct {
	Min time.Duration
	Max time.Duration
}

// TrueSkillRange is a range of Player.TrueSkill values, both ends inclusive.
type TrueSkillRange struct {
	Min float64
	Max float64
}

// Values returns the query as bar-db expects it: a list repeats its key, a range is a list of two values.
func (q ReplaysQuery) Values() url.Values {
	values := make(url.Values)
	if q.Page != 0 {
		values.Set("page", strconv.Itoa(q.Page))
	}
	if q.Limit != 0 {
		values.Set("limit", strconv.Itoa(q.Limit))
	}
	for _, preset := range q.Preset {
		values.Add("preset", string(preset))
	}
	if q.EndedNormally != nil {
		values.Set("endedNormally", strconv.FormatBool(*q.EndedNormally))
	}
	if q.HasBots != nil {
		values.Set("hasBots", strconv.FormatBool(*q.HasBots))
	}
	if q.Reported != nil {
		values.Set("reported", strconv.FormatBool(*q.Reported))
	}
	for _, player := range q.Players {
		values.Add("players", player)
	}
	for _, mapName := range q.Maps {
		values.Add("maps", mapName)
	}
	if q.Date != nil {
		// bar-db selects the games which started from the midnight of the first date to the midnight of the second.
		lastDay := q.Date.To
		if lastDay.IsZero() {
			lastDay = q.Date.From
		}
		values.Add("date", q.Date.From.Format(dateLayout))
		values.Add("date", lastDay.AddDate(0, 0, 1).Format(dateLayout))
	}
	if q.DurationRange != nil {
		values.Add("durationRangeMins", strconv.FormatFloat(q.DurationRange.Min.Minutes(), 'f', -1, 64))
		values.Add("durationRangeMins", strconv.FormatFloat(q.DurationRange.Max.Minutes(), 'f', -1, 64))
	}
	if q.TSRange != nil {
		values.Add("tsRange", strconv.FormatFloat(q.TSRange.Min, 'f', -1, 64))
		values.Add("tsRange", strconv.FormatFloat(q.TSRange.Max, 'f', -1, 64))
	}
	if q.ComputeTotalResults {
		values.Set("computeTotalResults", "true")
	}
	return values
}

// ReplaysPage is one page of replays.
type ReplaysPage struct {
	// TotalResults is the number of all selected replays if ReplaysQuery.ComputeTotalResults was set, else -1.
	TotalResults int `json:"totalResults"`

	// Page is the number of this page, starting at 1.
	Page int `json:"page"`

	// Limit is the number of replays per page.
	Limit int `json:"limit"`

	// Data holds the replays of this page, newest first. A page after the last one is empty.
	Data []ReplaySummary `json:"data"`
}

// ReplaySummary is a replay as a search returns it. Client.Replay returns all its details.
type ReplaySummary struct {
	// ID is Replay.ID.
	ID string `json:"id"`

	// StartTime is when the game started.
	StartTime time.Time `json:"startTime"`

	// DurationMs is the length of the game in milliseconds.
	DurationMs int64 `json:"durationMs"`

	// Map is the map the game was played on; only its ScriptName and FileName are set.
	Map Map `json:"Map"` //nolint:tagliatelle // bar-db's name

	// AllyTeams holds the allyteams, in no particular order; only their WinningTeam, Players and AIs are set.
	AllyTeams []AllyTeam `json:"AllyTeams"` //nolint:tagliatelle // bar-db's name
}

// Duration returns DurationMs as a time.Duration.
func (r *ReplaySummary) Duration() time.Duration {
	return time.Duration(r.DurationMs) * time.Millisecond
}

// Replays returns one page of the replays selected by query.
func (c *Client) Replays(ctx context.Context, query ReplaysQuery) (*ReplaysPage, error) {
	if query.Limit < 0 || query.Limit > MaxReplaysLimit {
		return nil, fmt.Errorf("limit: got %d, want 0 to %d", query.Limit, MaxReplaysLimit)
	}
	if query.Page < 0 {
		return nil, fmt.Errorf("page: got %d, want >= 0", query.Page)
	}

	requestURL, err := url.JoinPath(c.config.ReplayAPIURL, "/replays")
	if err != nil {
		return nil, err
	}
	if values := query.Values(); len(values) != 0 {
		requestURL += "?" + values.Encode()
	}

	var page ReplaysPage
	if err := c.getJSON(ctx, requestURL, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// AllReplays returns every replay selected by query, newest first, from query.Page on. It requests one page after the
// other as the caller iterates, and stops after the first error, which it yields with a zero ReplaySummary. Each
// iteration starts again at query.Page.
//
// The pages are requested one at a time, so a replay may be returned twice or skipped if bar-db stores a new replay
// while the caller iterates.
func (c *Client) AllReplays(ctx context.Context, query ReplaysQuery) iter.Seq2[ReplaySummary, error] {
	return func(yield func(ReplaySummary, error) bool) {
		// Each iteration pages through its own copy, so the next one starts at query.Page again.
		pageQuery := query
		if pageQuery.Page == 0 {
			pageQuery.Page = 1
		}
		for {
			page, err := c.Replays(ctx, pageQuery)
			if err != nil {
				yield(ReplaySummary{}, err)
				return
			}
			for _, replay := range page.Data {
				if !yield(replay, nil) {
					return
				}
			}
			// A short page is the last one, which saves requesting the empty page after it.
			if len(page.Data) == 0 || len(page.Data) < page.Limit {
				return
			}
			pageQuery.Page++
		}
	}
}
