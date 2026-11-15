package player

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// widgetOrder is the order a widget gets in the widget config file: the one BAR's widget handler gives a widget it
// enables without an entry in the order list (widgetHandler:LoadWidget in luaui/barwidgets.lua).
const widgetOrder = 12345

// widgetNamePattern matches the name in a widget's GetInfo as a Lua string literal in double or single quotes, quotes
// included. It has no escapes and no line breaks (Lua 5.1 rejects a raw CR or LF in a short string), so the literal is
// valid Lua as it is.
var widgetNamePattern = regexp.MustCompile(
	`(?s)function\s+widget:GetInfo\s*\(\s*\).*?\bname\s*=\s*("[^"\\\r\n]*"|'[^'\\\r\n]*')`,
)

// ReadWidgetNameLiteral returns the name the widget source gives in its GetInfo as the Lua string literal it is written
// as, quotes included, such as "Advisor", and whether it gives one. It only reads the text, it does not run the widget.
func ReadWidgetNameLiteral(source []byte) (string, bool) {
	match := widgetNamePattern.FindSubmatch(source)
	if match == nil {
		return "", false
	}
	return string(match[1]), true
}

// UserWidgetNameLiterals returns the names of the user widgets in widgetsDir as Lua string literals (see
// ReadWidgetNameLiteral), sorted: of the .lua files directly in it and one subfolder deep, which BAR's widget handler
// loads (loadWidgetFiles in luaui/barwidgets.lua). Files without a name in their GetInfo are left out. A missing
// widgetsDir has none.
func UserWidgetNameLiterals(widgetsDir string) ([]string, error) {
	var filePaths []string
	for _, pattern := range []string{"*.lua", filepath.Join("*", "*.lua")} {
		matches, err := filepath.Glob(filepath.Join(widgetsDir, pattern))
		if err != nil {
			return nil, err
		}
		filePaths = append(filePaths, matches...)
	}

	var literals []string
	for _, filePath := range filePaths {
		source, err := os.ReadFile(filePath) //nolint:gosec // The file path is under our control
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if literal, ok := ReadWidgetNameLiteral(source); ok {
			literals = append(literals, literal)
		}
	}
	slices.Sort(literals)
	return slices.Compact(literals), nil
}

// WidgetConfig returns a widget handler config (see utils.WidgetHandlerConfigFilePath) whose order list enables the
// widgets named by nameLiterals, Lua string literals written into it as they are. BAR's widget handler before commit
// 50fbf4d02f (2026-07-15) enables a user widget only if the order list has it, so without this file the playback would
// run none of them. A name given twice, also in different quotes, is a repeated key, which Lua allows.
func WidgetConfig(nameLiterals []string) []byte {
	nameLiterals = slices.Clone(nameLiterals)
	slices.Sort(nameLiterals)
	nameLiterals = slices.Compact(nameLiterals)

	var builder strings.Builder
	builder.WriteString("-- Widget order written by bar-replay-analytics for the playback\nreturn {\n\torder = {\n")
	for _, literal := range nameLiterals {
		fmt.Fprintf(&builder, "\t\t[%s] = %d,\n", literal, widgetOrder)
	}
	builder.WriteString("\t},\n}\n")
	return []byte(builder.String())
}
