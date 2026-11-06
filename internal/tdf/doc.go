// Package tdf reads setup scripts (script.txt, and the setup script in a demo file) in the TDF format, as the Recoil
// engine does in rts/System/TdfParser.cpp. The engine leaves the grammar to the Lua parser
// cont/base/springcontent/gamedata/parse_tdf.lua; Parse accepts and rejects the same scripts and produces the same
// names and values.
//
// A script is a sequence of elements, each either a value or a section:
//
//	key=value;
//	[name]
//	{
//		elements
//	}
//
// The rules, in the parser's terms:
//
//   - Whitespace is space, \t, \n, \v, \f and \r. Line comments (//) and block comments (/* */) may stand wherever
//     whitespace may. A "/*" without a closing "*/" is not a comment.
//   - A key is one or more bytes which are neither whitespace nor "=". Spaces and tabs may surround the "=", a newline
//     may not.
//   - A value runs up to the next ";" on the same line and keeps trailing spaces. A value in double quotes on one line
//     may contain ";" and loses its quotes; if anything but spaces and tabs stands between the closing quote and the
//     ";", the quotes are part of an unquoted value instead. One or more ";" end a value.
//   - A section name is one or more bytes which are not "]". Whitespace and comments may stand between the "]" and
//     the "{".
//   - Names of values and sections are lowercased (ASCII only, like Lua's string.lower); values keep their case.
//   - Within a section, a name is either a value or a section. A later element replaces an earlier one with the same
//     name, also if one is a value and the other a section; sections with the same name are not merged.
//   - Every value is a string; see Value for conversions.
//
// The engine passes the script to Lua as a long string, in which Lua turns "\r\n", "\n\r" and a lone "\r" into "\n".
// Parse does the same, so a "\r" never reaches the grammar.
//
// Parse deliberately differs from the engine in these points:
//
//   - The engine escapes "[[" and "]]" to embed the script in the long string, and unescapes them in values only. A
//     key or section name containing "[[" or "]]" thus breaks or changes there, and a value containing "\]\]" comes
//     out as "]]". Parse reads such scripts as written.
//   - The engine stops reading at a NUL byte; Parse rejects one, and also invalid UTF-8, so that every result can be
//     written as JSON and read back unchanged.
//   - Sections may be nested at most MaxDepth deep. The engine is limited only by Lua's call depth.
//
// The order of elements is not kept: the engine keeps them in hash maps, so the order carries no meaning. Iterating a
// Section yields names in sorted order.
//
// Values and sections are looked up by a location, names separated by LocationSeparator, as in the engine:
// Value(`game\ai0\options\profile`). Lookups ignore case. A name containing the separator cannot be looked up by
// location, but is still yielded when iterating.
package tdf
