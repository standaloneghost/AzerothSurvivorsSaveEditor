# Azeroth Survivors Save Editor

A local save editor for Azeroth Survivors. It is one Windows `.exe` with nothing to install. A small program reads and writes the save, and its interface opens in your browser.

## What it edits

- Marks, levels, zone clears, and Heroic tiers for each class
- Gear unlocks, including Rogue and Shaman off-hands
- Achievements, perks, companions, races, and tabards
- Lifetime stats, plus every raw save record under Advanced

It also has **Complete All**, **Delete all saves**, and a Backups page.

## Safety

- The save is backed up to `EditorBackups` before every write
- Saving is blocked while the game is running, or if the file changed since it was opened
- Files are written atomically, then read back and verified
- The program only listens on `127.0.0.1`, and API calls need a token generated at launch

## Building

Requires Go and Python 3. You also need your own copy of `AzerothSurvivors.exe`, because the save key comes from the game and is not stored in this repo.

```bash
python3 tools/extract_key.py path/to/AzerothSurvivors.exe
python3 tools/build_page.py
GOOS=windows GOARCH=amd64 go build -trimpath -buildvcs=false -ldflags "-H windowsgui" -o AzerothSurvivorsSaveEditor.exe .
```

Never commit `key.bin` or a built `.exe`. Both contain the key, and `.gitignore` excludes them.

## Testing

The tests use Playwright and a Linux build pointed at a **copy** of a save through `WOW_ROGUE_SAVE`. Never run them against a real save. `GAME_RUNNING_FLAG` names a file that stands in for the running game. Run them from the repo root:

```bash
go build -o editor-linux .
mkdir -p tests/shots /tmp/save && cp path/to/characters.txt /tmp/save/
WOW_ROGUE_SAVE=/tmp/save/characters.txt GAME_RUNNING_FLAG=/tmp/save/game.flag ./editor-linux > url.txt &
NODE_PATH=$(npm root -g) node tests/t13.js "$(cat url.txt)" /tmp/save/characters.txt
```

## Save format

See `tools/savelib.py` for the file format. Inside, it is plain text with one `kind|fields` record per line.

| Record | Holds |
|---|---|
| `items` | Owned gear IDs |
| `equip`, `look` | Worn gear and appearance per class |
| `stars` | Gear star levels; +0 pieces aren't listed |
| `marks` | Marks per class, stored in **tenths** |
| `talent`, `spec` | Experience, active talents, and three saved specs |
| `beat`, `hbeat` | Zone clears and highest Heroic tier per class |
| `ach`, `achat` | Achievements and when they were earned |
| `perk`, `pets`, `races`, `tabard` | Account-wide rewards; `pets` is in slot order |

Gear IDs are `1 CC S R 0`: class, slot (0 Head to 7 Hands, 8 Main Hand, 9 Off Hand), and rarity (0 Common to 4 Legendary, 5 PvP).

Level N costs `80 + 15 * (N - 1)` experience. The game's code stops awarding talent points at level 51, or 23,205 experience.

## Known limits

- The save key and format can change with a game update. The editor then refuses to open the save rather than risk damaging it, and `tools/extract_key.py` will need new addresses.
- Editing talent trees directly was tried, and the game wiped the edits, so the editor doesn't include it.
- Gear IDs for classes with no gear in the test save follow the Paladin and Mage pattern, and weren't individually confirmed.
