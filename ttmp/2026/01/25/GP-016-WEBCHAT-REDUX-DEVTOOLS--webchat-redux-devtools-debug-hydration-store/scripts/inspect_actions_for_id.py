#!/usr/bin/env python3
"""Print actions in a Redux DevTools state.json that touch a specific entity id."""

import json
import pathlib
import sys


def main() -> int:
    if len(sys.argv) < 2:
        print("usage: inspect_actions_for_id.py <entity-id> [path]", file=sys.stderr)
        return 2
    entity_id = sys.argv[1]
    path = pathlib.Path(sys.argv[2]) if len(sys.argv) > 2 else pathlib.Path("/home/manuel/Downloads/state.json")
    obj = json.loads(path.read_text())
    actions = obj.get("actionsById", {})
    for action_id, entry in actions.items():
        action = entry.get("action", {})
        payload = action.get("payload")
        if isinstance(payload, dict) and payload.get("id") == entity_id:
            print(action_id, action.get("type"), payload)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
