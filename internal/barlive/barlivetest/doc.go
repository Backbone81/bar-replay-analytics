// Package barlivetest provides stand-ins for the BAR Live Services, for tests of barlive and of code which uses a
// barlive.Client, like net/http/httptest does for HTTP servers.
//
// Server is a local HTTP server which records every request. A test registers the handlers it needs on it, so it
// can send exact responses (real ones from files, errors, broken downloads). Fake is a Server which plays Teiserver,
// bar-db and the replay storage from the data in its fields. This is for tests which care about what the services hold,
// not about their responses.
package barlivetest
