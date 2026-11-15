package utils_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

var _ = Describe("Filter", func() {
	DescribeTable("Matches",
		func(filter utils.Filter, expected bool) {
			replayInfo := &utils.ReplayInfo{
				MapName:     "Argent Strata 1.1",
				PlayerNames: []string{"Alice", "BARb Hard"},
			}

			Expect(filter.Matches(replayInfo)).To(Equal(expected))
		},
		Entry("the zero filter matches every replay",
			utils.Filter{}, true),
		Entry("a part of the map name matches, ignoring case",
			utils.Filter{MapNames: []string{"argent STRATA"}}, true),
		Entry("another map does not match",
			utils.Filter{MapNames: []string{"Supreme Isthmus"}}, false),
		Entry("one of several map names matches",
			utils.Filter{MapNames: []string{"Supreme Isthmus", "Strata"}}, true),
		Entry("a player name matches, ignoring case",
			utils.Filter{PlayerNames: []string{"alice"}}, true),
		Entry("an AI name matches",
			utils.Filter{PlayerNames: []string{"BARb Hard"}}, true),
		Entry("only a whole player name matches",
			utils.Filter{PlayerNames: []string{"Ali"}}, false),
		Entry("one of several player names matches",
			utils.Filter{PlayerNames: []string{"Bob", "Alice"}}, true),
		Entry("map name and player name must both match",
			utils.Filter{MapNames: []string{"Strata"}, PlayerNames: []string{"Bob"}}, false),
		Entry("map name and player name both match",
			utils.Filter{MapNames: []string{"Strata"}, PlayerNames: []string{"Alice"}}, true),
	)
})
