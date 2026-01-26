#!/usr/bin/env python3
"""List user message entries from a Redux DevTools state.json dump."""

import json
import pathlib
import sys


def main() -> int:
    path = pathlib.Path(sys.argv[1]) if len(sys.argv) > 1 else pathlib.Path("/home/manuel/Downloads/state.json")
    obj = json.loads(path.read_text())
    state = obj.get("committedState", {})
    by_id = state.get("timeline", {}).get("byId", {})
    users = []
    for key, val in by_id.items():
        props = val.get("props") or {}
        if props.get("role") == "user":
            users.append((key, val.get("createdAt"), props.get("content")))
    print("user entries:")
    for entry in users:
        print(entry)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
