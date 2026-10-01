# Changelog

What changed in each Relay DB release, for the people using it. The section for
a version is also its GitHub release notes and what the in-app updater shows
under *What's new*, so write for someone deciding whether to restart now.
Versions follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **The SQL editor points out mistakes as you type**: an unclosed quote, bracket or comment, a comma with nothing after it, a typo in the first word (`selec` → *did you mean SELECT?*). It only marks what is wrong in every database, so correct SQL is never underlined.
- **Run selection**: with text selected, the Run button says so and runs just that.

### Changed

- **Drop-down lists are Relay DB's own** — the schema switcher, filters, the connection form, settings and enum cells — with search on long lists and full keyboard control.
- **Fast scrolling through wide tables keeps up**: the grid draws only the columns in view and reuses rows instead of rebuilding them, so a frame takes a quarter of the time it did.
- **Nothing bounces past its edges** on a trackpad any more.

### Fixed

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
