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
			gameTypes := []string{
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

	Context("FindLeaderboard", func() {
		It("reports a missing game type", func() {
			_, found := barlive.FindLeaderboard([]barlive.Leaderboard{{Name: barlive.GameTypeFFA}}, barlive.GameTypeDuel)

			Expect(found).To(BeFalse())
		})
	})
})
