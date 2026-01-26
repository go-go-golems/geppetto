#!/usr/bin/env python3
"""List timeline/addEntity actions from a Redux DevTools state.json dump."""

import json
import pathlib
import sys


def main() -> int:
    path = pathlib.Path(sys.argv[1]) if len(sys.argv) > 1 else pathlib.Path("/home/manuel/Downloads/state.json")
    obj = json.loads(path.read_text())
    actions = obj.get("actionsById", {})
    for action_id, entry in actions.items():
        action = entry.get("action", {})
        if action.get("type") != "timeline/addEntity":
            continue
        payload = action.get("payload") or {}
        props = payload.get("props") or {}
        print(action_id, payload.get("id"), props.get("role"), props.get("content"))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
