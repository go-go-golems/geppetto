#!/usr/bin/env python3
import argparse
import json
import sys


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("path", help="timeline JSON from /timeline")
    args = ap.parse_args()

    with open(args.path, "r", encoding="utf-8") as f:
        data = json.load(f)

    entities = data.get("entities", [])
    for idx, ent in enumerate(entities):
        msg = ent.get("message") or {}
        role = msg.get("role") or ent.get("kind")
        content = msg.get("content")
        created = ent.get("createdAtMs")
        updated = ent.get("updatedAtMs")
        ent_id = ent.get("id")
        print(f"{idx:03d} id={ent_id} role={role} createdAtMs={created} updatedAtMs={updated} content={repr(content)[:120]}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
