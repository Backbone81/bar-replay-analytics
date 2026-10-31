package demo

import (
	"strconv"
	"strings"
)

// versionBefore reports whether the engine version string, such as "2026.07.04", "2026.07.04-12-gabcdef0" or
// "105.1.1-2590-gb9462a0 BAR105", names a release before version. It compares the up to three numbers separated by
// dots at the start of the string; missing numbers count as 0. A string which does not start with a number names no
// release, so it is not before any version.
func versionBefore(versionString string, version [3]int) bool {
	end := strings.IndexFunc(versionString, func(r rune) bool {
		return (r < '0' || r > '9') && r != '.'
	})
	if end < 0 {
		end = len(versionString)
	}
	parts := strings.Split(versionString[:end], ".")
	if len(parts) > len(version) {
		parts = parts[:len(version)]
	}

	var numbers [3]int
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil {
			return false
		}
		numbers[i] = number
	}
	for i := range numbers {
		if numbers[i] != version[i] {
			return numbers[i] < version[i]
		}
	}
	return false
}
