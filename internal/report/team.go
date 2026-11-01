package report

import (
	"slices"

	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// Team is one side of the game: the players allied with each other. The engine calls this an allyteam.
type Team struct {
	// TeamID is the ID of this team: the engine's allyteam ID.
	TeamID int `json:"teamID"` //nolint:tagliatelle

	// PlayerIDs provides the IDs of the players in this team, sorted.
	PlayerIDs []int `json:"playerIDs"`

	// Won is true if this team has won.
	Won bool `json:"won"`
}

// newTeams prepares the teams from the allyteam sections of the setup script's game section, given the IDs of the
// winning teams. Their PlayerIDs are left empty.
// Returns the teams and the team IDs by allyteam number.
func newTeams(game *tdf.Section, winners []int) ([]Team, map[int]int) {
	var teams []Team
	teamIDs := make(map[int]int)
	for number := range game.Indexed("allyteam") {
		teamID := len(teams)
		teamIDs[number] = teamID
		teams = append(teams, Team{
			TeamID:    teamID,
			PlayerIDs: []int{},
			Won:       slices.Contains(winners, teamID),
		})
	}
	return teams, teamIDs
}
