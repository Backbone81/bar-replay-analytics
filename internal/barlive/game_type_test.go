package barlive_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// testReplaySummary returns a replay whose allyteams have the given numbers of players.
func testReplaySummary(allyTeamSizes ...int) *barlive.ReplaySummary {
	replay := &barlive.ReplaySummary{}
	for _, size := range allyTeamSizes {
		allyTeam := barlive.AllyTeam{Players: []barlive.Player{}, AIs: []barlive.AI{}}
		for range size {
			allyTeam.Players = append(allyTeam.Players, barlive.Player{Name: "Player"})
		}
		replay.AllyTeams = append(replay.AllyTeams, allyTeam)
	}
	return replay
}

var _ = Describe("GameType", func() {
	DescribeTable("ReplayPreset",
		func(gameType barlive.GameType, expectedPreset barlive.ReplayPreset, expectedFound bool) {
			preset, found := gameType.ReplayPreset()

			Expect(found).To(Equal(expectedFound))
			Expect(preset).To(Equal(expectedPreset))
		},
		Entry("Duel", barlive.GameTypeDuel, barlive.ReplayPresetDuel, true),
		Entry("Small Team", barlive.GameTypeSmallTeam, barlive.ReplayPresetTeam, true),
		Entry("Large Team", barlive.GameTypeLargeTeam, barlive.ReplayPresetTeam, true),
		Entry("FFA", barlive.GameTypeFFA, barlive.ReplayPresetFFA, true),
		Entry("unknown", barlive.GameType("Team FFA"), barlive.ReplayPreset(""), false),
	)

	DescribeTable("Matches",
		func(replay *barlive.ReplaySummary, expected map[barlive.GameType]bool) {
			for _, gameType := range []barlive.GameType{
				barlive.GameTypeDuel, barlive.GameTypeSmallTeam, barlive.GameTypeLargeTeam, barlive.GameTypeFFA,
				"Team FFA",
			} {
				Expect(gameType.Matches(replay)).To(Equal(expected[gameType]), string(gameType))
			}
		},
		Entry("1v1", testReplaySummary(1, 1), map[barlive.GameType]bool{barlive.GameTypeDuel: true}),
		Entry("2v2", testReplaySummary(2, 2), map[barlive.GameType]bool{barlive.GameTypeSmallTeam: true}),
		Entry("5v5", testReplaySummary(5, 5), map[barlive.GameType]bool{barlive.GameTypeSmallTeam: true}),
		Entry("uneven 1v2", testReplaySummary(1, 2), map[barlive.GameType]bool{barlive.GameTypeSmallTeam: true}),
		Entry("6v6", testReplaySummary(6, 6), map[barlive.GameType]bool{barlive.GameTypeLargeTeam: true}),
		Entry("8v8", testReplaySummary(8, 8), map[barlive.GameType]bool{barlive.GameTypeLargeTeam: true}),
		Entry("3-way FFA", testReplaySummary(1, 1, 1), map[barlive.GameType]bool{barlive.GameTypeFFA: true}),
		Entry("FFA with an empty allyteam", testReplaySummary(1, 0, 1, 1),
			map[barlive.GameType]bool{barlive.GameTypeFFA: true}),
		Entry("1v1 with an empty allyteam", testReplaySummary(1, 0, 1),
			map[barlive.GameType]bool{barlive.GameTypeDuel: true}),
		Entry("team FFA", testReplaySummary(2, 2, 2), map[barlive.GameType]bool{}),
		Entry("one allyteam", testReplaySummary(2), map[barlive.GameType]bool{}),
		Entry("no allyteams", testReplaySummary(), map[barlive.GameType]bool{}),
	)

	It("does not match a game with an AI", func() {
		replay := testReplaySummary(1, 1)
		replay.AllyTeams[1].AIs = []barlive.AI{{ShortName: "BARb"}}

		Expect(barlive.GameTypeDuel.Matches(replay)).To(BeFalse())
	})
})
