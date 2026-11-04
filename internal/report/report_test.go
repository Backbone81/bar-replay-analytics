package report_test

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
	"github.com/backbone81/bar-replay-analytics-go/internal/events"
	"github.com/backbone81/bar-replay-analytics-go/internal/report"
	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

var _ = Describe("Report", func() {
	Context("Config", func() {
		It("rejects an invalid configuration", func() {
			Expect(report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile(), report.WithIntervalLength(0))).
				Error().To(HaveOccurred())
		})
	})

	Context("Metadata", func() {
		It("collects the metadata from the header and the setup script", func() {
			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Metadata).To(Equal(report.Metadata{
				EngineVersion: "2026.07.04",
				GameVersion:   "Beyond All Reason test-31756-0d4c3c3",
				Map:           "Argent Strata 1.1",
				GameID:        "0102030405060708090a0b0c0d0e0f10",
				StartTime:     time.Unix(1792849701, 0).UTC(),
				Duration:      report.Duration(60 * time.Second),
				WinningTeams:  []int{1},
				ModOptions:    *parseScript(`date_day=29;`),
				MapOptions:    tdf.Section{},
			}))
		})

		It("lists each winning team once, sorted", func() {
			replay := newTestDemo()
			replay.WinningAllyTeams = []uint8{1, 0, 1}

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Metadata.WinningTeams).To(Equal([]int{0, 1}))
		})

		It("reports a game left before game over without duration, winners and economy", func() {
			replay := newTestDemo()
			replay.FileHeader.GameTime = 0
			replay.FileHeader.NumTeams = 0
			replay.WinningAllyTeams = nil
			replay.TeamStats = nil

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Metadata.Duration).To(BeZero())
			Expect(result.Metadata.WinningTeams).To(BeEmpty())
			Expect(result.Players).To(HaveLen(2))
			Expect(result.Players[0].Economy).To(BeNil())
			Expect(result.Players[1].Economy).To(BeNil())
		})

		It("treats empty statistics like none", func() {
			replay := newTestDemo()
			replay.TeamStats = [][]demo.TeamStatistics{{}}

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players).To(HaveLen(2))
			Expect(result.Players[0].Economy).To(BeNil())
		})

		It("reads the map options", func() {
			result, err := report.NewReport(newTestDemo(), parseScript(hostedScript), newHostedEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Metadata.MapOptions).To(Equal(*parseScript(`dry=0;`)))
		})
	})

	Context("Teams", func() {
		It("reports one team per allyteam with its players and outcome", func() {
			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Teams).To(Equal([]report.Team{
				{
					TeamID:    0,
					PlayerIDs: []int{0},
					Won:       false,
				},
				{
					TeamID:    1,
					PlayerIDs: []int{1},
					Won:       true,
				},
			}))
		})

		It("reports a team without players with no player IDs", func() {
			script := strings.Replace(testSetupScript, "[player0]", "[allyteam5] {} [player0]", 1)

			result, err := report.NewReport(newTestDemo(), parseScript(script), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Teams).To(HaveLen(3))
			Expect(result.Teams[2].PlayerIDs).To(BeEmpty())
		})
	})

	Context("Players", func() {
		It("reports a human player and an AI", func() {
			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players).To(HaveLen(2))

			human := result.Players[0]
			Expect(human.PlayerID).To(Equal(0))
			Expect(human.TeamID).To(Equal(0))
			Expect(human.Name).To(Equal("Player"))
			Expect(human.Controllers).To(BeEmpty())
			Expect(human.Faction).To(Equal("Cortex"))
			Expect(human.IncomeMultiplier).To(Equal(1.0))
			Expect(human.AI).To(BeNil())
			Expect(human.Economy).NotTo(BeNil())

			ai := result.Players[1]
			Expect(ai.PlayerID).To(Equal(1))
			Expect(ai.TeamID).To(Equal(1))
			Expect(ai.Name).To(Equal("BARbarianAI(1)"))
			Expect(ai.Faction).To(Equal("Armada"))
			Expect(ai.AI).To(Equal(&report.AI{
				ShortName: "BARb",
				Profile:   "Hard",
				Options:   *parseScript(`profile=Hard;`),
			}))
			Expect(ai.Economy).NotTo(BeNil())
		})

		It("names an AI without a name by its short name", func() {
			script := strings.Replace(testSetupScript, "name=BARbarianAI(1);", "", 1)

			result, err := report.NewReport(newTestDemo(), parseScript(script), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[1].Name).To(Equal("BARb"))
		})

		It("leaves spectators out", func() {
			result, err := report.NewReport(newTestDemo(), parseScript(hostedScript), newHostedEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players).To(HaveLen(2))
			Expect(result.Players[0].Name).To(Equal("A"))
			Expect(result.Players[0].Controllers).To(BeEmpty())
			Expect(result.Players[1].Name).To(Equal("B"))
			Expect(result.Players[1].Controllers).To(BeEmpty())
		})

		It("lists the controllers of a shared player", func() {
			script := strings.Replace(hostedScript, "[player2] { name=Spec; spectator=1; }",
				"[player2] { name=C; team=0; spectator=0; }", 1)

			result, err := report.NewReport(newTestDemo(), parseScript(script), newHostedEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Name).To(Equal("A"))
			Expect(result.Players[0].Controllers).To(Equal([]string{"A", "C"}))
			Expect(result.Players[1].Controllers).To(BeEmpty())
		})

		It("takes the faction from the start unit, which resolves Random", func() {
			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), newTestEventsFile())

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[1].Faction).To(Equal("Armada"))
		})

		It("overwrites the faction of the setup script with the one of the start unit", func() {
			eventsFile := newTestEventsFile()
			eventsFile.Teams[0].StartUnit = new(7)

			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[0].Faction).To(Equal("Legion"))
		})

		It("keeps the faction of the setup script without a start unit", func() {
			eventsFile := newTestEventsFile()
			eventsFile.Teams[1].StartUnit = nil

			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[1].Faction).To(Equal("Random"))
		})

		It("keeps the faction of the setup script for a start unit of no side", func() {
			eventsFile := newTestEventsFile()
			eventsFile.Teams[1].StartUnit = new(4)

			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[1].Faction).To(Equal("Random"))
		})

		It("keeps the faction of the setup script without sides", func() {
			eventsFile := newTestEventsFile()
			eventsFile.Sides = nil

			result, err := report.NewReport(newTestDemo(), newTestSetupScript(), eventsFile)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Players[1].Faction).To(Equal("Random"))
		})

		DescribeTable("reads the income multiplier",
			func(values string, want float64) {
				script := strings.Replace(testSetupScript, "side=Cortex; handicap=0;", "side=Cortex; "+values, 1)

				result, err := report.NewReport(newTestDemo(), parseScript(script), newTestEventsFile())

				Expect(err).NotTo(HaveOccurred())
				Expect(result.Players[0].IncomeMultiplier).To(BeNumerically("~", want, 1e-9))
			},
			Entry("without a value", "", 1.0),
			Entry("from a handicap in percent", "handicap=20;", 1.2),
			Entry("from an advantage", "advantage=0.5;", 1.5),
			Entry("from an incomemultiplier", "incomemultiplier=3;", 3.0),
			Entry("as at least 0 from a handicap", "handicap=-200;", 0.0),
			Entry("as at least 0 from an incomemultiplier", "incomemultiplier=-1;", 0.0),
		)
	})

	Context("setup script", func() {
		DescribeTable("rejects",
			func(old, replacement string) {
				Expect(strings.Contains(testSetupScript, old)).To(BeTrue())
				script := strings.Replace(testSetupScript, old, replacement, 1)

				Expect(report.NewReport(newTestDemo(), parseScript(script), newTestEventsFile())).Error().To(HaveOccurred())
			},
			Entry("a team of another number than the statistics", "[team2]", "[team1] { allyteam=0; teamleader=0; } [team2]"),
			Entry("a team without allyteam", "[team0] { allyteam=0;", "[team0] {"),
			Entry("a team of a missing allyteam", "[team0] { allyteam=0;", "[team0] { allyteam=1;"),
			Entry("a team with an invalid allyteam", "[team0] { allyteam=0;", "[team0] { allyteam=x;"),
			Entry("a team without team leader", "allyteam=0; teamleader=0;", "allyteam=0;"),
			Entry("a team of a missing team leader", "allyteam=0; teamleader=0;", "allyteam=0; teamleader=4;"),
			Entry("a player of a missing team", "name=Player; team=0;", "name=Player; team=1;"),
			Entry("a player with an invalid spectator", "name=Player; team=0;", "name=Player; team=0; spectator=x;"),
			Entry("an AI of a missing team", "host=0;\n\t\t\tteam=2;", "host=0;\n\t\t\tteam=1;"),
			Entry("an invalid handicap", "side=Cortex; handicap=0;", "side=Cortex; handicap=x;"),
			Entry("two income multipliers", "side=Cortex; handicap=0;", "side=Cortex; handicap=0; advantage=1;"),
		)
	})

	Context("MarshalJSON", func() {
		It("writes the report as JSON", func() {
			replay := newTestDemo()
			replay.TeamStats = [][]demo.TeamStatistics{{{Frame: 0}}, {{Frame: 0}}}

			result, err := report.NewReport(replay, newTestSetupScript(), newTestEventsFile())
			Expect(err).NotTo(HaveOccurred())

			Expect(json.Marshal(result)).To(MatchJSON(`{
				"metadata": {
					"engineVersion": "2026.07.04",
					"gameVersion": "Beyond All Reason test-31756-0d4c3c3",
					"map": "Argent Strata 1.1",
					"gameID": "0102030405060708090a0b0c0d0e0f10",
					"startTime": "2026-10-24T13:48:21Z",
					"duration": "1:00",
					"winningTeams": [1],
					"modOptions": {"date_day": "29"},
					"mapOptions": {}
				},
				"teams": [
					{
						"teamID": 0,
						"playerIDs": [0],
						"won": false
					},
					{
						"teamID": 1,
						"playerIDs": [1],
						"won": true
					}
				],
				"players": [
					{
						"playerID": 0,
						"teamID": 0,
						"name": "Player",
						"faction": "Cortex",
						"incomeMultiplier": 1,
						"ai": null,
						"economy": {
							"total": {` + zeroResourcesJSON + `},
							"perSecond": {` + zeroResourcesJSON + `},
							"intervals": []
						},
						"combat": {
							"total": {"killed": {"count": 0, "metal": 0}, "lost": {"count": 1, "metal": 50}},
							"intervals": [{"time": "0:00-1:00", "killed": {"count": 0, "metal": 0}, "lost": {"count": 1, "metal": 50}}]
						},
						"openingBuildOrder": [
							{"time": "0:20", "name": "Metal Extractor", "buildRate": 1},
							{"time": "1:30", "name": "Bot Lab", "buildRate": 2}
						],
						"firstFinished": [
							{"time": "0:20", "name": "Metal Extractor"},
							{"time": "1:30", "name": "Bot Lab"},
							{"time": "5:10", "name": "Construction Bot"}
						],
						"builders": [
							{
								"name": "Cortex Commander 1",
								"time": "0:00-5:10",
								"idle": "0:00",
								"idleFraction": 0,
								"idleStretches": [],
								"builds": [{"name": "Metal Extractor", "count": 1}, {"name": "Bot Lab", "count": 1}]
							},
							{
								"name": "Cortex Commander 2",
								"time": "1:50-5:10",
								"idle": "0:00",
								"idleFraction": 0,
								"idleStretches": [],
								"builds": []
							},
							{
								"name": "Construction Bot 1",
								"time": "5:10-5:10",
								"idle": "0:00",
								"idleFraction": 0,
								"idleStretches": [],
								"builds": []
							}
						]
					},
					{
						"playerID": 1,
						"teamID": 1,
						"name": "BARbarianAI(1)",
						"faction": "Armada",
						"incomeMultiplier": 1,
						"ai": {"shortName": "BARb", "profile": "Hard", "options": {"profile": "Hard"}},
						"economy": {
							"total": {` + zeroResourcesJSON + `},
							"perSecond": {` + zeroResourcesJSON + `},
							"intervals": []
						},
						"combat": {
							"total": {"killed": {"count": 0, "metal": 0}, "lost": {"count": 0, "metal": 0}},
							"intervals": [{"time": "0:00-1:00", "killed": {"count": 0, "metal": 0}, "lost": {"count": 0, "metal": 0}}]
						},
						"openingBuildOrder": [
							{"time": "0:25", "name": "Metal Extractor", "buildRate": 0.83}
						],
						"firstFinished": [
							{"time": "0:25", "name": "Metal Extractor"}
						],
						"builders": [
							{
								"name": "Armada Commander 1",
								"time": "0:00-5:10",
								"idle": "0:00",
								"idleFraction": 0,
								"idleStretches": [],
								"builds": [{"name": "Metal Extractor", "count": 1}]
							}
						]
					}
				]
			}`))
		})
	})
})

// hostedScript is the setup script of a 1v1 hosted by SPADS, with a spectator.
const hostedScript = `
	[game]
	{
		gametype=Beyond All Reason test-31691-a617033;
		mapname=Pools of Ilys 1.1.2;
		hosttype=SPADS;
		server_match_id=11229606;
		[modoptions] { ranked_game=1; }
		[mapoptions] { dry=0; }
		[allyteam0] {}
		[allyteam1] {}
		[player0] { name=A; team=0; skill=[36.68]; skilluncertainty=2.57; spectator=0; }
		[player1] { name=B; team=1; skill=21.5; skilluncertainty=3; spectator=0; }
		[player2] { name=Spec; spectator=1; }
		[team0] { allyteam=0; teamleader=0; side=Cortex; }
		[team1] { allyteam=1; teamleader=1; side=Armada; }
	}
`

// newHostedEventsFile returns newTestEventsFile as the events file of the game of hostedScript.
func newHostedEventsFile() *events.File {
	eventsFile := newTestEventsFile()
	eventsFile.Game.MapName = "Pools of Ilys 1.1.2"
	eventsFile.Game.GameVersion = "test-31691-a617033"
	return eventsFile
}

// zeroResourcesJSON is the members of Resources with all amounts 0, as JSON.
const zeroResourcesJSON = `
	"metal": {"produced": 0, "used": 0, "wasted": 0, "received": 0, "sent": 0},
	"energy": {"produced": 0, "used": 0, "wasted": 0, "received": 0, "sent": 0}`
