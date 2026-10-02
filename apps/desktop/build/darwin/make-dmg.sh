#!/bin/sh
set -eu

usage="usage: make-dmg.sh <app> <output.dmg> <volume name>"
app="${1:?$usage}"
out="${2:?$usage}"
volume="${3:?$usage}"
here="$(cd "$(dirname "$0")" && pwd)"

[ -d "$app" ] || { echo "make-dmg: $app not found — build the app first" >&2; exit 1; }
command -v dmgbuild >/dev/null || { echo "make-dmg: needs dmgbuild — pip3 install -r $here/requirements.txt" >&2; exit 1; }

mkdir -p "$(dirname "$out")"
rm -f "$out"
dmgbuild -s "$here/dmg-settings.py" -D app="$app" -D settings="$here/dmg-settings.py" "$volume" "$out"
