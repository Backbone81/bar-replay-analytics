package player_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	replayplayer "github.com/backbone81/bar-replay-analytics-go/internal/replay/player"
)

var _ = Describe("Filter", func() {
	DescribeTable("Matches",
		func(filter replayplayer.Filter, expected bool) {
			replayInfo := &replayplayer.ReplayInfo{
				MapName:     "Argent Strata 1.1",
				PlayerNames: []string{"Alice", "BARb Hard"},
			}

			Expect(filter.Matches(replayInfo)).To(Equal(expected))
		},
		Entry("the zero filter matches every replay",
			replayplayer.Filter{}, true),
		Entry("a part of the map name matches, ignoring case",
			replayplayer.Filter{MapNames: []string{"argent STRATA"}}, true),
		Entry("another map does not match",
			replayplayer.Filter{MapNames: []string{"Supreme Isthmus"}}, false),
		Entry("one of several map names matches",
			replayplayer.Filter{MapNames: []string{"Supreme Isthmus", "Strata"}}, true),
		Entry("a player name matches, ignoring case",
			replayplayer.Filter{PlayerNames: []string{"alice"}}, true),
		Entry("an AI name matches",
			replayplayer.Filter{PlayerNames: []string{"BARb Hard"}}, true),
		Entry("only a whole player name matches",
			replayplayer.Filter{PlayerNames: []string{"Ali"}}, false),
		Entry("one of several player names matches",
			replayplayer.Filter{PlayerNames: []string{"Bob", "Alice"}}, true),
		Entry("map name and player name must both match",
			replayplayer.Filter{MapNames: []string{"Strata"}, PlayerNames: []string{"Bob"}}, false),
		Entry("map name and player name both match",
			replayplayer.Filter{MapNames: []string{"Strata"}, PlayerNames: []string{"Alice"}}, true),
	)
})
