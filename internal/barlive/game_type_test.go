package barlive_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
	"github.com/backbone81/bar-replay-analytics/internal/barlive/barlivetest"
)

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
		func(replay barlive.ReplaySummary, expected map[barlive.GameType]bool) {
			for _, gameType := range []barlive.GameType{
				barlive.GameTypeDuel, barlive.GameTypeSmallTeam, barlive.GameTypeLargeTeam, barlive.GameTypeFFA,
				"Team FFA",
			} {
				Expect(gameType.Matches(&replay)).To(Equal(expected[gameType]), string(gameType))
			}
		},
		Entry("1v1", barlivetest.ReplaySummary("", 1, 1), map[barlive.GameType]bool{barlive.GameTypeDuel: true}),
		Entry("2v2", barlivetest.ReplaySummary("", 2, 2), map[barlive.GameType]bool{barlive.GameTypeSmallTeam: true}),
		Entry("5v5", barlivetest.ReplaySummary("", 5, 5), map[barlive.GameType]bool{barlive.GameTypeSmallTeam: true}),
		Entry("uneven 1v2", barlivetest.ReplaySummary("", 1, 2), map[barlive.GameType]bool{barlive.GameTypeSmallTeam: true}),
		Entry("6v6", barlivetest.ReplaySummary("", 6, 6), map[barlive.GameType]bool{barlive.GameTypeLargeTeam: true}),
		Entry("8v8", barlivetest.ReplaySummary("", 8, 8), map[barlive.GameType]bool{barlive.GameTypeLargeTeam: true}),
		Entry("3-way FFA", barlivetest.ReplaySummary("", 1, 1, 1), map[barlive.GameType]bool{barlive.GameTypeFFA: true}),
		Entry("FFA with an empty allyteam", barlivetest.ReplaySummary("", 1, 0, 1, 1),
			map[barlive.GameType]bool{barlive.GameTypeFFA: true}),
		Entry("1v1 with an empty allyteam", barlivetest.ReplaySummary("", 1, 0, 1),
			map[barlive.GameType]bool{barlive.GameTypeDuel: true}),
		Entry("team FFA", barlivetest.ReplaySummary("", 2, 2, 2), map[barlive.GameType]bool{}),
		Entry("one allyteam", barlivetest.ReplaySummary("", 2), map[barlive.GameType]bool{}),
		Entry("no allyteams", barlivetest.ReplaySummary(""), map[barlive.GameType]bool{}),
	)

	It("does not match a game with an AI", func() {
		replay := barlivetest.ReplaySummary("", 1, 1)
		replay.AllyTeams[1].AIs = []barlive.AI{{ShortName: "BARb"}}

		Expect(barlive.GameTypeDuel.Matches(&replay)).To(BeFalse())
	})
})
