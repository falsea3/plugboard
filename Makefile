SHELL := /bin/sh

.PHONY: help dev frontend install check test build open clean db-up db-up-all db-down sample-db test-servers \
        screenshots release release-minor release-major update-keygen

help:
	@$(MAKE) -s -C apps/desktop help
	@printf '  %-14s %s\n' 'make db-up' 'Start sample PostgreSQL + MySQL in Docker'
	@printf '  %-14s %s\n' 'make db-up-all' 'Also PostgreSQL 14, MySQL 8.0 and MariaDB 10.4/10.6/11.4'
	@printf '  %-14s %s\n' 'make test-servers' 'Run the live-server tests against every one of them'
	@printf '  %-14s %s\n' 'make db-down' 'Stop them and drop their data'
	@printf '  %-14s %s\n' 'make sample-db' 'Rebuild dev/sample.db (SQLite)'
	@printf '  %-14s %s\n' 'make screenshots' 'Retake the README screenshots'
	@printf '  %-14s %s\n' 'make release' 'Tag the next patch release (release-minor, release-major, v=x.y.z)'

dev frontend install check test build open clean:
	@$(MAKE) -C apps/desktop $@

db-up:
	docker compose -f dev/docker-compose.yml up -d --wait

db-up-all:
	docker compose -f dev/docker-compose.yml --profile all up -d --wait

# One run per server (the ports dev/docker-compose.yml gives them), so a
# failure says which release it is.
PG_SERVERS    := 55432 55433
MYSQL_SERVERS := 53306 53307 53310 53311 53312
test-servers:
	@cd apps/desktop && for port in $(PG_SERVERS); do \
		echo "== PostgreSQL on :$$port"; RELAYDB_TEST_PG=127.0.0.1:$$port go test -count=1 ./internal/db/... || exit 1; done
	@cd apps/desktop && for port in $(MYSQL_SERVERS); do \
		echo "== MySQL/MariaDB on :$$port"; RELAYDB_TEST_MYSQL=127.0.0.1:$$port go test -count=1 ./internal/db/... || exit 1; done
	@cd apps/desktop && echo "== SSH tunnels" && RELAYDB_TEST_PG=127.0.0.1:55432 RELAYDB_TEST_MYSQL=127.0.0.1:53306 RELAYDB_TEST_SSH=127.0.0.1:52222 go test -count=1 ./internal/db/...

db-down:
	docker compose -f dev/docker-compose.yml --profile all down -v

sample-db:
	rm -f dev/sample.db && sqlite3 dev/sample.db < dev/seed/sqlite.sql

# The walkthrough in apps/desktop/frontend/e2e drives the UI against a
# stand-in backend, so the pictures are the same every time and hold no real data.
screenshots:
	cd apps/desktop/frontend && RELAYDB_SCREENSHOTS="$(CURDIR)/.github/assets/screenshots" npx playwright test

release:
	scripts/release.sh $(or $(v),patch)

release-minor:
	scripts/release.sh minor

release-major:
	scripts/release.sh major

# Once, on a trusted machine. See docs/RELEASING.md for where the halves go.
update-keygen:
	@command -v minisign >/dev/null || { echo "Install minisign first: brew install minisign"; exit 1; }
	@[ ! -e update-signing-key ] || { echo "update-signing-key already exists; refusing to overwrite it."; exit 1; }
	minisign -G -W -p update-signing-key.pub -s update-signing-key
