#!/usr/bin/env python3
"""Read a release's notes out of CHANGELOG.md.

    changelog.py notes 0.2.0     print the body of "## [0.2.0] - <date>"

Exits non-zero when the version has no section, so a release can't go out
without notes. The release workflow publishes this text as the GitHub release
body and as the notes the in-app updater shows.
"""
from __future__ import annotations

import re
import sys
from pathlib import Path

CHANGELOG = Path(__file__).resolve().parent.parent / "CHANGELOG.md"


def section(text: str, version: str) -> str | None:
    heading = re.compile(r"^## \[(?P<version>[^\]]+)\]", re.MULTILINE)
    matches = list(heading.finditer(text))
    for i, m in enumerate(matches):
        if m.group("version") == version:
            end = matches[i + 1].start() if i + 1 < len(matches) else len(text)
            body = text[m.end():end]
            body = body.split("\n", 1)[1] if "\n" in body else ""  # drop the rest of the heading line
            return body.strip()
    return None


def main() -> int:
    if len(sys.argv) != 3 or sys.argv[1] != "notes":
        print(__doc__.strip(), file=sys.stderr)
        return 2
    version = sys.argv[2].removeprefix("v")
    notes = section(CHANGELOG.read_text(encoding="utf-8"), version)
    if not notes:
        print(f"error: CHANGELOG.md has no notes under '## [{version}]'", file=sys.stderr)
        return 1
    print(notes)
    return 0


if __name__ == "__main__":
    sys.exit(main())
