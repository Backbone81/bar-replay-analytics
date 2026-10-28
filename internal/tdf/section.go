package tdf

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// Section is the content of a [name] { ... } block, or of a whole script.
//
// Every name is lowercased and is either a value or a section, never both: see TdfSection in
// rts/System/TdfParser.h. Only Parse and UnmarshalJSON create sections, which keeps it that way.
//
// A nil *Section reads as an empty section, so lookups can be chained without checks:
// script.Section(`game\ai0`).Section("options") is nil if there is no ai0.
type Section struct {
	values   map[string]Value
	sections map[string]*Section
}

// Value returns the value at path, e.g. `game\mapname`, ignoring case.
// Returns false if there is none.
func (s *Section) Value(path string) (Value, bool) {
	section, name := s.parent(path)
	if section == nil {
		return "", false
	}

	value, ok := section.values[name]
	return value, ok
}

// Section returns the section at path, e.g. `game\ai0\options`, ignoring case.
// Returns nil if there is none.
func (s *Section) Section(path string) *Section {
	section, name := s.parent(path)
	if section == nil {
		return nil
	}
	return section.sections[name]
}

// Values returns the values of this section, sorted by name. Subsections are not included.
func (s *Section) Values() iter.Seq2[string, Value] {
	if s == nil {
		return sortedByName[Value](nil)
	}

	return sortedByName(s.values)
}

// Sections returns the subsections of this section, sorted by name.
func (s *Section) Sections() iter.Seq2[string, *Section] {
	if s == nil {
		return sortedByName[*Section](nil)
	}

	return sortedByName(s.sections)
}

// Indexed returns the numbered sections at path, sorted by number: `game\player` returns player0, player1, ... of the
// section game. Numbers may have gaps, the engine numbers them anew without gaps. Like the engine, which looks them up
// with "%i", only plain decimal numbers count: player01 and player+1 do not.
//
// The engine reads players and AIs below MAX_PLAYERS (251), teams and allyteams below MAX_TEAMS (255), see
// rts/Game/GameSetup.cpp. Indexed returns every number.
func (s *Section) Indexed(path string) iter.Seq2[int, *Section] {
	section, prefix := s.parent(path)
	return func(yield func(int, *Section) bool) {
		if section == nil {
			return
		}

		indexed := make(map[int]*Section)
		for name, child := range section.sections {
			suffix, found := strings.CutPrefix(name, prefix)
			if !found {
				continue
			}
			if index, ok := parseIndex(suffix); ok {
				indexed[index] = child
			}
		}
		for _, index := range slices.Sorted(maps.Keys(indexed)) {
			if !yield(index, indexed[index]) {
				return
			}
		}
	}
}

// MarshalJSON writes the section as one JSON object: a value as a string, a subsection as a nested object, e.g.
// {"game":{"mapname":"Argent Strata 1.1","ai0":{"options":{"profile":"Hard"}}}}. A name is never both, so the object
// is unambiguous. A nil section is written as an empty object.
func (s *Section) MarshalJSON() ([]byte, error) {
	object := make(map[string]any)
	for name, value := range s.Values() {
		object[name] = value
	}
	for name, section := range s.Sections() {
		object[name] = section
	}
	return json.Marshal(object)
}

// UnmarshalJSON reads a section as MarshalJSON writes it. Like Parse, it lowercases names, and a later name replaces
// an earlier one which is the same after lowercasing. It accepts any names and values: they need not be ones a script
// could hold.
// Returns an error for an empty name, for anything but strings and objects, and for objects nested deeper than
// MaxDepth.
func (s *Section) UnmarshalJSON(data []byte) error {
	var section Section
	if err := section.unmarshalJSON(data, 0); err != nil {
		return err
	}
	*s = section
	return nil
}

// unmarshalJSON reads the members of a JSON object one by one, so that duplicate names are replaced in order.
func (s *Section) unmarshalJSON(data []byte, depth int) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("got %v, want a JSON object", token)
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		name, ok := token.(string)
		if !ok {
			return fmt.Errorf("got the name %v, want a string", token)
		}
		if name == "" {
			return errors.New("empty name")
		}

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return fmt.Errorf("%q: %w", name, err)
		}
		if err := s.unmarshalMember(name, bytes.TrimLeft(raw, " \t\r\n"), depth); err != nil {
			return fmt.Errorf("%q: %w", name, err)
		}
	}
	_, err = decoder.Token() // The closing "}".
	return err
}

// unmarshalMember adds the JSON value data under name: a string as a value, an object as a section.
func (s *Section) unmarshalMember(name string, data []byte, depth int) error {
	switch {
	case bytes.HasPrefix(data, []byte(`"`)):
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		s.setValue(name, Value(value))
	case bytes.HasPrefix(data, []byte("{")):
		if depth+1 > MaxDepth {
			return fmt.Errorf("nested deeper than %d sections", MaxDepth)
		}
		var section Section
		if err := section.unmarshalJSON(data, depth+1); err != nil {
			return err
		}
		s.setSection(name, &section)
	default:
		return errors.New("want a string or an object")
	}
	return nil
}

// setValue sets the value name, replacing a value or section of that name.
func (s *Section) setValue(name string, value Value) {
	name = lowerASCII(name)
	delete(s.sections, name)
	if s.values == nil {
		s.values = make(map[string]Value)
	}
	s.values[name] = value
}

// setSection sets the section name, replacing a value or section of that name.
func (s *Section) setSection(name string, section *Section) {
	name = lowerASCII(name)
	delete(s.values, name)
	if s.sections == nil {
		s.sections = make(map[string]*Section)
	}
	s.sections[name] = section
}

// parent returns the section holding the last name of path, and that name, lowercased.
// Returns a nil section if there is none.
func (s *Section) parent(path string) (*Section, string) {
	names := strings.Split(lowerASCII(path), PathSeparator)
	section := s
	for _, name := range names[:len(names)-1] {
		if section == nil {
			return nil, ""
		}
		section = section.sections[name]
	}
	return section, names[len(names)-1]
}

// sortedByName returns the entries of m sorted by name.
func sortedByName[V any](m map[string]V) iter.Seq2[string, V] {
	return func(yield func(string, V) bool) {
		for _, name := range slices.Sorted(maps.Keys(m)) {
			if !yield(name, m[name]) {
				return
			}
		}
	}
}

// parseIndex returns s as a section number: decimal digits without a sign or leading zeros, as "%i" prints them.
func parseIndex(s string) (int, bool) {
	if s == "" || (s[0] == '0' && len(s) > 1) {
		return 0, false
	}
	for i := range len(s) {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	index, err := strconv.Atoi(s)
	return index, err == nil
}

// lowerASCII lowercases the ASCII letters of s, like Lua's string.lower in the C locale. strings.ToLower would also
// lowercase other letters, which the engine keeps.
func lowerASCII(s string) string {
	if !strings.ContainsFunc(s, func(r rune) bool { return 'A' <= r && r <= 'Z' }) {
		return s
	}
	lowered := []byte(s)
	for i, c := range lowered {
		if 'A' <= c && c <= 'Z' {
			lowered[i] = c + 'a' - 'A'
		}
	}
	return string(lowered)
}
