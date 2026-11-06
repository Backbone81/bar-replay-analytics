package tdf

// LocationSeparator separates the names in a location.
// See TdfParser::GetLocationVector in rts/System/TdfParser.cpp.
const LocationSeparator = `\`

// MaxDepth limits how deep sections may be nested in Parse and UnmarshalJSON. Setup scripts nest three deep:
// [game] [ai0] [options].
var MaxDepth = 100
