package tdf

// PathSeparator separates the names in a path.
// See TdfParser::GetLocationVector in rts/System/TdfParser.cpp.
const PathSeparator = `\`

// MaxDepth limits how deep sections may be nested in Parse and UnmarshalJSON. The start scripts of the engine nest
// three deep, e.g. [game] [ai0] [options].
var MaxDepth = 100
