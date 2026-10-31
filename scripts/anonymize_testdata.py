"""Pseudonymize dumped BAR Live Services responses before they are committed as test data.

Reads every *.json file of SOURCE_DIR and writes it to DEST_DIR. DEST_DIR must not exist yet, so the original dumps stay
untouched. Replaces happen the same way in every file, so references between the files still match:

- player and spectator names: Player001, Player002, ... in order of first appearance
- Teiserver account IDs (leaderboard "id", replay "userId"): 10000001, 10000002, ...
- country codes: XX; clan IDs: Clan1, Clan2, ...
- replay IDs (also "demoId" and the replay_<id>.json file names): 00000000000000000000000000000001, ...
- Teiserver match IDs (hostSettings "server_match_id"): 1, 2, ...

Everything else, such as maps, dates, durations, skills and settings, is kept. The files are written in the compact
format the services send.

Dump new responses into a directory outside the repository, run this script on it and copy the output into
internal/barlive/testdata, so the real names never reach the repository.

Usage: python3 -I scripts/anonymize_testdata.py SOURCE_DIR DEST_DIR
"""

import json
import pathlib
import re
import sys

REPLAY_FILE_PATTERN = re.compile(r"^replay_([0-9a-f]{32})\.json$")


class Pseudonyms:
    """Hands out a replacement per original value, the same one each time the value appears."""

    def __init__(self, make):
        self.make = make
        self.replacements = {}

    def __call__(self, value):
        if value is None:
            return None
        if value not in self.replacements:
            self.replacements[value] = self.make(len(self.replacements) + 1)
        return self.replacements[value]


names = Pseudonyms(lambda n: f"Player{n:03d}")
# Far above the real account IDs (627084 in 2026-10), so a replacement is not some other player's real ID.
account_ids = Pseudonyms(lambda n: 10_000_000 + n)
clan_ids = Pseudonyms(lambda n: f"Clan{n}")
replay_ids = Pseudonyms(lambda n: f"{n:032x}")
match_ids = Pseudonyms(lambda n: str(n))


def anonymize_person(person):
    """Anonymizes a player or spectator of a replay, or a player of a replay summary."""
    if "name" in person:
        person["name"] = names(person["name"])
    if "userId" in person:
        person["userId"] = account_ids(person["userId"])
    if "countryCode" in person:
        person["countryCode"] = "XX"
    if "clanId" in person:
        person["clanId"] = clan_ids(person["clanId"])


def anonymize_replay(replay):
    """Anonymizes a replay or a replay summary."""
    replay["id"] = replay_ids(replay["id"])
    if "server_match_id" in replay.get("hostSettings", {}):
        replay["hostSettings"]["server_match_id"] = match_ids(replay["hostSettings"]["server_match_id"])
    for ally_team in replay.get("AllyTeams", []):
        if "demoId" in ally_team:
            ally_team["demoId"] = replay_ids(ally_team["demoId"])
        for player in ally_team.get("Players", []):
            anonymize_person(player)
    for spectator in replay.get("Spectators", []):
        anonymize_person(spectator)
        if "demoId" in spectator:
            spectator["demoId"] = replay_ids(spectator["demoId"])


def anonymize(data):
    """Anonymizes one response, recognized by its shape. Error responses and season lists hold nothing to replace."""
    if isinstance(data, list):
        for leaderboard in data:
            for player in leaderboard.get("players", []):
                player["id"] = account_ids(player["id"])
                player["name"] = names(player["name"])
    elif isinstance(data, dict) and "data" in data:
        for replay in data["data"]:
            anonymize_replay(replay)
    elif isinstance(data, dict) and "AllyTeams" in data:
        anonymize_replay(data)


def main():
    if len(sys.argv) != 3:
        sys.exit(__doc__.strip().splitlines()[-1])
    source_dir, dest_dir = pathlib.Path(sys.argv[1]), pathlib.Path(sys.argv[2])
    dest_dir.mkdir(parents=True)

    for source in sorted(source_dir.glob("*.json")):
        data = json.loads(source.read_text(encoding="utf-8"))
        anonymize(data)
        dest_name = source.name
        if match := REPLAY_FILE_PATTERN.match(source.name):
            dest_name = f"replay_{replay_ids(match[1])}.json"
        text = json.dumps(data, separators=(",", ":"), ensure_ascii=False)
        (dest_dir / dest_name).write_text(text, encoding="utf-8")
        print(f"{source.name} -> {dest_name}")

    print(f"names:          {len(names.replacements)}")
    print(f"account IDs:    {len(account_ids.replacements)}")
    print(f"clan IDs:       {len(clan_ids.replacements)}")
    print(f"replay IDs:     {len(replay_ids.replacements)}")
    print(f"match IDs:      {len(match_ids.replacements)}")


if __name__ == "__main__":
    main()
