package tdf

// LocationSeparator separates the names in a location.
// See TdfParser::GetLocationVector in rts/System/TdfParser.cpp.
const LocationSeparator = `\`

// MaxDepth limits how deep sections may be nested in Parse and UnmarshalJSON. The setup scripts of the engine nest
// three deep, e.g. [game] [ai0] [options].
var MaxDepth = 100
