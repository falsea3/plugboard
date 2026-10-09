<div align="center">

# Plugboard

**A fast, local-first database client for PostgreSQL, MySQL, SQLite, ClickHouse, Cassandra and Redis. No accounts, no cloud sync, no telemetry — just you and your data.**

[![CI](https://github.com/relay-client/plugboard/actions/workflows/ci.yml/badge.svg)](https://github.com/relay-client/plugboard/actions/workflows/ci.yml)
[![Latest release](https://img.shields.io/github/v/release/relay-client/plugboard?sort=semver)](https://github.com/relay-client/plugboard/releases/latest)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20Windows%20%7C%20Linux-lightgrey)](https://github.com/relay-client/plugboard/releases/latest)

The database counterpart of [Relay](https://github.com/relay-client/relay), formerly called Relay DB. Built with Go, Svelte 5 and Wails.

</div>

![Plugboard showing a PostgreSQL table](.github/assets/screenshots/table.png)

---

## Download

Grab the latest build from the [releases page](https://github.com/relay-client/plugboard/releases/latest).

| Platform | File |
| --- | --- |
| macOS 12+ on Apple Silicon (M1 and later) | `-darwin-arm64.dmg` — drag Plugboard to Applications |
| macOS 12+ on Intel | `-darwin-amd64.dmg` |
| Windows 10/11 (x64 or Arm64) | `-installer.exe` |
| Linux (x64) | `.AppImage` — `chmod +x`, then run it |

Plugboard looks for a new version when it starts and offers to install it, or installs it by itself if you turn that on in Settings. Every release ships SHA-256 checksums and signatures, and the app refuses an update that fails either. What changed in each version is in the [changelog](CHANGELOG.md).

**The builds aren't signed by Apple or Microsoft yet**, so both systems warn on the first launch. Check the download against `SHA256SUMS.txt` on the release page, then:

- **macOS** — *"can't be opened because Apple cannot check it"*: open it once, then click **Open Anyway** in System Settings ▸ Privacy & Security. If it says the app *is damaged*, run `xattr -dr com.apple.quarantine "/Applications/Plugboard.app"`.
- **Windows** — SmartScreen's *"Windows protected your PC"*: click **More info** ▸ **Run anyway**.

You do this once; updates the app installs itself don't ask again.

---

## Features

**Connections**
- PostgreSQL, MySQL and MariaDB, SQLite files, ClickHouse, Cassandra, and Redis or Valkey — several open at once, each with its own tabs. `⌘K` jumps to any of them by name.
- Tag a connection *local*, *dev*, *staging* or *prod*: the tag colours the window, so you always know where you are.
- Paste a connection URL — `postgres://…`, `mysql://…`, `clickhouse://…`, `cassandra://…`, `redis://…`, a JDBC URL or a line from a `.env` file — and the form fills itself in.
- **SSH tunnels** through a bastion, or straight into the database server, with a password, a private key or ssh-agent. A dropped tunnel comes back on its own.
- Passwords stay in the system keychain (Keychain, Windows Credential Manager, Secret Service) — never in a file on disk.

**Hard to break production by accident**
- **Read-only connections**, on by default for anything tagged Production. Plugboard refuses writing statements before they run, and the database session itself is read-only too, so a statement that slips past one still meets the other.
- Writes to a Production connection ask first, and show exactly what they will run.
- Edits in the grid are applied in one transaction — every change lands, or none does — and a value too long for its column is an error, never quietly cut short.
- If a connection drops while a statement is running, Plugboard never re-runs a write behind your back, and tells you when an open transaction was lost.
- A changed SSH host key is refused until you've seen the new fingerprint and trusted it.

**Tables**
- Big tables page by primary key, so the last page of a few million rows opens as fast as the first. The row count shows the server's estimate at once and the exact number on request.
- Filter rows by any column (`⌘F`): equals, contains, starts with, in a list, is NULL and more — or right-click a cell and *Filter by this value*.
- Edit like a spreadsheet: type into a cell, `⌥⌫` for NULL, `⌘I` to add a row, `⌘⌫` to delete. Right-click offers what fits the column — true/false, `now()`, DEFAULT, the values of an enum.
- Changes wait until `⌘S`; *Preview SQL* shows the statements first.
- A Structure tab with types, nullability, defaults and keys.
- Follow a foreign key: the arrow in a key cell opens the row it points at.
- A schema diagram of tables, columns and foreign keys, laid out for you; click a relation to see what it joins.

![The schema diagram of the sample shop](.github/assets/screenshots/diagram.png)

**SQL editor**
- Highlighting for each database's dialect and completion of table names.
- `⌘↵` runs the statement under the cursor, `⇧⌘↵` the whole script, Stop cancels.
- Statements share one connection, so `BEGIN … COMMIT`, `SET` and temporary tables behave the way you expect.
- Results arrive a thousand rows at a time instead of the whole table at once.

![The SQL editor with a query and its result](.github/assets/screenshots/query.png)

**ClickHouse**
- Databases, tables and views from `system.*`, with each table's sorting key and data-skipping indexes, dictionaries and SQL functions.
- Tables open read-only — ClickHouse has no transactions and its updates are background mutations, so changes go through the SQL editor, where you see exactly what runs.
- The editor checks syntax with `EXPLAIN AST` as you type, which parses without running anything.
- Read-only connections run with the server's `readonly` setting as well as Plugboard's own check.

**Cassandra**
- Keyspaces, tables, materialized views, user types, functions and secondary indexes, with partition and clustering keys shown as the primary key.
- Tables page through the server's own paging, so a large table scrolls without `OFFSET`. Filter with `=`, `<`, `>`, `IN`, and `CONTAINS` on collections.
- Edit rows in the grid: each change is a lightweight transaction (`IF EXISTS` / `IF NOT EXISTS`), so it lands on exactly the row you saw, and values go through `fromJson`, collections and user types included.
- A CQL editor that keeps `BEGIN BATCH … APPLY BATCH` together and follows `USE`.
- Works through SSH tunnels to a whole cluster: every node is reached through the tunnel.

**Redis**
- Keys as a tree, grouped by `:`, from every database 0–15 (or however many the server has), scanned page by page so a huge keyspace doesn't stall the server. Filter by text or a pattern like `user:*`.
- Strings, hashes, lists, sets, sorted sets and streams each open in a view that fits them: edit a JSON string with formatting, double-click a field or a member to change it, add or remove items, set a TTL, rename or delete the key.
- Changes that depend on what's there — renaming a field, removing a list item — run atomically and refuse if someone else changed the value first.
- A command console with completion, one command per line on its own connection, so `SELECT`, `MULTI … EXEC` and `WATCH` work.
- Read-only connections refuse any command the server marks as writing, and the Production confirmation asks before one runs.

**AI agents (MCP)**
- Plugboard is also an MCP server: `claude mcp add plugboard -- /Applications/Plugboard.app/Contents/MacOS/plugboard mcp` (Settings ▸ AI agents shows the exact command), and Claude Code, Cursor or any other agent can list your tables, read their structure and run queries.
- Agents never see a password: they ask Plugboard, which connects with your saved connections, Keychain and SSH tunnels.
- Nothing is shared until you turn on *AI agents (MCP)* for a connection. Sessions are read-only unless you start it with `--write`, and Production is read-only for agents no matter what. `--env staging,dev` narrows it further.
- Every call is logged to `logs/mcp.log` in the profile folder.

**Everything else**
- Keyboard-first (`Ctrl` instead of `⌘` on Windows and Linux): tabs with `⌘1`–`⌘9`, `⌘T` for a new query, `⌘W` to close, `⌘C` / `⇧⌘C` to copy a cell or a row, `⌘A` for the whole result as TSV.
- Dark and light themes, or follow the system.

**Local-first.** Your connections live in a plain JSON file in your profile folder and your passwords in the system keychain. Plugboard talks to your databases, your SSH servers and, at launch, to GitHub to look for updates — nothing else.

| | Profile folder |
| --- | --- |
| macOS | `~/Library/Application Support/Plugboard` |
| Windows | `%AppData%\Plugboard` |
| Linux | `~/.config/Plugboard` |

---

## Building from source

You need Go 1.26+, Node.js 22.12+ and the [Wails v2](https://wails.io/docs/gettingstarted/installation) CLI.

```bash
git clone https://github.com/relay-client/plugboard
cd plugboard
npm install
make dev
```

`make build` builds the app for your platform, `make check` runs the tests. [CONTRIBUTING.md](CONTRIBUTING.md) has the rest: sample databases to develop against, what CI checks, and how the code is laid out.

---

## Contributing

Bug reports, ideas and pull requests are welcome — start with [CONTRIBUTING.md](CONTRIBUTING.md). Everyone taking part is expected to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

Found a security problem — a way past read-only, a leaked password, an SSH key accepted when it shouldn't be? Please **don't** open a public issue; see [SECURITY.md](SECURITY.md).

---

## License

[MIT](LICENSE). Bundled icons, logos and fonts keep their own licences — see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
