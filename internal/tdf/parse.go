package tdf

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ParseError is a syntax error in a script.
type ParseError struct {
	// Line is the line of the error, counting from 1.
	Line int

	// Message describes the error.
	Message string
}

// Error returns the line and the message.
func (e *ParseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// Parse parses a script, see the package documentation for the rules.
// Returns a *ParseError for a syntax error, and an error for a NUL byte or invalid UTF-8.
func Parse(script string) (*Section, error) {
	if i := strings.IndexByte(script, 0); i >= 0 {
		return nil, fmt.Errorf("NUL byte at offset %d", i)
	}
	if !utf8.ValidString(script) {
		return nil, errors.New("not valid UTF-8")
	}

	p := parser{text: normalizeNewlines(script)}
	root := &Section{}
	closed, err := p.parseElements(root, 0)
	if err != nil {
		return nil, err
	}
	if closed {
		return nil, p.errorAt(p.pos-1, `"}" outside of a section`)
	}
	return root, nil
}

// parser holds the state of Parse: the script and the offset of the next byte to read.
type parser struct {
	text string
	pos  int
}

// parseElements adds values and sections to section until the end of the text or a "}", which it consumes.
// Returns whether it stopped at a "}". depth is the nesting depth of section; the whole script is depth 0.
func (p *parser) parseElements(section *Section, depth int) (bool, error) {
	for {
		p.skipWhitespace()
		if p.pos == len(p.text) {
			return false, nil
		}

		switch p.text[p.pos] {
		case '}':
			p.pos++
			return true, nil
		case '[':
			name, child, err := p.parseSection(depth + 1)
			if err != nil {
				return false, err
			}
			section.setSection(name, child)
		default:
			name, value, err := p.parseValue()
			if err != nil {
				return false, err
			}
			section.setValue(name, value)
		}
	}
}

// parseSection parses "[name] { elements }", starting at the "[". depth is the nesting depth of the new section.
func (p *parser) parseSection(depth int) (string, *Section, error) {
	start := p.pos
	length := strings.IndexByte(p.text[start+1:], ']')
	if length < 0 {
		return "", nil, p.errorAt(start, `section: missing "]"`)
	}
	if length == 0 {
		return "", nil, p.errorAt(start, "section: empty name")
	}
	name := p.text[start+1 : start+1+length]
	p.pos = start + 1 + length + 1
	if depth > MaxDepth {
		return "", nil, p.errorAt(start, fmt.Sprintf("section %q: nested deeper than %d sections", name, MaxDepth))
	}

	p.skipWhitespace()
	if p.pos == len(p.text) || p.text[p.pos] != '{' {
		return "", nil, p.errorAt(p.pos, fmt.Sprintf(`section %q: missing "{"`, name))
	}
	p.pos++

	section := &Section{}
	closed, err := p.parseElements(section, depth)
	if err != nil {
		return "", nil, err
	}
	if !closed {
		return "", nil, p.errorAt(start, fmt.Sprintf(`section %q: missing "}"`, name))
	}
	return name, section, nil
}

// parseValue parses "key=value;", starting at the key.
func (p *parser) parseValue() (string, Value, error) {
	start := p.pos
	end := start
	for end < len(p.text) && !isWhitespace(p.text[end]) && p.text[end] != '=' {
		end++
	}
	if end == start {
		return "", "", p.errorAt(start, `missing key before "="`)
	}
	key := p.text[start:end]

	p.pos = p.skipBlanks(end)
	if p.pos == len(p.text) || p.text[p.pos] != '=' {
		return "", "", p.errorAt(start, fmt.Sprintf(`key %q: missing "="`, key))
	}
	p.pos = p.skipBlanks(p.pos + 1)

	if value, ok := p.quotedValue(); ok {
		return key, value, nil
	}

	rest := p.text[p.pos:]
	length := strings.IndexAny(rest, ";\n")
	if length < 0 || rest[length] != ';' {
		return "", "", p.errorAt(p.pos, fmt.Sprintf(`key %q: missing ";" after the value`, key))
	}
	value := Value(rest[:length])
	p.pos = p.skipSemicolons(p.pos + length)
	return key, value, nil
}

// quotedValue parses a value in double quotes on one line, followed by blanks and ";". If the value is not like that,
// it reads nothing and returns false.
func (p *parser) quotedValue() (Value, bool) {
	rest := p.text[p.pos:]
	if !strings.HasPrefix(rest, `"`) {
		return "", false
	}

	length := strings.IndexAny(rest[1:], "\"\n")
	if length < 0 || rest[1+length] != '"' {
		return "", false
	}

	end := p.skipBlanks(p.pos + 1 + length + 1)
	if end == len(p.text) || p.text[end] != ';' {
		return "", false
	}

	value := Value(rest[1 : 1+length])
	p.pos = p.skipSemicolons(end)
	return value, true
}

// skipWhitespace skips whitespace, line comments and closed block comments.
func (p *parser) skipWhitespace() {
	for {
		start := p.pos
		for p.pos < len(p.text) && isWhitespace(p.text[p.pos]) {
			p.pos++
		}

		rest := p.text[p.pos:]
		switch {
		case strings.HasPrefix(rest, "//"):
			if length := strings.IndexByte(rest, '\n'); length >= 0 {
				p.pos += length
			} else {
				p.pos = len(p.text)
			}
		case strings.HasPrefix(rest, "/*"):
			if length := strings.Index(rest[2:], "*/"); length >= 0 {
				p.pos += 2 + length + 2
			}
		}

		if p.pos == start {
			return
		}
	}
}

// skipBlanks returns the offset of the first byte from pos on which is neither a space nor a tab.
func (p *parser) skipBlanks(pos int) int {
	for pos < len(p.text) && (p.text[pos] == ' ' || p.text[pos] == '\t') {
		pos++
	}
	return pos
}

// skipSemicolons returns the offset of the first byte from pos on which is not a ";".
func (p *parser) skipSemicolons(pos int) int {
	for pos < len(p.text) && p.text[pos] == ';' {
		pos++
	}
	return pos
}

// errorAt returns a ParseError at the line holding offset pos.
func (p *parser) errorAt(pos int, message string) *ParseError {
	return &ParseError{
		Line:    1 + strings.Count(p.text[:pos], "\n"),
		Message: message,
	}
}

// isWhitespace reports whether c is whitespace like Lua's %s in the C locale.
func isWhitespace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	default:
		return false
	}
}

// normalizeNewlines turns "\r\n", "\n\r" and a lone "\r" into "\n", as Lua does in the long string through which the
// engine passes the script, see read_long_string and inclinenumber in rts/lib/lua/src/llex.cpp.
func normalizeNewlines(text string) string {
	if !strings.Contains(text, "\r") {
		return text
	}
	var result bytes.Buffer
	result.Grow(len(text))
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c != '\n' && c != '\r' {
			result.WriteByte(c)
			continue
		}
		result.WriteByte('\n')
		if i+1 < len(text) && (text[i+1] == '\n' || text[i+1] == '\r') && text[i+1] != c {
			i++
		}
	}
	return result.String()
}
