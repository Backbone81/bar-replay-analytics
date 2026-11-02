package report

import (
	"fmt"

	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// Player is one player of the game, human or AI: the units and resources one player or AI controls, or several players
// sharing control. The engine calls this a team.
type Player struct {
	// PlayerID is the ID of this player: the engine's team ID.
	PlayerID int `json:"playerID"` //nolint:tagliatelle

	// TeamID is the ID of the team this player plays in: the engine's allyteam ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle

	// Name is the name of the AI if an AI controls this player, otherwise the name of the player's team leader.
	Name string `json:"name"`

	// Controllers provides the names of the players and AIs sharing control of this player, if there are several.
	// Is empty if one player or AI controls it.
	Controllers []string `json:"controllers,omitempty"`

	// Faction is the faction the player played, like "Cortex": the side of the commander it started with
	// (events.Team.StartUnit, events.File.Sides). The game resolves a faction chosen as "Random" in the lobby only when
	// it starts, and a player may change the faction after the lobby, so the setup script does not tell. Is the
	// faction of the setup script, possibly "Random", if the events file has no start unit or no side of it.
	Faction string `json:"faction"`

	// IncomeMultiplier is the factor the engine applies to this player's income: 1 normally, 1.2 with a bonus of 20%
	// as the lobby sets it (the team's handicap in the setup script, see TeamBase::SetValue in
	// rts/Sim/Misc/TeamBase.cpp). BAR also multiplies the starting resources with it if the game option
	// bonusstartresourcemultiplier is set.
	IncomeMultiplier float64 `json:"incomeMultiplier"`

	// AI describes the AI controlling this player. Is nil for a human player.
	AI *AI `json:"ai"`

	// Economy provides the resources this player produced and used, from the statistics history of the replay, or of
	// the events file if that one has more entries (see economyHistory).
	// Is nil if neither has statistics of the player: the engine writes them to the replay only at game over.
	Economy *Economy `json:"economy"`

	// OpeningBuildOrder provides the units and buildings this player built in the first Config.OpeningBuildOrderLength
	// of the game, in the order they were finished. Units which were not built, such as the commander at the start, are
	// left out. Is empty if Config.OpeningBuildOrderLength is 0.
	OpeningBuildOrder []BuildOrderEntry `json:"openingBuildOrder"`
}

// AI describes an AI controlling a player.
type AI struct {
	// ShortName names the AI, like "BARb" or "SimpleAI".
	ShortName string `json:"shortName"`

	// Profile is the AI's profile, like "hard" for BARb; its case varies between lobby versions. Is empty if the AI
	// has no profile option.
	Profile string `json:"profile"`

	// Options provides all the AI's options, the profile included, without empty values. Which there are depends on
	// the AI.
	Options tdf.Section `json:"options"`
}

// newPlayers prepares the players, without economy and build order, from the team sections of the setup script's game
// section, given the team IDs by allyteam number.
// Returns an error if a value of the setup script is invalid, or if it refers to a missing section.
func newPlayers(game *tdf.Section, teamIDs map[int]int) ([]Player, error) {
	if err := checkTeamReferences(game); err != nil {
		return nil, err
	}

	var players []Player
	for number, section := range game.Indexed("team") {
		player, err := newPlayer(game, number, section, teamIDs)
		if err != nil {
			return nil, fmt.Errorf("team%d: %w", number, err)
		}
		player.PlayerID = len(players)
		players = append(players, player)
	}
	return players, nil
}

// newPlayer prepares a player, without ID, economy and build order, from the team section of the given number.
func newPlayer(game *tdf.Section, number int, section *tdf.Section, teamIDs map[int]int) (Player, error) {
	value, _ := section.Value("allyteam")
	allyTeam, err := value.AsInt()
	if err != nil {
		return Player{}, fmt.Errorf("allyteam: %w", err)
	}
	teamID, ok := teamIDs[allyTeam]
	if !ok {
		return Player{}, fmt.Errorf("allyteam: %d does not exist", allyTeam)
	}
	incomeMultiplier, err := incomeMultiplier(section)
	if err != nil {
		return Player{}, err
	}
	side, _ := section.Value("side")
	player := Player{
		TeamID:           teamID,
		Faction:          side.String(),
		IncomeMultiplier: incomeMultiplier,
	}

	humans, err := controllers(game, "player", number)
	if err != nil {
		return Player{}, err
	}
	ais, err := controllers(game, "ai", number)
	if err != nil {
		return Player{}, err
	}
	var names []string
	for _, human := range humans {
		name, _ := human.Value("name")
		names = append(names, name.String())
	}
	for _, ai := range ais {
		names = append(names, aiName(ai))
	}
	if len(names) > 1 {
		player.Controllers = names
	}

	leader, err := teamLeader(game, section)
	if err != nil {
		return Player{}, err
	}
	// An AI's team leader is the player hosting the AI, so only a human team is named after it.
	if len(ais) > 0 {
		player.Name = aiName(ais[0])
		player.AI = newAI(ais[0])
	} else {
		name, _ := leader.Value("name")
		player.Name = name.String()
	}
	return player, nil
}

// teamLeader returns the player section of a team section's team leader.
// Returns an error if there is none: the engine refuses such a setup script.
func teamLeader(game *tdf.Section, section *tdf.Section) (*tdf.Section, error) {
	value, _ := section.Value("teamleader")
	number, err := value.AsInt()
	if err != nil {
		return nil, fmt.Errorf("teamleader: %w", err)
	}
	leader := game.Section(fmt.Sprintf("player%d", number))
	if leader == nil {
		return nil, fmt.Errorf("teamleader: %d does not exist", number)
	}
	return leader, nil
}

// controllers returns the player or AI sections (by prefix: "player" or "ai") which control the team of the given
// number, sorted by their numbers. Spectators are not included.
func controllers(game *tdf.Section, prefix string, team int) ([]*tdf.Section, error) {
	var result []*tdf.Section
	for number, section := range game.Indexed(prefix) {
		controlled, playing, err := controlledTeam(section)
		if err != nil {
			return nil, fmt.Errorf("%s%d: %w", prefix, number, err)
		}
		if playing && controlled == team {
			result = append(result, section)
		}
	}
	return result, nil
}

// checkTeamReferences returns an error if a player or AI refers to a team section which does not exist: the engine
// refuses such a setup script.
func checkTeamReferences(game *tdf.Section) error {
	teams := make(map[int]bool)
	for number := range game.Indexed("team") {
		teams[number] = true
	}
	for _, prefix := range []string{"player", "ai"} {
		for number, section := range game.Indexed(prefix) {
			team, playing, err := controlledTeam(section)
			if err != nil {
				return fmt.Errorf("%s%d: %w", prefix, number, err)
			}
			if playing && !teams[team] {
				return fmt.Errorf("%s%d: team: %d does not exist", prefix, number, team)
			}
		}
	}
	return nil
}

// controlledTeam returns the number of the team a player or AI section controls, and false for a spectator.
func controlledTeam(section *tdf.Section) (int, bool, error) {
	if value, ok := section.Value("spectator"); ok {
		spectator, err := value.AsBool()
		if err != nil {
			return 0, false, fmt.Errorf("spectator: %w", err)
		}
		if spectator {
			return 0, false, nil
		}
	}
	value, _ := section.Value("team")
	team, err := value.AsInt()
	if err != nil {
		return 0, false, fmt.Errorf("team: %w", err)
	}
	return team, true, nil
}

// newAI describes the AI of an AI section.
func newAI(section *tdf.Section) *AI {
	shortName, _ := section.Value("shortname")
	profile, _ := section.Value(`options\profile`)
	return &AI{
		ShortName: shortName.String(),
		Profile:   profile.String(),
		Options:   withoutEmptyValues(section.Section("options")),
	}
}

// aiName returns the name of an AI: its name, or its short name if it has none, as CGameSetup::LoadSkirmishAIs names
// it.
func aiName(section *tdf.Section) string {
	if name, ok := section.Value("name"); ok {
		return name.String()
	}
	shortName, _ := section.Value("shortname")
	return shortName.String()
}

// incomeMultiplier returns the income multiplier of a team section, as TeamBase::SetValue reads it from handicap
// (a bonus in percent), advantage (a bonus as a factor) or incomemultiplier.
// Returns an error if the value is invalid, or if more than one is set: the engine applies them in no defined order.
func incomeMultiplier(section *tdf.Section) (float64, error) {
	var result float64 = 1
	found := 0
	for name, value := range section.Values() {
		var err error
		switch name {
		case "handicap":
			result, err = advantageMultiplier(value, 100)
		case "advantage":
			result, err = advantageMultiplier(value, 1)
		case "incomemultiplier":
			result, err = value.AsFloat()
			result = max(0, result)
		default:
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("%s: %w", name, err)
		}
		found++
	}
	if found > 1 {
		return 0, fmt.Errorf("got %d of handicap, advantage and incomemultiplier, want at most one", found)
	}
	return result, nil
}

// advantageMultiplier returns the income multiplier of an advantage, given in units of divisor: 100 for a percentage.
// See TeamBase::SetAdvantage in rts/Sim/Misc/TeamBase.h.
func advantageMultiplier(value tdf.Value, divisor float64) (float64, error) {
	advantage, err := value.AsFloat()
	if err != nil {
		return 0, err
	}
	return max(-1, advantage/divisor) + 1, nil
}

// setFactionFromStartUnit overwrites the faction of the setup script with the side of the start unit the events file
// holds for the player, if it holds both. eventsFile must be valid (events.File.Validate).
func (p *Player) setFactionFromStartUnit(eventsFile *events.File) {
	for _, team := range eventsFile.Teams {
		if team.TeamID != p.PlayerID || team.StartUnit == nil {
			continue
		}
		startUnit := eventsFile.UnitDefs[*team.StartUnit].Name
		for _, side := range eventsFile.Sides {
			if side.StartUnit == startUnit {
				p.Faction = side.CaseName
			}
		}
	}
}
