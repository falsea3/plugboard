# Changelog

What changed in each Relay DB release, for the people using it. The section for
a version is also its GitHub release notes and what the in-app updater shows
under *What's new*, so write for someone deciding whether to restart now.
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **The SQL editor points out mistakes as you type**: an unclosed quote, bracket or comment, a comma with nothing after it, a typo in the first word (`selec` → *did you mean SELECT?*). It only marks what is wrong in every database, so correct SQL is never underlined.
- **Run selection**: with text selected, the Run button says so and runs just that.
- **Change a table's columns in the Structure tab** — add, rename and drop columns, change their type, NULL and default — on PostgreSQL and MySQL. It works like editing rows: changes collect until ⌘S, *Preview SQL* shows the `ALTER TABLE` statements, Production asks first. PostgreSQL applies them all or none; MySQL commits each one, so when one fails Relay DB says which went through. A long change can be stopped, and one that would have to wait for a table another transaction holds gives up after 5 seconds instead of holding up every query on it. SQLite can rename, add and drop columns; for anything more its tables need recreating in the SQL editor.

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
