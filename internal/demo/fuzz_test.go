package demo_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// FuzzReadUncompressed feeds ReadUncompressed with mutations of valid demos. Besides not panicking, every demo it
// accepts must satisfy what its validation promises, encode to JSON and encode back to the same bytes.
func FuzzReadUncompressed(f *testing.F) {
	for _, fuzzDemo := range fuzzSeeds() {
		data, err := fuzzDemo.encode()
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		result, err := demo.ReadUncompressed(bytes.NewReader(data))
		if err != nil {
			if result != nil {
				t.Fatalf("got a result together with the error %v", err)
			}
			return
		}
		if result == nil {
			t.Fatal("got neither a result nor an error")
		}

		checkConsistent(t, result)
		checkJSON(t, result)
		checkRoundTrip(t, data, result)
	})
}

// fuzzSeeds returns valid demos that reach every section of the parser.
func fuzzSeeds() []*testDemo {
	complete := newTestDemo()

	crashed := newTestDemo()
	crashed.winningAllyTeams = nil
	crashed.playerStats = nil
	crashed.teamStats = nil
	crashed.updateSizes()
	crashed.header.DemoStreamSize = 0

	crashedWithStream := newTestDemo()
	crashedWithStream.header.DemoStreamSize = 0

	empty := newTestDemo()
	empty.winningAllyTeams = nil
	empty.playerStats = nil
	empty.teamStats = nil
	empty.updateSizes()

	withoutEntries := newTestDemo()
	withoutEntries.teamStats[1] = nil
	withoutEntries.updateSizes()

	withHeaderExtension := newTestDemo()
	withHeaderExtension.headerExtension = []byte{1, 2, 3, 4}
	withHeaderExtension.updateSizes()

	return []*testDemo{complete, crashed, crashedWithStream, empty, withoutEntries, withHeaderExtension}
}

// checkConsistent checks that an accepted demo agrees with its header and its validation rules.
func checkConsistent(t *testing.T, result *demo.Demo) {
	t.Helper()
	header := result.FileHeader
	if len(result.SetupScript) != int(header.ScriptSize) {
		t.Fatalf("start script: %d bytes, want scriptSize %d", len(result.SetupScript), header.ScriptSize)
	}
	if strings.Contains(result.SetupScript, "\x00") {
		t.Fatal("start script: contains a NUL byte")
	}
	if !utf8.ValidString(result.SetupScript) {
		t.Fatal("start script: not valid UTF-8")
	}

	if header.DemoStreamSize == 0 {
		if result.WinningAllyTeams != nil || result.PlayerStats != nil || result.TeamStats != nil {
			t.Fatal("crashed demo: got statistics, want none")
		}
		return
	}

	if len(result.WinningAllyTeams) != int(header.WinningAllyTeamsSize) {
		t.Fatalf("winning allyteams: %d, want winningAllyTeamsSize %d", len(result.WinningAllyTeams),
			header.WinningAllyTeamsSize)
	}

	if len(result.PlayerStats) != int(header.NumPlayers) {
		t.Fatalf("player statistics: %d players, want numPlayers %d", len(result.PlayerStats), header.NumPlayers)
	}

	if len(result.TeamStats) != int(header.NumTeams) {
		t.Fatalf("team statistics: %d teams, want numTeams %d", len(result.TeamStats), header.NumTeams)
	}
	for team, history := range result.TeamStats {
		for i := 1; i < len(history); i++ {
			if history[i].Frame < history[i-1].Frame {
				t.Fatalf("team %d entry %d: frame %d is before the previous entry's frame %d", team, i,
					history[i].Frame, history[i-1].Frame)
			}
		}
	}
}

// checkJSON checks that an accepted demo encodes to JSON. json.Marshal fails on NaN and infinite numbers.
func checkJSON(t *testing.T, result *demo.Demo) {
	t.Helper()
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("encoding to JSON: %v", err)
	}
	if !json.Valid(encoded) {
		t.Fatal("encoding to JSON: not valid JSON")
	}
}

// checkRoundTrip checks that an accepted demo encodes back to data. The parser skips the header extension and the
// demo stream, so they are taken from data. In a demo of a crashed engine, everything after the start script is
// ignored, so it is taken from data too.
func checkRoundTrip(t *testing.T, data []byte, result *demo.Demo) {
	t.Helper()
	header := result.FileHeader
	fuzzDemo := &testDemo{
		header:           header,
		headerExtension:  data[binary.Size(demo.DemoFileHeader{}):header.HeaderSize],
		setupScript:      result.SetupScript,
		winningAllyTeams: result.WinningAllyTeams,
		playerStats:      result.PlayerStats,
		teamStats:        result.TeamStats,
	}
	streamStart := header.HeaderSize + header.ScriptSize
	if header.DemoStreamSize == 0 {
		fuzzDemo.trailingData = data[streamStart:]
	} else {
		fuzzDemo.demoStream = data[streamStart : streamStart+header.DemoStreamSize]
	}

	encoded, err := fuzzDemo.encode()
	if err != nil {
		t.Fatalf("encoding: %v", err)
	}
	if !bytes.Equal(encoded, data) {
		t.Fatalf("encoding: got %d bytes that differ from the %d input bytes", len(encoded), len(data))
	}
}
