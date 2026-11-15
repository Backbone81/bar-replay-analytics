package barlive_test

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/backbone81/bar-replay-analytics/internal/barlive"
	"github.com/backbone81/bar-replay-analytics/internal/barlive/barlivetest"
)

var _ = Describe("Replay", func() {
	var server *barlivetest.Server

	BeforeEach(func() {
		server = newServer()
		server.HandleFunc("GET /replays/00000000000000000000000000000001", barlivetest.ServeJSONFile(
			http.StatusOK, filepath.Join("testdata", "replay_00000000000000000000000000000001.json")))
		server.HandleFunc("GET /replays/00000000000000000000000000000002", barlivetest.ServeJSONFile(
			http.StatusOK, filepath.Join("testdata", "replay_00000000000000000000000000000002.json")))
		server.HandleFunc("GET /replays/{id}", barlivetest.ServeJSONFile(
			http.StatusNotFound, filepath.Join("testdata", "replay_not_found.json")))
	})

	It("returns the details of a replay", func() {
		replay, err := server.Client().Replay(context.Background(), "00000000000000000000000000000001")

		Expect(err).NotTo(HaveOccurred())
		Expect(replay.ID).To(Equal("00000000000000000000000000000001"))
		Expect(replay.FileName).To(Equal("2026-09-10_00-14-08-789_Cape_Violet_V1_2026.07.04.sdfz"))
		Expect(replay.EngineVersion).To(Equal("2026.07.04"))
		Expect(replay.GameVersion).To(Equal("Beyond All Reason test-31251-0ceadc7"))
		Expect(replay.StartTime).To(BeTemporally("==", time.Date(2026, 9, 10, 0, 14, 8, 0, time.UTC)))
		Expect(replay.Duration()).To(Equal(890333 * time.Millisecond))
		Expect(replay.FullDurationMs).To(Equal(int64(1007000)))
		Expect(replay.HostSettings).To(HaveKeyWithValue("mapname", "Cape_Violet_V1"))
		Expect(replay.GameSettings).To(HaveKeyWithValue("startmetal", "1000"))
		Expect(replay.MapSettings).To(Equal(map[string]string{"waterlevel": "0"}))
		Expect(replay.SpadsSettings).To(HaveKeyWithValue("teamsize", "1"))
		Expect(replay.GameEndedNormally).To(BeTrue())
		Expect(replay.HasBots).To(BeFalse())
		Expect(replay.Preset).To(Equal(barlive.ReplayPresetDuel))
		Expect(replay.Reported).To(BeFalse())
		Expect(replay.MapID).To(Equal(131))
		Expect(replay.Map).To(Equal(barlive.Map{
			ID: 131, ScriptName: "Cape_Violet_V1", FileName: "cape_violet_v1", Width: 20, Height: 10,
		}))

		Expect(replay.Awards).NotTo(BeNil())
		Expect(replay.Awards.EconDestroyed).To(Equal([]barlive.Award{
			{TeamID: 0, Value: 50750}, {TeamID: 1, Value: 5510}, {TeamID: -1, Value: 0},
		}))
		Expect(replay.Awards.MostResourcesProduced).To(Equal(barlive.Award{TeamID: 0, Value: 4161653.75}))
		Expect(replay.Awards.Cow).To(Equal(barlive.Award{TeamID: -1}))

		Expect(replay.AllyTeams).To(HaveLen(2))
		allyTeam := replay.AllyTeams[0]
		Expect(allyTeam.ID).To(Equal(5970873))
		Expect(allyTeam.AllyTeamID).To(Equal(0))
		Expect(allyTeam.StartBox).To(Equal(&barlive.StartBox{Top: 0, Bottom: 1, Left: 0.8, Right: 1}))
		Expect(allyTeam.WinningTeam).To(BeTrue())
		Expect(allyTeam.DemoID).To(Equal(replay.ID))
		Expect(allyTeam.AIs).To(BeEmpty())
		Expect(allyTeam.Players).To(HaveLen(1))

		player := allyTeam.Players[0]
		Expect(player.ID).To(Equal(23425337))
		Expect(player.PlayerID).To(Equal(0))
		Expect(player.Name).To(Equal("Player002"))
		Expect(player.TeamID).To(Equal(0))
		Expect(player.Faction).To(Equal("Cortex"))
		Expect(player.CountryCode).To(Equal("XX"))
		Expect(player.RGBColor).To(Equal(&barlive.Color{R: 11, G: 62, B: 243}))
		Expect(player.Rank).To(HaveValue(Equal(7)))
		Expect(player.SkillUncertainty).To(HaveValue(Equal(3.58)))
		Expect(player.Skill).To(Equal("[67.41]"))
		Expect(player.TrueSkill).To(BeNil())
		Expect(player.StartPos).To(Equal(&barlive.Position{X: 1726, Y: 81.325927734375, Z: 1867}))
		Expect(player.ClanID).To(BeNil())
		Expect(player.AllyTeamRecordID).To(Equal(allyTeam.ID))
		Expect(player.UserID).To(Equal(10000002))

		Expect(replay.Spectators).To(HaveLen(5))
		Expect(replay.Spectators[0].Name).To(Equal("Player299"))
		Expect(replay.Spectators[0].UserID).To(Equal(10000299))
		Expect(replay.Spectators[0].Skill).To(Equal("[33.32]"))
	})

	It("returns the AIs of a replay", func() {
		replay, err := server.Client().Replay(context.Background(), "00000000000000000000000000000002")

		Expect(err).NotTo(HaveOccurred())
		Expect(replay.HasBots).To(BeTrue())
		Expect(replay.AllyTeams).To(HaveLen(2))
		Expect(replay.AllyTeams[1].Players).To(BeEmpty())
		Expect(replay.AllyTeams[1].AIs).To(Equal([]barlive.AI{{
			ID:               3910232,
			AIID:             0,
			ShortName:        "BARb",
			Name:             "BARbarianAI(1)",
			Host:             false,
			StartPos:         nil,
			Handicap:         0,
			Faction:          "Random",
			RGBColor:         nil,
			CreatedAt:        time.Date(2026, 10, 28, 17, 6, 55, 464000000, time.UTC),
			UpdatedAt:        time.Date(2026, 10, 28, 17, 6, 55, 464000000, time.UTC),
			AllyTeamRecordID: replay.AllyTeams[1].ID,
		}}))
	})

	It("reports an unknown replay", func() {
		Expect(server.Client().Replay(context.Background(), "00000000000000000000000000000000")).Error().To(HaveOccurred())
	})

	It("escapes the ID", func() {
		Expect(server.Client().Replay(context.Background(), "../a b")).Error().To(HaveOccurred())
		Expect(server.Requests()[0].Path).To(Equal("/replays/..%2Fa%20b"))
	})

	It("rejects an empty ID without a request", func() {
		Expect(server.Client().Replay(context.Background(), "")).Error().To(HaveOccurred())
		Expect(server.Requests()).To(BeEmpty())
	})
})
