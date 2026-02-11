#!/usr/bin/env python3
import argparse
import sqlite3


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", required=True)
    ap.add_argument("--conv-id", required=True)
    args = ap.parse_args()

    conn = sqlite3.connect(args.db)
    try:
        cur = conn.execute(
            "SELECT entity_id, kind, created_at_ms, updated_at_ms, version FROM timeline_entities WHERE conv_id = ? ORDER BY version ASC, entity_id ASC",
            (args.conv_id,),
        )
        for idx, row in enumerate(cur.fetchall()):
            entity_id, kind, created, updated, version = row
            print(f"{idx:03d} id={entity_id} kind={kind} version={version} createdAtMs={created} updatedAtMs={updated}")
    finally:
        conn.close()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
