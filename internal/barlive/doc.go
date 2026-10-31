// Package barlive is a client for the web services behind BAR Live Services (https://bar-rts.com): the leaderboards,
// the replay database and the replay storage.
//
// None of these services is documented. Their behavior was taken from their source code and checked against their
// responses (testdata/ holds real ones, with the players pseudonymized by scripts/anonymize_testdata.py):
//
//   - Teiserver, BAR's lobby server, serves the leaderboards: GET /teiserver/api/public/leaderboard lists the seasons,
//     GET /teiserver/api/public/leaderboard/<season> returns up to 100 of the best recently active players of each
//     game type. The leaderboard page on bar-rts.com fetches the same data through a proxy at
//     https://bar-rts.com/api/leaderboard (beyond-all-reason/bar-live-services, serverMiddleware/proxy.js).
//   - bar-db (beyond-all-reason/bar-db) serves the replay database at https://api.bar-rts.com: GET /replays searches
//     replays (src/model/rest-api/replays.ts, src/rest-api/routes/replays.ts) and GET /replays/<id> returns one replay
//     (src/rest-api/routes/replay.ts). Its database model is in src/model/db/. The server rate limits per IP address.
//   - An OpenStack Swift container on OVH stores the replay files under demos/<fileName>. The site links them as
//     objectStorageUrl + "/demos/" + fileName. Files older than about a year may have been deleted.
//
// On top of one method per endpoint, Client combines them for finding and fetching the games of top players:
// LeaderboardSeason (the active season), Leaderboard (one game type), RecentReplays (a player's recent games of one
// game type, see GameType.Matches) and DownloadDemoIntoDir (a demo file stored safely under its name).
//
// Client sends at most Config.MaxRequestRate requests per second to Teiserver and bar-db together, so a tool walking
// many pages or replays does not overload them. Downloads from the storage are not rate limited. Client sets no
// timeout on its requests: pass a context with a deadline to bound them.
package barlive
