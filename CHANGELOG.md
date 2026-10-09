# Changelog

What changed in each Plugboard release, for the people using it. Up to 0.3.0
the app was called Relay DB. The section for a version is also its GitHub
release notes and what the in-app updater shows under *What's new*, so write
for someone deciding whether to restart now.
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.2] - 2026-10-10

### Fixed

- **Sidebar menus speak each database's language.** *Delete all rows…* is `TRUNCATE` on ClickHouse and Cassandra (a bare `DELETE` fails there), and dropping a Cassandra column writes valid CQL. Cassandra's sidebar can create and drop indexes and rename key columns, and *Rename table…* is gone where the database can't rename tables.
- **Redis key menus have icons** like every other menu.

## [0.7.1] - 2026-10-09

### Added

- **Plugboard as an MCP server for AI agents.** Run `plugboard mcp` from Claude Code, Cursor or any other agent (Settings ▸ AI agents has the command to copy) and it can list tables, read their structure and run queries on the connections you open to it — without ever seeing a password. Only connections with *AI agents (MCP)* turned on are visible; agents are read-only unless started with `--write`, and never write to Production. Every call is logged.

## [0.7.0] - 2026-10-09

Three new kinds of database: Redis, ClickHouse and Cassandra.

### Added

- **Redis and Valkey.** Connect over TLS or an SSH tunnel and browse keys as a tree grouped by `:`, in any database. Strings, hashes, lists, sets, sorted sets and streams open in their own views: edit values and JSON, add and remove items, set TTLs, rename and delete keys. A command console runs one command per line. Read-only connections refuse writing commands, and Production asks before running one.
- **ClickHouse.** Browse databases, tables, views, dictionaries and SQL functions, with each table's sorting key and skip indexes. Tables open read-only — change data in the SQL editor, which checks syntax as you type without running anything. Read-only connections also use ClickHouse's own `readonly` setting.
- **Apache Cassandra.** Browse keyspaces, tables, materialized views, user types, functions and indexes, and page through large tables with the server's own paging. Edit rows in the grid — every change is checked against the row you saw — and run CQL, batches included. Works over SSH tunnels, including clusters behind a bastion.

### Changed

- **Connection initials skip punctuation.** A connection named “Redis (sample)” shows RS, not R(.

## [0.6.5] - 2026-10-09

### Added

- **A Value panel for table cells.** In a table tab, select a cell to see its full value beside the table. Resize the panel or hide it with the Value button.
- **Reorder tabs by dragging.** Drag a tab, or a connection in the title bar, to a new place. The new order of query tabs is kept with the connection.

### Fixed

- **Grid keyboard focus after closing the value editor.** Closing it with Esc returns focus to the selected cell, so keyboard navigation works immediately.
- **Update check errors stay readable.** Long error messages wrap instead of being cut off.

## [0.6.4] - 2026-10-06

### Fixed

- **Errors say what went wrong again.** A failed connection, and every other error, showed only “[object Object]”. You now see the message, for example “Nothing answers at 127.0.0.1:55432”, with the server's own text under it.

## [0.6.3] - 2026-10-06

### Added

- **Saved scripts.** Save a query as a script with ⌘S. Scripts belong to their connection and are listed under *Scripts* in the sidebar, where you can open, rename or delete them.
- **Closing an edited query asks first.** You can *Save* (or *Save as…* for a new query), choose *Don’t save*, or cancel. A dot on the tab marks unsaved changes.
- **Query tabs survive a restart.** Open SQL editor tabs and their text are kept per connection. They come back the next time you connect, including after quitting the app.

## [0.6.2] - 2026-10-05

### Changed

- **Double-click a table in the sidebar to expand or collapse its columns and indexes.**

### Fixed

- **Restart now after an update starts the app again on macOS.** It only quit: macOS ends everything an app opened from the Dock or Finder started when the app quits, and that included the helper meant to start it again. The new version is now launched first and waits for the old one to close. This works from the next update on — the version doing the restart has to have the fix.

## [0.6.1] - 2026-10-05

### Added

- **Create and drop indexes from the sidebar.** An unfolded table lists its indexes under its columns. Right-click a table or a column → *Create index…*: pick the columns in order, make it unique, and review the statement in a query tab. Right-click an index → *Drop index…* to open its `DROP INDEX` statement there too.
- **An editor for long values.** ⇧↵, *Open in editor* in a cell's menu or the button next to the value at the bottom opens the value in its own window — wrapped lines, a character and line count, ⌘S to save it to the cell. In query results it opens read-only.
- **Type suggestions from your database.** *Change type…* suggests the types your database has, its own enums and domains, and the types the table already uses.

### Fixed

- **Switches react to every click.** Only a small spot on the left of a switch took clicks; the whole switch does now. Clicking faster than the setting saved could also flip it back.
- **⌘S in the JSON or value editor no longer commits the table.** It saved the value and then committed every pending change of the table to the database straight away.
- **Changing a filter's operator or column applies it at once** when the condition already has a value — no need to press Apply.
- **Fit width fits.** It sized a column to the longest of all its loaded values instead of always making it wider.

## [0.6.0] - 2026-10-05

### Added

- **Functions, procedures, sequences, types, triggers, events and extensions in the sidebar**, each kind in its own group, with what matters at a glance — a function's arguments, a trigger's table, an enum's values (unfold it to see them), an extension's version. Click one to see its DDL.
- **DDL for everything.** Tables get a DDL tab next to Data and Structure — the full `CREATE TABLE` with keys, foreign keys, defaults, indexes, triggers and comments on PostgreSQL, the server's own `SHOW CREATE` on MySQL and MariaDB, the stored SQL on SQLite. Copy it or open it in a query tab.
- **A table tree in the sidebar.** Expand a table or view to see its columns, with the primary key marked and each column's type. Right-click a table to open its data or structure, start a `SELECT *` query, show it in the diagram, copy its name or a `SELECT`, or rename, empty or drop it; right-click a column to copy its name, query it, rename it, change its type or drop it. Renaming and changing a type ask for the new name or type and write the statement the way your database needs it — MySQL's `CHANGE COLUMN` keeps the column's default, `NOT NULL` and collation, and a renamed MySQL table stays in its own database. Changes open as SQL in a new query tab, so you see the statement and run it yourself — read-only sessions and the Production check still apply. ←/→ fold and unfold the selected table.

### Fixed

- **Triggers, procedures and functions with a `BEGIN … END` body run from the SQL editor.** The editor used to cut them at the first `;` inside the body. Scripts with `DELIMITER` lines, as MySQL dumps have them, run too.
- **Dialogs no longer close when you select text inside them and let go outside.** A dialog closes on a click outside only when the click both starts and ends outside it.

## [0.5.2] - 2026-10-04

### Changed

- **Column types in the table header.** Each column shows its type next to its name in small grey letters, and the sort arrow now sits before the name. Columns open wide enough for both; when you narrow one, the type gives way before the name does.
- **A calmer title bar and home screen**: the open connection's tab is a filled pill without a frame, and the home screen's left panel no longer has a divider line.

## [0.5.1] - 2026-10-04

### Added

- **A JSON editor.** Double-click a `json`/`jsonb` cell — or right-click any text cell holding JSON → *Edit as JSON* — for a formatted editor with highlighting, folding and live validation; Save stays off until the JSON is valid. Formatting never touches numbers or strings, so a big integer comes back exactly as it went in, and the cell keeps its compact or multi-line style.
- **Indexes in the Structure tab**: name, columns, primary/unique, method and the condition of a partial index; hover for the full definition.
- **Pick several rows at once**: ⇧-click (or ⇧↑/↓) for a range, ⌘-click to add or drop a row. Copying, deleting and *Copy/Open N rows as SQL* use all of them, so one INSERT carries every row you picked.

### Fixed

- **The diagram background no longer shimmers when zoomed out**: the dots stay the same size and thin out instead of crowding together.
- **The JSON editor points at the mistake itself**: the exact spot is underlined and the status line says where and what — "Line 3, column 3: Expected “,” or “}”" — instead of a mark on line 1.
- **A submenu that opens to the left stays open** while the mouse crosses over to it. Near the right edge of the window *Copy as SQL* and *Open SQL in new query* open their lists on the left, and they used to vanish half the time on the way there.
- **The loading bar shows while more rows load**, in a table as you scroll and in query results on *Load 1,000 more*, not only on the first load.

## [0.5.0] - 2026-10-03

### Added

- **A crash log.** If Plugboard ever crashes, the full report is saved to `logs/crash.log` in its data folder, and the next launch says so, with a button to show it. Settings › About has the logs folder too.

- **Jump along a foreign key.** A cell that points at another table gets an arrow (on hover, and always on the selected cell): click it, or right-click → *Go to customers*, and the referenced table opens filtered to exactly that row. If the table is already open, its filter switches to the row, unless it has unsaved edits.

- **Schema diagram.** The diagram button at the bottom of the sidebar draws every table of the schema with its columns and the foreign keys between them, laid out from the tables others point at to the ones pointing at them. Click a relation to see exactly which columns it joins and what happens on delete and update; click a table to light up its relations; double-click to open it. Drag tables around, pan with the trackpad, zoom with pinch or ⌘ and the wheel. Works on PostgreSQL, MySQL, MariaDB and SQLite.

### Changed

- **Errors in plain words.** Connection and network problems say what happened and what to check — "Nothing answers at db:5432. Is the server running, and is the port right?", "Can't find the host…", "The server's certificate isn't signed by an authority this computer trusts", "The SSH server refused the sign-in" — with the original error underneath.
- **An internal error now ends the app instead of leaving it half-working.** Before, a crash inside one action could leave a spinner turning forever; now Plugboard stops, so nothing is written in an unknown state, and the crash log says where it happened.
- **Menus have icons, one per kind of action**: every filter shows the same funnel, every change to a value the same pencil, every copy the same copy icon; sorting keeps its up and down arrows.

### Fixed

- **The time in the table footer now belongs to the last load.** It used to keep showing how long the first rows took while more were loaded below.

## [0.4.0] - 2026-10-02

### Changed

- **Tables load as you scroll.** The page buttons are gone: when you near the bottom, the next rows load by themselves, up to 100,000 rows. Edited cells stay edited while more rows load, so there's no need to commit first; while a new row waits to be saved, loading pauses until you commit or discard it.
- **Fast scrolling no longer shows empty rows.** The grid scrolls itself on the wheel and trackpad and draws the rows in the same frame, so rows and their numbers stay in step however hard you flick. PageUp/PageDown move by a page, Home/End go to the first and last loaded row.
- **Relay DB is now Plugboard.** Same app, new name: a database client called "something DB" read like a database. On first launch Plugboard moves your connections, settings and trusted SSH host keys over from Relay DB, and your saved passwords from the Keychain, so there is nothing to set up again.
- Relay DB 0.3.0 doesn't update itself to Plugboard. Download Plugboard from GitHub, then delete Relay DB.
- Connections show up as *Plugboard* in the database's session list (PostgreSQL's `application_name`).

## [0.3.0] - 2026-10-02

### Added

- **Turn rows into SQL.** Right-click a row, or a selection of rows, and pick *Copy as SQL* or *Open SQL in new query*: SELECT, INSERT, UPDATE or DELETE, matched on the primary key. UPDATE is written one statement per row, with every column set to its current value, ready to edit.
- **Open several tables at once.** ⌘-click or ⇧-click tables in the sidebar, then *Open* (or Enter).
- **Each connection in the title bar shows how many tabs it has open.**
- **Relay DB says when it's waiting**: a progress line while a table page loads, a spinner while a table opens, the tables load, a connection opens, a query runs — with how long it has been running — or rows are counted, saved or tested.
- **A proper Mac installer window**: drag Relay DB onto Applications, with the app's own icon on the disk.

### Changed

- **Many open connections no longer run off the title bar.** They shrink like browser tabs, the current one stays readable and in view, and the strip scrolls with the wheel or trackpad.
- *Copy as INSERT* moved into *Copy as SQL*. Binary columns are left out of generated SQL, since the grid shows them shortened.

## [0.2.1] - 2026-10-02

### Added

- **The database checks your SQL as you type.** When Relay DB's own checks find nothing, it asks the server to parse each statement — without running it, and not on the editor's connection, so an open transaction is safe — and underlines exactly where the server says the syntax is wrong, on PostgreSQL, MySQL, MariaDB and SQLite. A statement that fails when you run it is marked the same way.

### Changed

- **Mistakes are easier to see**: a thicker wavy underline, and a red dot beside the line.
- **About says when Relay DB is up to date** instead of offering to check again.

### Fixed

- **The statement that ran stayed marked after the next run.** Running a selection or everything left the earlier mark in place, in the selection's colour, so it looked like the old text was still selected. Now whatever ran last gets a bar in the margin, gone at the next edit or click.

## [0.2.0] - 2026-10-02

### Added

- **The SQL editor points out mistakes as you type**: an unclosed quote, bracket or comment, a comma with nothing after it, a typo in the first word (`selec` → *did you mean SELECT?*). It only marks what is wrong in every database, so correct SQL is never underlined.
- **Run selection**: with text selected, the Run button says so and runs just that.
- **Change a table's columns in the Structure tab** — add, rename and drop columns, change their type, NULL and default — on PostgreSQL, MySQL and MariaDB. It works like editing rows: changes collect until ⌘S, *Preview SQL* shows the `ALTER TABLE` statements, Production asks first. PostgreSQL applies them all or none; MySQL commits each one, so when one fails Relay DB says which went through. A long change can be stopped, and one that would have to wait for a table another transaction holds gives up after 5 seconds instead of holding up every query on it. SQLite can rename, add and drop columns; for anything more its tables need recreating in the SQL editor.

### Changed

- **Drop-down lists are Relay DB's own** — the schema switcher, filters, the connection form, settings and enum cells — with search on long lists and full keyboard control.
- **Fast scrolling through wide tables keeps up**: the grid draws only the columns in view and reuses rows instead of rebuilding them, so a frame takes a quarter of the time it did.
- **Nothing bounces past its edges** on a trackpad any more.
- **macOS has a download for Apple Silicon and one for Intel** instead of a universal app of both, so each is half the size. Updates bring each Mac the build for its processor; an Intel build running under Rosetta on Apple Silicon moves over to the native one. 0.1.0 can't find its file under the new names, so it has to be installed again by hand once.

### Fixed

- **A table read failed after its columns changed** (*cached plan must not change result type*) on PostgreSQL, for example after an `ALTER TABLE` in the SQL editor. The read now simply runs again.
- **NULL in a new row's cell turned into DEFAULT**, so the insert took the column's default instead of NULL.
- **MySQL's `--` without a space was taken for a comment.** MySQL starts a comment only at `-- ` (there `1--1` is `1 - -1`), so `select 1--1; select 2` ran as one statement, and the read-only check didn't look past the dashes.
- **A `$` inside a PostgreSQL name** (`a$b$`) was read as the start of a `$$` body, swallowing the rest of the script into one statement.
- **A SQLite `FLOAT` column lost digits**: it was shown at single precision like a PostgreSQL `real`, though SQLite stores it as a double.
- **A PostgreSQL `real` showed digits it doesn't have** — `5.2` came out as `5.199999809265137`, because the driver widens it to a double. It shows as stored now.

## [0.1.0] - 2026-10-01

The first public release.

### Added

- **PostgreSQL, MySQL/MariaDB and SQLite**, several connections open at once, each with its own tabs. Profiles carry an environment tag (local, dev, staging, prod) shown as a colour stripe; passwords live in the system keychain.
- **SSH tunnels** with a password, a private key or ssh-agent. A server is trusted on first connect, a changed host key is refused, and trusting a new one records exactly the key you were shown.
- **Read-only connections**, on by default for Production, enforced by Relay DB before anything runs and by the server session itself. Turning it off on a Production connection asks first.
- **Tables**: paged by primary key, so the last page of a big table is as quick as the first; filters, sorting, type-aware context menus, and the row count as an estimate or exact.
- **Editing in the grid**: changes collect until ⌘S and apply in one transaction — all or nothing — with a preview of the SQL. A value too long for its column is an error, never cut to fit.
- **SQL editor** with per-dialect highlighting, completion of table names, and results 1,000 rows at a time. Writes on a Production connection ask first. When a cancel or a dropped connection takes an open transaction with it, Relay DB says so.
- **Updates from GitHub**: Relay DB finds a new release at launch and installs it after checking its checksum and signature.
