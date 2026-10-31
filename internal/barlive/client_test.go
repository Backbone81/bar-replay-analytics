package barlive_test

import (
	"context"
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// maxRequestRate is the number of requests per second the client is limited to.
const maxRequestRate = 20

// interval is the time between two requests at maxRequestRate.
const interval = time.Second / maxRequestRate

var _ = Describe("Client", func() {
	var server *testServer

	BeforeEach(func() {
		server = newTestServer()
		server.HandleFunc("GET /teiserver/api/public/leaderboard", serveTestdata(http.StatusOK, "leaderboard_seasons.json"))
	})

	Context("requests", func() {
		It("asks for JSON", func() {
			Expect(server.client().LeaderboardSeasons(context.Background())).Error().NotTo(HaveOccurred())
			Expect(server.recorded()).To(HaveLen(1))
			Expect(server.recorded()[0].Header.Get("Accept")).To(Equal("application/json"))
		})

		It("reports a status other than 200 OK", func() {
			server.HandleFunc("GET /teiserver/api/public/leaderboard/1", serveString(
				http.StatusBadGateway, "text/html", "<html>Bad Gateway</html>"))

			Expect(server.client().Leaderboards(context.Background(), 1)).Error().To(HaveOccurred())
		})

		It("rejects a response which is not JSON", func() {
			server.HandleFunc("GET /teiserver/api/public/leaderboard/1", serveString(http.StatusOK, "text/html", "<html>"))

			Expect(server.client().Leaderboards(context.Background(), 1)).Error().To(HaveOccurred())
		})
	})

	Context("rate limiting", func() {
		It("waits between requests", func() {
			client := server.client(barlive.WithMaxRequestRate(maxRequestRate))

			start := time.Now()
			for range 3 {
				Expect(client.LeaderboardSeasons(context.Background())).Error().NotTo(HaveOccurred())
			}

			Expect(time.Since(start)).To(BeNumerically(">=", 2*interval))
			Expect(server.recorded()).To(HaveLen(3))
		})

		It("spreads concurrent requests out", func() {
			client := server.client(barlive.WithMaxRequestRate(maxRequestRate))

			start := time.Now()
			var waitGroup sync.WaitGroup
			for range 3 {
				waitGroup.Go(func() {
					defer GinkgoRecover()
					Expect(client.LeaderboardSeasons(context.Background())).Error().NotTo(HaveOccurred())
				})
			}
			waitGroup.Wait()

			Expect(time.Since(start)).To(BeNumerically(">=", 2*interval))
			Expect(server.recorded()).To(HaveLen(3))
		})

		It("does not wait past the deadline of the context", func() {
			client := server.client(barlive.WithMaxRequestRate(1.0 / 3600))

			Expect(client.LeaderboardSeasons(context.Background())).Error().NotTo(HaveOccurred())

			ctx, cancel := context.WithTimeout(context.Background(), interval)
			defer cancel()

			Expect(client.LeaderboardSeasons(ctx)).Error().To(HaveOccurred())
			Expect(server.recorded()).To(HaveLen(1))
		})

		It("sends nothing with a context which is already done", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			Expect(server.client().LeaderboardSeasons(ctx)).Error().To(MatchError(context.Canceled))
			Expect(server.recorded()).To(BeEmpty())
		})
	})
})
