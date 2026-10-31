package barlive_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

var _ = Describe("Leaderboard", func() {
	var server *testServer

	BeforeEach(func() {
		server = newTestServer()
		server.HandleFunc("GET /teiserver/api/public/leaderboard", serveTestdata(http.StatusOK, "leaderboard_seasons.json"))
		server.HandleFunc("GET /teiserver/api/public/leaderboard/3", serveTestdata(
			http.StatusOK, "leaderboard_season_3.json"))
		server.HandleFunc("GET /teiserver/api/public/leaderboard/99", serveTestdata(
			http.StatusBadRequest, "leaderboard_season_99.json"))
	})

	Context("LeaderboardSeasons", func() {
		It("returns the seasons", func() {
			seasons, err := server.client().LeaderboardSeasons(context.Background())

			Expect(err).NotTo(HaveOccurred())
			gameTypes := []barlive.GameType{
				barlive.GameTypeSmallTeam, barlive.GameTypeLargeTeam, barlive.GameTypeDuel, barlive.GameTypeFFA,
			}
			Expect(seasons).To(Equal([]barlive.LeaderboardSeason{
				{Season: 1, GameTypes: gameTypes},
				{Season: 2, GameTypes: gameTypes},
				{Season: 3, GameTypes: gameTypes},
			}))
		})
	})

	Context("Leaderboards", func() {
		It("returns the top 100 players of each game type", func() {
			leaderboards, err := server.client().Leaderboards(context.Background(), 3)

			Expect(err).NotTo(HaveOccurred())
			Expect(leaderboards).To(HaveLen(4))
			for _, leaderboard := range leaderboards {
				Expect(leaderboard.Players).To(HaveLen(100))
				for i := 1; i < len(leaderboard.Players); i++ {
					Expect(leaderboard.Players[i].Rating).To(BeNumerically("<=", leaderboard.Players[i-1].Rating))
				}
			}

			duel, found := barlive.FindLeaderboard(leaderboards, barlive.GameTypeDuel)
			Expect(found).To(BeTrue())
			Expect(duel.Players[0].ID).To(Equal(10000002))
			Expect(duel.Players[0].Name).To(Equal("Player002"))
			Expect(duel.Players[0].Rating).To(BeNumerically("~", 61.3966, 0.0001))
		})

		It("reports an unknown season", func() {
			Expect(server.client().Leaderboards(context.Background(), 99)).Error().To(HaveOccurred())
		})
	})

	Context("LeaderboardSeason", func() {
		It("returns the active season for 0", func() {
			season, err := server.client().LeaderboardSeason(context.Background(), 0)

			Expect(err).NotTo(HaveOccurred())
			Expect(season.Season).To(Equal(3))
			Expect(season.GameTypes).To(ContainElement(barlive.GameTypeDuel))
		})

		It("returns an earlier season", func() {
			season, err := server.client().LeaderboardSeason(context.Background(), 2)

			Expect(err).NotTo(HaveOccurred())
			Expect(season.Season).To(Equal(2))
		})

		It("reports an unknown season", func() {
			Expect(server.client().LeaderboardSeason(context.Background(), 4)).Error().To(
				MatchError("season 4: teiserver lists seasons 1 to 3"))
		})

		It("rejects a negative season without a request", func() {
			Expect(server.client().LeaderboardSeason(context.Background(), -1)).Error().To(HaveOccurred())
			Expect(server.recorded()).To(BeEmpty())
		})

		It("reports a list without seasons", func() {
			emptyServer := newTestServer()
			emptyServer.HandleFunc("GET /teiserver/api/public/leaderboard", serveString(
				http.StatusOK, "application/json", "[]"))

			Expect(emptyServer.client().LeaderboardSeason(context.Background(), 0)).Error().To(HaveOccurred())
		})
	})

	Context("Leaderboard", func() {
		It("returns the leaderboard of a game type", func() {
			leaderboard, err := server.client().Leaderboard(context.Background(), 3, barlive.GameTypeFFA)

			Expect(err).NotTo(HaveOccurred())
			Expect(leaderboard.Name).To(Equal(barlive.GameTypeFFA))
			Expect(leaderboard.Players).To(HaveLen(100))
		})

		It("reports a game type without leaderboard", func() {
			Expect(server.client().Leaderboard(context.Background(), 3, "Team FFA")).Error().To(
				MatchError(`season 3 has no leaderboard for game type "Team FFA"`))
		})

		It("reports an unknown season", func() {
			Expect(server.client().Leaderboard(context.Background(), 99, barlive.GameTypeDuel)).Error().To(HaveOccurred())
		})
	})

	Context("FindLeaderboard", func() {
		It("reports a missing game type", func() {
			_, found := barlive.FindLeaderboard([]barlive.Leaderboard{{Name: barlive.GameTypeFFA}}, barlive.GameTypeDuel)

			Expect(found).To(BeFalse())
		})
	})
})
