package barlive

import (
	"context"
	"errors"
	"net/url"
	"time"
)

// ReplayPreset is the kind of game, see Replay.Preset.
type ReplayPreset string

// The presets bar-db knows.
const (
	ReplayPresetDuel ReplayPreset = "duel"
	ReplayPresetTeam ReplayPreset = "team"
	ReplayPresetFFA  ReplayPreset = "ffa"
)

// Replay is a replay as bar-db stores it, without the chat log. bar-db reads it from the replay file: most fields
// mirror the replay's setup script and statistics.
//
// The ID fields of bar-db's own records (Player.ID, AllyTeam.ID and so on) are not engine numbers; the engine numbers
// are Player.PlayerID, Player.TeamID, AllyTeam.AllyTeamID and AI.AIID.
//
// See src/model/db/demo.ts in beyond-all-reason/bar-db.
type Replay struct {
	// ID is the game ID from the replay file's header in lowercase hex, as demo.DemoFileHeader.GameIDAsString returns
	// it.
	ID string `json:"id"`

	// FileName is the name of the replay file, see Client.DownloadDemo.
	FileName string `json:"fileName"`

	// EngineVersion is the engine version, a Recoil release tag such as "2026.07.04".
	EngineVersion string `json:"engineVersion"`

	// GameVersion is the game version, the gametype of the setup script.
	GameVersion string `json:"gameVersion"`

	// StartTime is when the game started.
	StartTime time.Time `json:"startTime"`

	// DurationMs is the length of the game in milliseconds.
	DurationMs int64 `json:"durationMs"`

	// FullDurationMs is the length of the replay in milliseconds, including the time before the game started.
	FullDurationMs int64 `json:"fullDurationMs"`

	// HostSettings holds the values of the setup script's game section.
	HostSettings map[string]string `json:"hostSettings"`

	// GameSettings holds the values of the setup script's modoptions section.
	GameSettings map[string]string `json:"gameSettings"`

	// MapSettings holds the values of the setup script's mapoptions section.
	MapSettings map[string]string `json:"mapSettings"`

	// SpadsSettings holds the settings of the SPADS autohost, or nil if the game was not hosted by SPADS.
	SpadsSettings map[string]string `json:"spadsSettings"`

	// GameEndedNormally is true if the replay names a winning allyteam; false for a game without a winner, such as an
	// aborted or crashed one.
	GameEndedNormally bool `json:"gameEndedNormally"`

	// HasBots is true if an AI took part.
	HasBots bool `json:"hasBots"`

	// Preset is "duel", "team" or "ffa", see the ReplayPreset constants. bar-db takes it from SLDB's match data if it
	// has any, else "ffa" for more than 2 allyteams, "team" for more than 2 players and AIs, and "duel" otherwise.
	Preset ReplayPreset `json:"preset"`

	// Reported is true if a chat message contains "!report".
	Reported bool `json:"reported"`

	// Awards holds the awards the game sent at its end, or nil if the replay has none.
	Awards *Awards `json:"awards"`

	// CreatedAt is when bar-db stored the replay.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when bar-db last changed the replay.
	UpdatedAt time.Time `json:"updatedAt"`

	// MapID is Map.ID.
	MapID int `json:"mapId"`

	// Map is the map the game was played on.
	Map Map `json:"Map"` //nolint:tagliatelle // bar-db's name

	// AllyTeams holds the allyteams in order of AllyTeam.AllyTeamID.
	AllyTeams []AllyTeam `json:"AllyTeams"` //nolint:tagliatelle // bar-db's name

	// Spectators holds the spectators.
	Spectators []Spectator `json:"Spectators"` //nolint:tagliatelle // bar-db's name
}

// Duration returns DurationMs as a time.Duration.
func (r *Replay) Duration() time.Duration {
	return time.Duration(r.DurationMs) * time.Millisecond
}

// Map is a map as bar-db knows it.
type Map struct {
	// ID identifies the map in bar-db. Not set in a ReplaySummary.
	ID int `json:"id"`

	// ScriptName is the map's name in the setup script, such as "Cape_Violet_V1".
	ScriptName string `json:"scriptName"`

	// FileName is the map archive's name without the extension, in lower case, such as "cape_violet_v1".
	FileName string `json:"fileName"`

	// Width is the map's width in map units (512 elmos). Not set in a ReplaySummary.
	Width int `json:"width"`

	// Height is the map's height in map units (512 elmos). Not set in a ReplaySummary.
	Height int `json:"height"`
}

// AllyTeam is an allyteam: the players and AIs on one side.
//
// A ReplaySummary sets only WinningTeam, Players and AIs.
type AllyTeam struct {
	// ID identifies the allyteam in bar-db.
	ID int `json:"id"`

	// AllyTeamID is the allyteam's number in the game.
	AllyTeamID int `json:"allyTeamId"`

	// StartBox is the area the allyteam could start in, or nil if there was none.
	StartBox *StartBox `json:"startBox"`

	// WinningTeam is true if this allyteam won (the first winning allyteam of the replay).
	WinningTeam bool `json:"winningTeam"`

	// CreatedAt is when bar-db stored the allyteam.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when bar-db last changed the allyteam.
	UpdatedAt time.Time `json:"updatedAt"`

	// DemoID is Replay.ID.
	DemoID string `json:"demoId"`

	// Players holds the human players of the allyteam.
	Players []Player `json:"Players"` //nolint:tagliatelle // bar-db's name

	// AIs holds the AIs of the allyteam.
	AIs []AI `json:"AIs"` //nolint:tagliatelle // bar-db's name
}

// StartBox is a start area as fractions of the map size, from 0 to 1, as in the setup script.
type StartBox struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
}

// Player is a human player of an allyteam.
//
// A ReplaySummary sets only Name.
type Player struct {
	// ID identifies the player record in bar-db.
	ID int `json:"id"`

	// PlayerID is the player's number in the game.
	PlayerID int `json:"playerId"`

	// Name is the player's name in this game.
	Name string `json:"name"`

	// TeamID is the number of the team the player controlled.
	TeamID int `json:"teamId"`

	// Handicap is the resource bonus of the player's team in percent.
	Handicap int `json:"handicap"`

	// Faction is the player's side, such as "Armada" or "Cortex".
	Faction string `json:"faction"`

	// CountryCode is the player's country as an ISO 3166-1 alpha-2 code.
	CountryCode string `json:"countryCode"`

	// RGBColor is the player's team color, or nil if bar-db has none.
	RGBColor *Color `json:"rgbColor"`

	// Rank is the player's lobby rank, or nil if bar-db has none.
	Rank *int `json:"rank"`

	// SkillUncertainty is the uncertainty of Skill, or nil if bar-db has none.
	SkillUncertainty *float64 `json:"skillUncertainty"`

	// Skill is the player's skill as the setup script holds it, such as "[67.41]".
	Skill string `json:"skill"`

	// TrueSkill is Skill as a number, or nil if Skill is not a plain number: bar-db converts it with Number(), which
	// fails for the bracketed values of current replays.
	TrueSkill *float64 `json:"trueSkill"`

	// StartPos is where the player's commander started, or nil if bar-db has no position.
	StartPos *Position `json:"startPos"`

	// ClanID is the player's clan, or nil if the player has none.
	ClanID *string `json:"clanId"`

	// TrueSkillMuBefore is the player's skill before the game from SLDB, or nil if SLDB has none.
	TrueSkillMuBefore *float64 `json:"trueSkillMuBefore"`

	// TrueSkillSigmaBefore is the uncertainty of TrueSkillMuBefore, or nil if bar-db has none.
	TrueSkillSigmaBefore *float64 `json:"trueSkillSigmaBefore"`

	// TrueSkillMuAfter is the player's skill after the game from SLDB, or nil if SLDB has none.
	TrueSkillMuAfter *float64 `json:"trueSkillMuAfter"`

	// TrueSkillSigmaAfter is the uncertainty of TrueSkillMuAfter, or nil if bar-db has none.
	TrueSkillSigmaAfter *float64 `json:"trueSkillSigmaAfter"`

	// CreatedAt is when bar-db stored the player.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when bar-db last changed the player.
	UpdatedAt time.Time `json:"updatedAt"`

	// AllyTeamRecordID is AllyTeam.ID, not the allyteam's number in the game.
	AllyTeamRecordID int `json:"allyTeamId"`

	// UserID is the player's Teiserver account ID, which the leaderboards call LeaderboardPlayer.ID.
	UserID int `json:"userId"`
}

// AI is an AI of an allyteam.
//
// A ReplaySummary sets only ShortName.
type AI struct {
	// ID identifies the AI record in bar-db.
	ID int `json:"id"`

	// AIID is the AI's number in the game.
	AIID int `json:"aiId"`

	// ShortName is the AI's library name, such as "BARb".
	ShortName string `json:"shortName"`

	// Name is the AI's name in this game, such as "BARbarianAI(1)".
	Name string `json:"name"`

	// Host is true if the AI ran on the host.
	Host bool `json:"host"`

	// StartPos is where the AI's commander started, or nil if bar-db has no position.
	StartPos *Position `json:"startPos"`

	// Handicap is the resource bonus of the AI's team in percent.
	Handicap int `json:"handicap"`

	// Faction is the AI's side, such as "Armada", "Cortex" or "Random".
	Faction string `json:"faction"`

	// RGBColor is the AI's team color, or nil if bar-db has none.
	RGBColor *Color `json:"rgbColor"`

	// CreatedAt is when bar-db stored the AI.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when bar-db last changed the AI.
	UpdatedAt time.Time `json:"updatedAt"`

	// AllyTeamRecordID is AllyTeam.ID, not the allyteam's number in the game.
	AllyTeamRecordID int `json:"allyTeamId"`
}

// Spectator is a spectator of a game.
type Spectator struct {
	// ID identifies the spectator record in bar-db.
	ID int `json:"id"`

	// PlayerID is the spectator's player number in the game.
	PlayerID int `json:"playerId"`

	// Name is the spectator's name in this game.
	Name string `json:"name"`

	// CountryCode is the spectator's country as an ISO 3166-1 alpha-2 code.
	CountryCode string `json:"countryCode"`

	// Rank is the spectator's lobby rank, or nil if bar-db has none.
	Rank *int `json:"rank"`

	// SkillUncertainty is the uncertainty of Skill, or nil if bar-db has none.
	SkillUncertainty *float64 `json:"skillUncertainty"`

	// Skill is the spectator's skill as the setup script holds it, such as "[33.32]".
	Skill string `json:"skill"`

	// CreatedAt is when bar-db stored the spectator.
	CreatedAt time.Time `json:"createdAt"`

	// UpdatedAt is when bar-db last changed the spectator.
	UpdatedAt time.Time `json:"updatedAt"`

	// DemoID is Replay.ID.
	DemoID string `json:"demoId"`

	// UserID is the spectator's Teiserver account ID.
	UserID int `json:"userId"`
}

// Color is a team color.
type Color struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// Position is a position on the map in elmos.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Awards holds the awards of the award screen at the end of a game, as the game sent them in its "AWARDS" Lua
// message. The lists hold the first, second and third place. A TeamID of -1 means that no team got the place.
type Awards struct {
	EconDestroyed          []Award `json:"econDestroyed"`
	FightingUnitsDestroyed []Award `json:"fightingUnitsDestroyed"`
	ResourceEfficiency     []Award `json:"resourceEfficiency"`
	MostResourcesProduced  Award   `json:"mostResourcesProduced"`
	MostDamageTaken        Award   `json:"mostDamageTaken"`
	Sleep                  Award   `json:"sleep"`

	// Cow has no Value.
	Cow Award `json:"cow"`
}

// Award is an award for a team.
type Award struct {
	// TeamID is the number of the team, or -1 if no team got the award.
	TeamID int `json:"teamId"`

	// Value is what the award measures.
	Value float64 `json:"value"`
}

// Replay returns the replay with the given ID, see ReplaySummary.ID.
func (c *Client) Replay(ctx context.Context, id string) (*Replay, error) {
	if id == "" {
		return nil, errors.New("replay ID is empty")
	}

	replayURL, err := url.JoinPath(c.config.ReplayAPIURL, "/replays/", url.PathEscape(id))
	if err != nil {
		return nil, err
	}

	var replay Replay
	if err := c.getJSON(ctx, replayURL, &replay); err != nil {
		return nil, err
	}
	return &replay, nil
}
