package report_test

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
)

var _ = Describe("Report", func() {
	Context("Metadata", func() {
		It("collects the metadata from the header and the setup script", func() {
			result, err := report.NewReport(newTestDemo())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Metadata).To(Equal(report.Metadata{
				GameID:        "0102030405060708090a0b0c0d0e0f10",
				EngineVersion: "2026.07.04",
				StartTime:     time.Unix(1792849701, 0).UTC(),
				Duration:      report.Duration(100 * time.Second),
				WinningTeams:  []int{1},
			}))
		})

		It("lists each winning team once, sorted", func() {
			replay := newTestDemo()
			replay.WinningAllyTeams = []uint8{1, 0, 1}

			result, err := report.NewReport(replay)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Metadata.WinningTeams).To(Equal([]int{0, 1}))
		})

		It("reports a game left before game over without duration, winners and players", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 0
			replay.FileHeader.NumTeams = 0
			replay.WinningAllyTeams = nil
			replay.TeamStats = nil

			result, err := report.NewReport(replay)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Metadata.Duration).To(BeZero())
			Expect(result.Metadata.WinningTeams).To(BeEmpty())
			Expect(result.Players).To(BeEmpty())
		})
	})

	Context("Players", func() {
		It("reports one player per team statistics history, numbered by its index", func() {
			result, err := report.NewReport(newTestDemo())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players).To(HaveLen(2))
			Expect(result.Players[0].PlayerID).To(Equal(0))
			Expect(result.Players[0].Economy).NotTo(BeNil())
			Expect(result.Players[1].PlayerID).To(Equal(1))
			Expect(result.Players[1].Economy).NotTo(BeNil())
		})
	})

	Context("MarshalJSON", func() {
		It("writes the report as JSON", func() {
			replay := newTestDemo()
			replay.TeamStats = [][]demo.TeamStatistics{{{Frame: 0}}, {{Frame: 0}}}

			result, err := report.NewReport(replay)
			Expect(err).NotTo(HaveOccurred())

			Expect(json.Marshal(result)).To(MatchJSON(`{
				"metadata": {
					"gameID": "0102030405060708090a0b0c0d0e0f10",
					"engineVersion": "2026.07.04",
					"startTime": "2026-10-24T13:48:21Z",
					"duration": "1:40",
					"winningTeams": [1]
				},
				"players": [
					{
						"playerID": 0,
						"economy": {
							"total": {` + zeroResourcesJSON + `},
							"perSecond": {` + zeroResourcesJSON + `},
							"intervals": []
						}
					},
					{
						"playerID": 1,
						"economy": {
							"total": {` + zeroResourcesJSON + `},
							"perSecond": {` + zeroResourcesJSON + `},
							"intervals": []
						}
					}
				]
			}`))
		})
	})
})

// zeroResourcesJSON is the members of Resources with all amounts 0, as JSON.
const zeroResourcesJSON = `
	"metal": {"produced": 0, "used": 0, "wasted": 0, "received": 0, "sent": 0},
	"energy": {"produced": 0, "used": 0, "wasted": 0, "received": 0, "sent": 0}`
