# Changelog

What changed in each Plugboard release, for the people using it. Up to 0.3.0
the app was called Relay DB. The section for a version is also its GitHub
release notes and what the in-app updater shows under *What's new*, so write
for someone deciding whether to restart now.
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
