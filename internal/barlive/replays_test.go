package barlive_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

var _ = Describe("Replays", func() {
	var server *testServer

	BeforeEach(func() {
		server = newTestServer()
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
			server.HandleFunc("GET /replays", serveTestdata(http.StatusOK, "replays_page_1.json"))
			hasBots := false

			page, err := server.client().Replays(context.Background(), barlive.ReplaysQuery{
				Limit:   3,
				Preset:  []barlive.ReplayPreset{barlive.ReplayPresetDuel},
				HasBots: &hasBots,
				Players: []string{"Player002"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(server.recorded()[0].Query).To(Equal(url.Values{
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
			server.HandleFunc("GET /replays", serveTestdata(http.StatusOK, "replays_bots.json"))

			page, err := server.client().Replays(context.Background(), barlive.ReplaysQuery{})

			Expect(err).NotTo(HaveOccurred())
			Expect(page.Data).To(HaveLen(1))
			Expect(page.Data[0].AllyTeams).To(Equal([]barlive.AllyTeam{
				{WinningTeam: true, Players: []barlive.Player{}, AIs: []barlive.AI{{ShortName: "BARb"}}},
				{WinningTeam: false, Players: []barlive.Player{{Name: "Player301"}}, AIs: []barlive.AI{}},
			}))
		})

		It("reports a query bar-db rejects", func() {
			server.HandleFunc("GET /replays", serveTestdata(http.StatusBadRequest, "replays_bad_request.json"))

			Expect(server.client().Replays(context.Background(), barlive.ReplaysQuery{
				Preset: []barlive.ReplayPreset{"nonsense"},
			})).Error().To(HaveOccurred())
		})

		DescribeTable("rejects an invalid page or limit without a request",
			func(query barlive.ReplaysQuery) {
				Expect(server.client().Replays(context.Background(), query)).Error().To(HaveOccurred())
				Expect(server.recorded()).To(BeEmpty())
			},
			Entry("negative limit", barlive.ReplaysQuery{Limit: -1}),
			Entry("limit above the max", barlive.ReplaysQuery{Limit: barlive.MaxReplaysLimit + 1}),
			Entry("negative page", barlive.ReplaysQuery{Page: -1}),
		)
	})

	Context("AllReplays", func() {
		It("requests pages until an empty one", func() {
			server.HandleFunc("GET /replays", func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Query().Get("page") == "1" {
					serveTestdata(http.StatusOK, "replays_page_1.json")(writer, request)
				} else {
					serveTestdata(http.StatusOK, "replays_page_empty.json")(writer, request)
				}
			})

			var ids []string
			for replay, err := range server.client().AllReplays(
				context.Background(), barlive.ReplaysQuery{Limit: 3}) {
				Expect(err).NotTo(HaveOccurred())
				ids = append(ids, replay.ID)
			}

			Expect(ids).To(Equal([]string{
				"00000000000000000000000000000001",
				"00000000000000000000000000000003",
				"00000000000000000000000000000004",
			}))
			Expect(server.recorded()).To(HaveLen(2))
			Expect(server.recorded()[1].Query.Get("page")).To(Equal("2"))
		})

		It("stops after a page with fewer replays than the limit", func() {
			server.HandleFunc("GET /replays", serveString(http.StatusOK, "application/json",
				`{"totalResults":-1,"page":1,"limit":3,"data":[{"id":"a"},{"id":"b"}]}`))

			count := 0
			for _, err := range server.client().AllReplays(context.Background(), barlive.ReplaysQuery{}) {
				Expect(err).NotTo(HaveOccurred())
				count++
			}

			Expect(count).To(Equal(2))
			Expect(server.recorded()).To(HaveLen(1))
		})

		It("starts every iteration at the page of the query", func() {
			server.HandleFunc("GET /replays", func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Query().Get("page") == "1" {
					serveTestdata(http.StatusOK, "replays_page_1.json")(writer, request)
				} else {
					serveTestdata(http.StatusOK, "replays_page_empty.json")(writer, request)
				}
			})
			replays := server.client().AllReplays(context.Background(), barlive.ReplaysQuery{Limit: 3})

			for range 2 {
				count := 0
				for _, err := range replays {
					Expect(err).NotTo(HaveOccurred())
					count++
				}
				Expect(count).To(Equal(3))
			}

			Expect(server.recorded()).To(HaveLen(4))
			Expect(server.recorded()[2].Query.Get("page")).To(Equal("1"))
		})

		It("starts at the page of the query", func() {
			server.HandleFunc("GET /replays", serveTestdata(http.StatusOK, "replays_page_empty.json"))

			for _, err := range server.client().AllReplays(
				context.Background(), barlive.ReplaysQuery{Page: 9999}) {
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(server.recorded()).To(HaveLen(1))
			Expect(server.recorded()[0].Query.Get("page")).To(Equal("9999"))
		})

		It("requests no further page when the caller stops", func() {
			server.HandleFunc("GET /replays", serveTestdata(http.StatusOK, "replays_page_1.json"))

			for _, err := range server.client().AllReplays(context.Background(), barlive.ReplaysQuery{}) {
				Expect(err).NotTo(HaveOccurred())
				break
			}

			Expect(server.recorded()).To(HaveLen(1))
		})

		It("yields an error and stops", func() {
			server.HandleFunc("GET /replays", serveTestdata(http.StatusBadRequest, "replays_bad_request.json"))

			var errs []error
			for replay, err := range server.client().AllReplays(
				context.Background(), barlive.ReplaysQuery{}) {
				Expect(replay).To(BeZero())
				errs = append(errs, err)
			}

			Expect(errs).To(HaveLen(1))
			Expect(errs[0]).To(HaveOccurred())
		})
	})

	Context("RecentReplays", func() {
		// now is when the spec starts; the replays of the page started before it.
		var now time.Time

		// since is the start of the selected time span.
		var since time.Time

		BeforeEach(func() {
			now = time.Now().UTC().Truncate(time.Second)
			since = now.Add(-48 * time.Hour)
		})

		It("selects the replays of the player and game type in bar-db", func() {
			server.HandleFunc("GET /replays", serveReplaysPage())

			Expect(recentIDs(server, since, barlive.GameTypeSmallTeam)).To(BeEmpty())

			Expect(server.recorded()[0].Query).To(Equal(url.Values{
				"page":          {"1"},
				"limit":         {"100"},
				"preset":        {"team"},
				"endedNormally": {"true"},
				"hasBots":       {"false"},
				"players":       {"Player002"},
				"date":          {since.Format(time.DateOnly), now.AddDate(0, 0, 1).Format(time.DateOnly)},
			}))
		})

		It("selects the maps in bar-db", func() {
			server.HandleFunc("GET /replays", serveReplaysPage())

			Expect(recentIDs(server, since, barlive.GameTypeDuel, "Argent Strata 1.1", "Cape_Violet_V1")).To(BeEmpty())

			Expect(server.recorded()[0].Query["maps"]).To(Equal([]string{"Argent Strata 1.1", "Cape_Violet_V1"}))
		})

		It("returns the replays of the game type which started since then", func() {
			server.HandleFunc("GET /replays", serveReplaysPage(
				replayStartedAt("a", now.Add(-time.Hour), 1, 1),
				replayStartedAt("b", now.Add(-2*time.Hour), 2, 2),
				replayStartedAt("c", now.Add(-3*time.Hour), 1, 1),
				replayStartedAt("d", now.Add(-49*time.Hour), 1, 1),
				replayStartedAt("e", now.Add(-50*time.Hour), 1, 1),
			))

			Expect(recentIDs(server, since, barlive.GameTypeDuel)).To(Equal([]string{"a", "c"}))
			// The page is full, but its older replays end the search.
			Expect(server.recorded()).To(HaveLen(1))
		})

		It("requests the next page while the replays are recent", func() {
			server.HandleFunc("GET /replays", serveReplaysPage(replayStartedAt("a", now.Add(-time.Hour), 1, 1)))

			Expect(recentIDs(server, since, barlive.GameTypeDuel)).To(Equal([]string{"a"}))
			Expect(server.recorded()).To(HaveLen(2))
		})

		It("yields an error for a game type without preset and sends no request", func() {
			var errs []error
			for _, err := range server.client().RecentReplays(context.Background(), "Player002", "Team FFA", since, nil) {
				errs = append(errs, err)
			}

			Expect(errs).To(ConsistOf(MatchError(`no replay preset for game type "Team FFA"`)))
			Expect(server.recorded()).To(BeEmpty())
		})

		It("yields the error of a request", func() {
			server.HandleFunc("GET /replays", serveTestdata(http.StatusBadRequest, "replays_bad_request.json"))

			var errs []error
			for _, err := range server.client().RecentReplays(
				context.Background(),
				"Player002",
				barlive.GameTypeDuel,
				since,
				nil,
			) {
				errs = append(errs, err)
			}

			Expect(errs).To(ConsistOf(HaveOccurred()))
		})
	})
})

// serveReplaysPage answers with a full first page holding replays, and an empty page after it.
func serveReplaysPage(replays ...*barlive.ReplaySummary) http.HandlerFunc {
	page := barlive.ReplaysPage{TotalResults: -1, Page: 1, Limit: max(len(replays), 1)}
	for _, replay := range replays {
		page.Data = append(page.Data, *replay)
	}
	data, err := json.Marshal(page)
	Expect(err).NotTo(HaveOccurred())
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("page") != "1" {
			serveTestdata(http.StatusOK, "replays_page_empty.json")(writer, request)
			return
		}
		serveString(http.StatusOK, "application/json", string(data))(writer, request)
	}
}

// replayStartedAt returns a replay with ID id which started at startTime and has allyteams of the given sizes.
func replayStartedAt(id string, startTime time.Time, allyTeamSizes ...int) *barlive.ReplaySummary {
	replay := testReplaySummary(allyTeamSizes...)
	replay.ID = id
	replay.StartTime = startTime
	return replay
}

// recentIDs returns the IDs of the replays RecentReplays returns for Player002 on maps from server, and fails the spec
// on an error.
func recentIDs(server *testServer, since time.Time, gameType barlive.GameType, maps ...string) []string {
	var ids []string
	for replay, err := range server.client().RecentReplays(context.Background(), "Player002", gameType, since, maps) {
		Expect(err).NotTo(HaveOccurred())
		ids = append(ids, replay.ID)
	}
	return ids
}
