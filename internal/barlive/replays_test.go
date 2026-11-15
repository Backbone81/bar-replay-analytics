package barlive_test

import (
	"context"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
	"github.com/backbone81/bar-replay-analytics/internal/barlive/barlivetest"
)

var _ = Describe("Replays", func() {
	var server *barlivetest.Server

	BeforeEach(func() {
		server = newServer()
	})

	Context("ReplaysQuery.Values", func() {
		It("is empty for the zero value", func() {
			query := barlive.ReplaysQuery{}

			Expect(query.Values()).To(BeEmpty())
		})

		It("sets every field", func() {
			endedNormally, hasBots, reported := true, false, false
			query := barlive.ReplaysQuery{
				Page:          2,
				Limit:         50,
				Preset:        []barlive.ReplayPreset{barlive.ReplayPresetDuel, barlive.ReplayPresetFFA},
				EndedNormally: &endedNormally,
				HasBots:       &hasBots,
				Reported:      &reported,
				Players:       []string{"Player002", "Player033"},
				Maps:          []string{"Cape_Violet_V1"},
				Date: &barlive.DateRange{
					From: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
					To:   time.Date(2026, 9, 30, 23, 0, 0, 0, time.UTC),
				},
				DurationRange:       &barlive.DurationRange{Min: 90 * time.Second, Max: time.Hour},
				TSRange:             &barlive.TrueSkillRange{Min: 20, Max: 45.5},
				ComputeTotalResults: true,
			}

			Expect(query.Values()).To(Equal(url.Values{
				"page":                {"2"},
				"limit":               {"50"},
				"preset":              {"duel", "ffa"},
				"endedNormally":       {"true"},
				"hasBots":             {"false"},
				"reported":            {"false"},
				"players":             {"Player002", "Player033"},
				"maps":                {"Cape_Violet_V1"},
				"date":                {"2026-09-01", "2026-10-01"},
				"durationRangeMins":   {"1.5", "60"},
				"tsRange":             {"20", "45.5"},
				"computeTotalResults": {"true"},
			}))
		})

		It("selects one day if Date has no end", func() {
			query := barlive.ReplaysQuery{Date: &barlive.DateRange{From: time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)}}

			Expect(query.Values()).To(Equal(url.Values{"date": {"2026-12-31", "2027-01-01"}}))
		})
	})

	Context("Replays", func() {
		It("returns a page of replays", func() {
			server.HandleFunc("GET /replays", barlivetest.ServeJSONFile(
				http.StatusOK, filepath.Join("testdata", "replays_page_1.json")))
			hasBots := false

			page, err := server.Client().Replays(context.Background(), barlive.ReplaysQuery{
				Limit:   3,
				Preset:  []barlive.ReplayPreset{barlive.ReplayPresetDuel},
				HasBots: &hasBots,
				Players: []string{"Player002"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(server.Requests()[0].Query).To(Equal(url.Values{
				"limit":   {"3"},
				"preset":  {"duel"},
				"hasBots": {"false"},
				"players": {"Player002"},
			}))
			Expect(page.TotalResults).To(Equal(-1))
			Expect(page.Page).To(Equal(1))
			Expect(page.Limit).To(Equal(3))
			Expect(page.Data).To(HaveLen(3))

			replay := page.Data[0]
			Expect(replay.ID).To(Equal("00000000000000000000000000000001"))
			Expect(replay.StartTime).To(BeTemporally("==", time.Date(2026, 9, 10, 0, 14, 8, 0, time.UTC)))
			Expect(replay.Duration()).To(Equal(890333 * time.Millisecond))
			Expect(replay.Map).To(Equal(barlive.Map{ScriptName: "Cape_Violet_V1", FileName: "cape_violet_v1"}))
			Expect(replay.AllyTeams).To(Equal([]barlive.AllyTeam{
				{WinningTeam: false, Players: []barlive.Player{{Name: "Player033"}}, AIs: []barlive.AI{}},
				{WinningTeam: true, Players: []barlive.Player{{Name: "Player002"}}, AIs: []barlive.AI{}},
			}))
		})

		It("returns the AIs of a replay", func() {
			server.HandleFunc("GET /replays", barlivetest.ServeJSONFile(
				http.StatusOK, filepath.Join("testdata", "replays_bots.json")))

			page, err := server.Client().Replays(context.Background(), barlive.ReplaysQuery{})

			Expect(err).NotTo(HaveOccurred())
			Expect(page.Data).To(HaveLen(1))
			Expect(page.Data[0].AllyTeams).To(Equal([]barlive.AllyTeam{
				{WinningTeam: true, Players: []barlive.Player{}, AIs: []barlive.AI{{ShortName: "BARb"}}},
				{WinningTeam: false, Players: []barlive.Player{{Name: "Player301"}}, AIs: []barlive.AI{}},
			}))
		})

		It("reports a query bar-db rejects", func() {
			server.HandleFunc("GET /replays", barlivetest.ServeJSONFile(
				http.StatusBadRequest, filepath.Join("testdata", "replays_bad_request.json")))

			Expect(server.Client().Replays(context.Background(), barlive.ReplaysQuery{
				Preset: []barlive.ReplayPreset{"nonsense"},
			})).Error().To(HaveOccurred())
		})

		DescribeTable("rejects an invalid page or limit without a request",
			func(query barlive.ReplaysQuery) {
				Expect(server.Client().Replays(context.Background(), query)).Error().To(HaveOccurred())
				Expect(server.Requests()).To(BeEmpty())
			},
			Entry("negative limit", barlive.ReplaysQuery{Limit: -1}),
			Entry("limit above the max", barlive.ReplaysQuery{Limit: barlive.MaxReplaysLimit + 1}),
			Entry("negative page", barlive.ReplaysQuery{Page: -1}),
		)
	})
})
