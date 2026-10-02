#!/bin/sh
# Tags a release and pushes the tag; .github/workflows/release.yml builds and
# publishes it from there.
#
#   scripts/release.sh patch|minor|major|<x.y.z>
#
# CHANGELOG.md must already have a "## [x.y.z] - <date>" section for the new
# version: it becomes the release notes and what the in-app updater shows.
set -eu

cd "$(git rev-parse --show-toplevel)"

if [ -n "$(git status --porcelain)" ]; then
  echo "error: the working tree has uncommitted changes:" >&2
  git status --short >&2
  exit 1
fi

last="$(git describe --tags --abbrev=0 --match 'v[0-9]*' 2>/dev/null || echo v0.0.0)"
IFS=. read -r major minor patch <<EOV
${last#v}
EOV
case "${1:-patch}" in
  patch) next="$major.$minor.$((patch + 1))" ;;
  minor) next="$major.$((minor + 1)).0" ;;
  major) next="$((major + 1)).0.0" ;;
  [0-9]*.[0-9]*.[0-9]*) next="$1" ;;
  *) echo "usage: release.sh patch|minor|major|<x.y.z>" >&2; exit 2 ;;
esac

notes="$(python3 scripts/changelog.py notes "$next")" || {
  echo "Move the [Unreleased] notes in CHANGELOG.md under '## [$next] - $(date +%Y-%m-%d)', commit, and run this again." >&2
  exit 1
}

printf 'Last release: %s\nNext:         v%s\n\n%s\n\nTag and push v%s? [y/N] ' "$last" "$next" "$notes" "$next"
read -r answer
[ "$answer" = y ] || [ "$answer" = Y ] || { echo "Aborted."; exit 1; }

git tag -a "v$next" -m "Plugboard $next"
git push origin "v$next"
echo "Pushed v$next — GitHub Actions is building the release."
