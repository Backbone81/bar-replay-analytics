package barlivetest

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/backbone81/bar-replay-analytics-go/internal/barlive"
)

// Fake is a Server which plays Teiserver, bar-db and the replay storage from the data in its fields. Set the fields
// before the client sends its first request, and do not change them while it runs.
type Fake struct {
	*Server

	// Season is the active season; Teiserver lists the seasons 1 to Season, all with GameTypes.
	Season int

	// GameTypes are the game types with a leaderboard.
	GameTypes []barlive.GameType

	// Leaderboards holds the players of the leaderboard of each game type, best first, in every season.
	Leaderboards map[barlive.GameType][]barlive.LeaderboardPlayer

	// Replays holds the replays a bar-db search finds by player name, newest first. The search looks at the first
	// player of the query only, and at nothing else of it but its page and limit.
	Replays map[string][]barlive.ReplaySummary

	// FileNames holds the file names bar-db returns by replay ID; bar-db does not find any other replay.
	FileNames map[string]string

	// Demos holds the demo files of the storage by file name.
	Demos map[string][]byte
}

// NewFake starts a Fake with one season of the four game types and no players or replays. Close it when done.
func NewFake() *Fake {
	fake := &Fake{
		Server: NewServer(),
		Season: 1,
		GameTypes: []barlive.GameType{
			barlive.GameTypeDuel, barlive.GameTypeSmallTeam, barlive.GameTypeLargeTeam, barlive.GameTypeFFA,
		},
		Leaderboards: make(map[barlive.GameType][]barlive.LeaderboardPlayer),
		Replays:      make(map[string][]barlive.ReplaySummary),
		FileNames:    make(map[string]string),
		Demos:        make(map[string][]byte),
	}
	fake.HandleFunc("GET /teiserver/api/public/leaderboard", fake.serveSeasons)
	fake.HandleFunc("GET /teiserver/api/public/leaderboard/{season}", fake.serveLeaderboards)
	fake.HandleFunc("GET /replays", fake.serveReplays)
	fake.HandleFunc("GET /replays/{id}", fake.serveReplay)
	fake.HandleFunc("GET "+StoragePath+"/demos/{fileName}", fake.serveDemo)
	return fake
}

// AddReplay adds a game with ID id and the allyteam sizes, which each of players finds in bar-db, under fileName,
// and its demo file to the storage.
func (f *Fake) AddReplay(id string, fileName string, allyTeamSizes []int, players ...string) error {
	demoFile, err := DemoFile(id)
	if err != nil {
		return err
	}
	replay := ReplaySummary(id, allyTeamSizes...)
	for _, player := range players {
		f.Replays[player] = append(f.Replays[player], replay)
	}
	f.FileNames[id] = fileName
	f.Demos[fileName] = demoFile
	return nil
}

// StorageRequests returns the number of requests the storage received so far.
func (f *Fake) StorageRequests() int {
	count := 0
	for _, request := range f.Requests() {
		if strings.HasPrefix(request.Path, StoragePath+"/") {
			count++
		}
	}
	return count
}

func (f *Fake) serveSeasons(writer http.ResponseWriter, request *http.Request) {
	seasons := make([]barlive.LeaderboardSeason, f.Season)
	for i := range seasons {
		seasons[i] = barlive.LeaderboardSeason{Season: i + 1, GameTypes: f.GameTypes}
	}
	ServeJSON(http.StatusOK, seasons)(writer, request)
}

func (f *Fake) serveLeaderboards(writer http.ResponseWriter, request *http.Request) {
	season, err := strconv.Atoi(request.PathValue("season"))
	if err != nil || season < 1 || season > f.Season {
		http.Error(writer, "no such season", http.StatusBadRequest)
		return
	}
	leaderboards := make([]barlive.Leaderboard, 0, len(f.GameTypes))
	for _, gameType := range f.GameTypes {
		leaderboards = append(leaderboards, barlive.Leaderboard{Name: gameType, Players: f.Leaderboards[gameType]})
	}
	ServeJSON(http.StatusOK, leaderboards)(writer, request)
}

// serveReplays answers a search with one page, with bar-db's defaults for a missing page or limit.
func (f *Fake) serveReplays(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	page, pageErr := intOrDefault(query.Get("page"), 1)
	limit, limitErr := intOrDefault(query.Get("limit"), 24) //nolint:mnd // bar-db's default
	if pageErr != nil || limitErr != nil || page < 1 || limit < 1 {
		http.Error(writer, "bad page or limit", http.StatusBadRequest)
		return
	}

	replays := f.Replays[query.Get("players")]
	start := min((page-1)*limit, len(replays))
	end := min(start+limit, len(replays))
	ServeJSON(http.StatusOK, barlive.ReplaysPage{
		TotalResults: -1,
		Page:         page,
		Limit:        limit,
		Data:         append([]barlive.ReplaySummary{}, replays[start:end]...),
	})(writer, request)
}

func (f *Fake) serveReplay(writer http.ResponseWriter, request *http.Request) {
	id := request.PathValue("id")
	fileName, found := f.FileNames[id]
	if !found {
		ServeString(http.StatusNotFound, "application/json; charset=utf-8",
			`{"statusCode":404,"error":"Not Found","message":"Not Found"}`)(writer, request)
		return
	}
	ServeJSON(http.StatusOK, barlive.Replay{ID: id, FileName: fileName})(writer, request)
}

func (f *Fake) serveDemo(writer http.ResponseWriter, request *http.Request) {
	content, found := f.Demos[request.PathValue("fileName")]
	if !found {
		http.NotFound(writer, request)
		return
	}
	ServeDemo(content)(writer, request)
}

// intOrDefault returns the number value, or defaultValue if value is empty.
func intOrDefault(value string, defaultValue int) (int, error) {
	if value == "" {
		return defaultValue, nil
	}
	return strconv.Atoi(value)
}
