#!/usr/bin/env python3
"""Write latest.json, the manifest the in-app updater reads.

    make-latest-json.py --release-dir release --tag v0.2.0 \
        --repo relay-client/relay-db --notes-file release-notes.md

For each platform it records the URL and SHA-256 of the file that platform
installs, and the URL of that file's minisign signature, which must sit next
to it as <file>.minisig. Every platform below must be present: a release that
leaves one out would quietly stop updating it.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

# Platform key (see internal/update.currentTarget) → file it installs. The
# AppImage's name carries the version, so it is filled in per release.
ASSETS = {
    "darwin-universal": "relay-db-darwin-universal.app.zip",
    "windows-amd64": "relay-db-windows-amd64.exe",
    "windows-arm64": "relay-db-windows-arm64.exe",
    "linux-amd64": "relay-db-linux-amd64",
    "linux-amd64-appimage": "relay-db-{version}-linux-amd64.AppImage",
}


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1 << 20), b""):
            h.update(chunk)
    return h.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--release-dir", required=True, type=Path)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--repo", required=True)
    parser.add_argument("--notes-file", required=True, type=Path)
    args = parser.parse_args()

    tag = args.tag if args.tag.startswith("v") else "v" + args.tag
    version = tag[1:]
    base = f"https://github.com/{args.repo}/releases/download/{tag}"
    manifest = {
        "version": version,
        "notes": args.notes_file.read_text(encoding="utf-8").strip(),
        "published_at": datetime.now(timezone.utc).isoformat(timespec="seconds").replace("+00:00", "Z"),
        "platforms": {},
    }
    for platform, pattern in ASSETS.items():
        name = pattern.format(version=version)
        asset = args.release_dir / name
        signature = args.release_dir / (name + ".minisig")
        for required in (asset, signature):
            if not required.is_file():
                print(f"error: {platform}: missing {required}", file=sys.stderr)
                return 1
        manifest["platforms"][platform] = {
            "url": f"{base}/{name}",
            "sha256": sha256(asset),
            "signature": f"{base}/{name}.minisig",
        }

    out = args.release_dir / "latest.json"
    out.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {out}: {', '.join(manifest['platforms'])}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
