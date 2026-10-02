#!/bin/sh
# Packs an .app into a disk image with an Applications shortcut beside it,
# the usual drag-to-install layout. Uses only hdiutil, which every Mac has.
#
#   make-dmg.sh "build/bin/Relay DB.app" relay-db-0.2.0-darwin-arm64.dmg "Relay DB 0.2.0"
set -eu

app="${1:?usage: make-dmg.sh <app> <output.dmg> <volume name>}"
out="${2:?usage: make-dmg.sh <app> <output.dmg> <volume name>}"
volume="${3:?usage: make-dmg.sh <app> <output.dmg> <volume name>}"

[ -d "$app" ] || { echo "make-dmg: $app not found — build the app first" >&2; exit 1; }

stage="$(mktemp -d)"
trap 'rm -rf "$stage"' EXIT
ditto "$app" "$stage/$(basename "$app")"
ln -s /Applications "$stage/Applications"

mkdir -p "$(dirname "$out")"
rm -f "$out"
hdiutil create -volname "$volume" -srcfolder "$stage" -fs HFS+ -format UDZO -ov "$out"
