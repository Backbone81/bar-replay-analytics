package tdf_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/backbone81/bar-replay-analytics-go/internal/tdf"
)

// FuzzParse feeds Parse with mutations of valid and invalid scripts. Besides not panicking, every script it accepts
// must keep the invariants of Section and encode to JSON and back unchanged; every script it rejects must be rejected
// with a ParseError, unless it holds a NUL byte or invalid UTF-8.
func FuzzParse(f *testing.F) {
	for _, script := range fuzzScripts() {
		f.Add(script)
	}

	f.Fuzz(func(t *testing.T, script string) {
		result, err := tdf.Parse(script)
		if err != nil {
			if result != nil {
				t.Fatalf("got a result together with the error %v", err)
			}
			checkParseError(t, script, err)
			return
		}
		if result == nil {
			t.Fatal("got neither a result nor an error")
		}

		checkInvariants(t, result, 0)
		checkJSONRoundTrip(t, result)
	})
}

// FuzzUnmarshalJSON feeds Section.UnmarshalJSON with mutations of JSON objects. Every object it accepts must keep the
// invariants of Section and encode to JSON and back unchanged.
func FuzzUnmarshalJSON(f *testing.F) {
	for _, script := range fuzzScripts() {
		result, err := tdf.Parse(script)
		if err != nil {
			continue
		}

		encoded, err := json.Marshal(result)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(encoded)
	}
	f.Add([]byte(`{"A":"x","a":{"B":"y"},"":"z","n":1}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var result tdf.Section
		if err := json.Unmarshal(data, &result); err != nil {
			return
		}

		checkInvariants(t, &result, 0)
		checkJSONRoundTrip(t, &result)
	})
}

// fuzzScripts returns scripts that reach every rule of the parser, valid ones and invalid ones.
func fuzzScripts() []string {
	return []string{
		testScript,
		"",
		"a = b ;;; c=\"x;y\" ; d=\"x\" y;",
		"[A B]\r\n// comment\r\n{\r\n/* block\n */ KEY=Value;\n}\n",
		"a=1;[a]{b=2;}[A]{c=3;}a=4;",
		"[a]{[b]{[c]{}}}",
		"/*a=1;",
		"a=\"x;\nb=c;",
		"ÄB=Ä;",
		"[a]{",
		"}",
		"a=b",
		"[]{}",
	}
}

// checkParseError checks that err is a ParseError with a line of the script, or one of the two other errors.
func checkParseError(t *testing.T, script string, err error) {
	t.Helper()
	if parseErr, ok := errors.AsType[*tdf.ParseError](err); ok {
		lines := 1 + strings.Count(script, "\n") + strings.Count(script, "\r")
		if parseErr.Line < 1 || parseErr.Line > lines {
			t.Fatalf("got line %d, want 1 to %d", parseErr.Line, lines)
		}
		return
	}
	if !strings.Contains(script, "\x00") && !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("got %v, want a ParseError", err)
	}
}

// checkInvariants checks that every name of section is not empty, lowercased, and either a value or a section, and
// that sections are nested at most MaxDepth deep.
func checkInvariants(t *testing.T, section *tdf.Section, depth int) {
	t.Helper()
	if depth > tdf.MaxDepth {
		t.Fatalf("got sections nested %d deep, want at most %d", depth, tdf.MaxDepth)
	}

	names := make(map[string]bool)
	for name := range section.Values() {
		checkName(t, name)
		names[name] = true
	}
	for name, child := range section.Sections() {
		checkName(t, name)
		if names[name] {
			t.Fatalf("%q is both a value and a section", name)
		}
		if child == nil {
			t.Fatalf("section %q is nil", name)
		}
		checkInvariants(t, child, depth+1)
	}
}

// checkName checks that name is not empty and has no upper case ASCII letters.
func checkName(t *testing.T, name string) {
	t.Helper()
	if name == "" {
		t.Fatal("got an empty name")
	}
	if strings.ContainsAny(name, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		t.Fatalf("name %q is not lowercased", name)
	}
}

// checkJSONRoundTrip checks that section encodes to JSON, decodes back, and encodes to the same JSON again.
func checkJSONRoundTrip(t *testing.T, section *tdf.Section) {
	t.Helper()
	encoded, err := json.Marshal(section)
	if err != nil {
		t.Fatalf("encoding to JSON: %v", err)
	}

	var decoded tdf.Section
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decoding from JSON: %v", err)
	}

	reencoded, err := json.Marshal(&decoded)
	if err != nil {
		t.Fatalf("encoding to JSON again: %v", err)
	}
	if !bytes.Equal(reencoded, encoded) {
		t.Fatalf("JSON changed from %s to %s", encoded, reencoded)
	}
}
