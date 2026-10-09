# Contributing to Plugboard

Thanks for wanting to help. Bug reports, docs fixes and pull requests are all welcome.

By taking part you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Before you start

- **Found a security problem?** Don't open a public issue. Follow [SECURITY.md](SECURITY.md).
- **Fixing a bug?** Go ahead and open a pull request.
- **Adding a feature or changing behaviour?** Open an issue first. Plugboard is deliberately
  local-first and careful with production data, and it's much cheaper to agree on the shape
  of a change before it's written than to turn down a finished pull request.

## Development setup

Requirements:

- Go 1.26+
- Node.js 22.12+
- [Wails v2](https://wails.io/docs/gettingstarted/installation) — `go install github.com/wailsapp/wails/v2/cmd/wails@v2.16.0`
- macOS: Xcode Command Line Tools
- Linux: `libgtk-3-dev`, `libwebkit2gtk-4.0-dev`
- Windows: NSIS, for installer builds
- Docker, for the sample databases

```bash
git clone https://github.com/relay-client/plugboard
cd plugboard
npm install
make dev
```

`make dev` runs the Go backend and the Svelte frontend with hot reload, and keeps passwords
in `secrets.dev.json` in the profile folder rather than the keychain, so rebuilds don't set off
keychain prompts. The running app is also reachable in a browser at http://localhost:34115.
`make help` lists every target.

### Databases to develop against

```bash
make db-up      # PostgreSQL :55432, MySQL :53306, Redis :56379 and an SSH bastion :52222 — all relay/relay, database "shop"
make db-down    # stop them and drop their data
make sample-db  # dev/sample.db, a small SQLite file
```

From the bastion, the databases are `postgres:5432` and `mysql:3306` — handy for trying SSH
tunnels. The ports listen on 127.0.0.1 only.

## What CI checks

Run this before pushing — it covers most of it:

```bash
make check
```

The full pipeline in [`ci.yml`](.github/workflows/ci.yml):

| Check | Command |
| --- | --- |
| Frontend types | `npm run frontend:check` |
| Frontend unit tests | `npm run frontend:test` |
| Frontend build | `npm run frontend:build` |
| Go formatting | `gofmt -l apps/desktop` must print nothing |
| Go vet | `go vet ./apps/desktop/...` |
| Go tests | `go test -race ./apps/desktop/...` on Linux and macOS, without `-race` on Windows |
| Tests against real servers | the `make db-up` servers, with the variables below |
| Dead code | `golang.org/x/tools/cmd/deadcode` must report nothing |
| End-to-end | `npm run frontend:e2e` (Playwright, WebKit) |

The tests that need real servers skip themselves unless you point them at some:

```bash
make db-up
cd apps/desktop
PLUGBOARD_TEST_PG=127.0.0.1:55432 PLUGBOARD_TEST_MYSQL=127.0.0.1:53306 PLUGBOARD_TEST_SSH=127.0.0.1:52222 go test ./internal/db/
```

Run them when you touch anything that talks to a database: read-only enforcement, grid edits,
filters, paging or SSH. Two checks surprise people:

- **Dead code.** Unreachable Go code fails the build. Use it or delete it.
- **End-to-end.** The walkthrough drives the UI against a stand-in backend
  ([`frontend/e2e/bridge.ts`](apps/desktop/frontend/e2e/bridge.ts)). A new method on the Go
  `App` needs a fake there too.

## How the code is laid out

```
apps/desktop/
  main.go                  Wails entry point, window and menus
  internal/api/            App — every exported method is callable from the frontend
  internal/db/             Session, the Dialect per engine, read-only screening, the SQL splitter
  internal/sshtunnel/      SSH tunnels and host-key checks
  internal/store/          connections.json, settings and the keychain-backed secrets
  internal/update/         finding, verifying and installing new releases
  internal/model/          types shared with the frontend
  frontend/src/
    lib/wire.ts            TypeScript mirrors of the Go types
    lib/backend.ts         typed calls into the Go App
    lib/stores/            app state (Svelte 5 runes)
    lib/components/        the UI
    styles/tokens.css      colours, dark and light
  frontend/e2e/            Playwright walkthrough and the stand-in backend
dev/                       docker-compose and seed data for the sample databases
```

## Code style

Match the surrounding code. Beyond that:

- **Go** — `gofmt`. Comments explain *why*, not *what*; a comment that restates the line below
  it will be asked about in review.
- **Engines** — anything that differs between PostgreSQL, MySQL and SQLite goes behind
  `db.Dialect`; shared logic uses `database/sql`. Engines that aren't SQL (Redis) implement
  `db.Session` themselves, in their own package under `internal/db/engines`.
- **Svelte 5** — runes (`$state`, `$derived`, `$effect`), not the legacy store API.
- **TypeScript** — no `any` in new code. Go types sent to the frontend are mirrored by hand in
  `lib/wire.ts`: change the json tags and the TypeScript fields together.
- **Colours** — only through the tokens in `styles/tokens.css`, so both themes keep working.
- **Tests** — a behaviour change needs a test. Anything touching read-only, edits or secrets
  needs one most of all: those are the paths where a regression costs someone their data.

### Changing what the screenshots show

The pictures in the README are taken by the end-to-end walkthrough, not by hand. After changing
the interface, retake them and commit the result:

```bash
make screenshots
```

They come from the stand-in backend, so they're the same every time and hold no real data.

## Commits and pull requests

- Write commit messages in the imperative mood with a short subject line:
  `Refuse set_config() on read-only connections`.
- Keep a pull request to one logical change. Unrelated cleanups go in their own.
- Fill in the pull request template and say how you checked the change.
- Add a line to [`CHANGELOG.md`](CHANGELOG.md) under `[Unreleased]` for anything a user would
  notice. It becomes the release notes and what the app shows when it updates.

## Reporting bugs

Use the issue templates. What helps most is the Plugboard version (Settings ▸ About), the
database and its version, and the smallest set of steps that shows the problem.

**Scrub credentials from anything you attach.** Connection URLs, screenshots of the connection
form, `connections.json` and query text often carry hosts, user names and passwords.
