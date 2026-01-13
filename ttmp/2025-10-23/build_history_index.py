#!/usr/bin/env python3
"""Build a SQLite index of the current git branch history.

The database captures commit metadata, per-file change details, and a light-weight
symbol index for Go/Python/TypeScript sources touched in each commit. The output
is written to ttmp/2025-10-23/git-history-and-code-index.db.
"""
import json
import os
import re
import sqlite3
import subprocess
from datetime import datetime, timezone
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Tuple

HERE = Path(__file__).resolve().parent
REPO_ROOT = HERE.parents[1]
DB_PATH = HERE / "git-history-and-code-index.db"

SYMBOL_REGEXES = {
    ".go": {
        "func": re.compile(r"^\s*func\s+(?:\([^)]*\)\s*)?([A-Za-z_][A-Za-z0-9_]*)"),
        "type": re.compile(r"^\s*type\s+([A-Za-z_][A-Za-z0-9_]*)"),
        "const": re.compile(r"^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)"),
        "var": re.compile(r"^\s*var\s+([A-Za-z_][A-Za-z0-9_]*)"),
    },
    ".py": {
        "class": re.compile(r"^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)"),
        "def": re.compile(r"^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)"),
    },
    ".ts": {
        "function": re.compile(r"^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)"),
        "class": re.compile(r"^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)"),
    },
    ".tsx": {
        "function": re.compile(r"^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)"),
        "class": re.compile(r"^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)"),
    },
    ".js": {
        "function": re.compile(r"^\s*function\s+([A-Za-z_][A-Za-z0-9_]*)"),
        "class": re.compile(r"^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)"),
    },
}


def run_git(args: Iterable[str]) -> bytes:
    return subprocess.check_output(["git", *args], cwd=REPO_ROOT)


def reset_db() -> sqlite3.Connection:
    if DB_PATH.exists():
        DB_PATH.unlink()
    conn = sqlite3.connect(DB_PATH)
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA synchronous=NORMAL")
    conn.execute("PRAGMA foreign_keys=ON")
    return conn


def ensure_schema(conn: sqlite3.Connection) -> None:
    conn.executescript(
        """
        CREATE TABLE commits (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            hash TEXT UNIQUE NOT NULL,
            parents TEXT,
            author_name TEXT,
            author_email TEXT,
            authored_at TEXT,
            committer_name TEXT,
            committer_email TEXT,
            committed_at TEXT,
            subject TEXT,
            body TEXT,
            document_summary TEXT
        );

        CREATE TABLE files (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            path TEXT UNIQUE NOT NULL
        );

        CREATE TABLE commit_files (
            commit_id INTEGER NOT NULL,
            file_id INTEGER NOT NULL,
            change_type TEXT,
            old_path TEXT,
            additions INTEGER,
            deletions INTEGER,
            PRIMARY KEY (commit_id, file_id),
            FOREIGN KEY (commit_id) REFERENCES commits(id),
            FOREIGN KEY (file_id) REFERENCES files(id)
        );

        CREATE TABLE commit_symbols (
            commit_id INTEGER NOT NULL,
            file_id INTEGER NOT NULL,
            symbol_name TEXT NOT NULL,
            symbol_kind TEXT,
            PRIMARY KEY (commit_id, file_id, symbol_name, symbol_kind),
            FOREIGN KEY (commit_id) REFERENCES commits(id),
            FOREIGN KEY (file_id) REFERENCES files(id)
        );

        CREATE TABLE IF NOT EXISTS analysis_notes (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            commit_id INTEGER,
            file_id INTEGER,
            note_type TEXT,
            note TEXT NOT NULL,
            tags TEXT,
            created_at TEXT DEFAULT (datetime('now')),
            FOREIGN KEY (commit_id) REFERENCES commits(id),
            FOREIGN KEY (file_id) REFERENCES files(id)
        );

        CREATE TABLE IF NOT EXISTS prs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT UNIQUE NOT NULL,
            description TEXT,
            status TEXT,
            created_at TEXT DEFAULT (datetime('now')),
            updated_at TEXT
        );

        CREATE TABLE IF NOT EXISTS pr_changelog (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            pr_id INTEGER,
            commit_id INTEGER,
            file_id INTEGER,
            action TEXT NOT NULL,
            details TEXT,
            created_at TEXT DEFAULT (datetime('now')),
            FOREIGN KEY (pr_id) REFERENCES prs(id),
            FOREIGN KEY (commit_id) REFERENCES commits(id),
            FOREIGN KEY (file_id) REFERENCES files(id)
        );

        CREATE INDEX idx_commit_files_commit ON commit_files (commit_id);
        CREATE INDEX idx_commit_files_file ON commit_files (file_id);
        CREATE INDEX idx_commit_symbols_commit ON commit_symbols (commit_id);
        CREATE INDEX idx_commit_symbols_file ON commit_symbols (file_id);
        CREATE INDEX idx_commit_symbols_symbol ON commit_symbols (symbol_name);
        CREATE INDEX idx_analysis_notes_commit ON analysis_notes (commit_id);
        CREATE INDEX idx_analysis_notes_file ON analysis_notes (file_id);
        CREATE INDEX IF NOT EXISTS idx_prs_name ON prs (name);
        CREATE INDEX IF NOT EXISTS idx_pr_changelog_pr ON pr_changelog (pr_id);
        CREATE INDEX IF NOT EXISTS idx_pr_changelog_commit ON pr_changelog (commit_id);
        CREATE INDEX IF NOT EXISTS idx_pr_changelog_file ON pr_changelog (file_id);
        """
    )
    conn.commit()


def get_or_create_file_id(conn: sqlite3.Connection, path: str) -> int:
    cur = conn.execute("SELECT id FROM files WHERE path = ?", (path,))
    row = cur.fetchone()
    if row:
        return row[0]
    cur = conn.execute("INSERT INTO files(path) VALUES (?)", (path,))
    conn.commit()
    return cur.lastrowid


def parse_commit_metadata(commit_hash: str) -> Dict[str, str]:
    raw = run_git([
        "show",
        "-s",
        "--format=%H%x00%P%x00%an%x00%ae%x00%at%x00%cn%x00%ce%x00%ct%x00%s%x00%b",
        commit_hash,
    ])
    parts = raw.decode("utf-8", errors="replace").split("\x00")
    (
        commit_hash,
        parents,
        author_name,
        author_email,
        authored_ts,
        committer_name,
        committer_email,
        committed_ts,
        subject,
        body,
    ) = parts[:10]
    return {
        "hash": commit_hash,
        "parents": parents.strip(),
        "author_name": author_name,
        "author_email": author_email,
        "authored_at": timestamp_to_iso(authored_ts),
        "committer_name": committer_name,
        "committer_email": committer_email,
        "committed_at": timestamp_to_iso(committed_ts),
        "subject": subject,
        "body": body.strip(),
    }


def timestamp_to_iso(ts: str) -> str:
    try:
        value = int(ts)
    except ValueError:
        return ts
    return datetime.fromtimestamp(value, tz=timezone.utc).isoformat()


def parse_status_entries(commit_hash: str) -> List[Dict[str, Optional[str]]]:
    output = run_git([
        "diff-tree",
        "--root",
        "--no-commit-id",
        "-r",
        "-z",
        "--name-status",
        commit_hash,
    ])
    parts = output.split(b"\x00")
    entries: List[Dict[str, Optional[str]]] = []
    idx = 0
    while idx < len(parts):
        raw = parts[idx]
        idx += 1
        if not raw:
            continue
        status = raw.decode("utf-8", errors="replace")
        code = status[0]
        score = status[1:] if len(status) > 1 else ""
        if code in {"R", "C"}:
            if idx + 1 >= len(parts):
                break
            old_path = parts[idx].decode("utf-8", errors="replace")
            new_path = parts[idx + 1].decode("utf-8", errors="replace")
            idx += 2
            entries.append(
                {
                    "status": code,
                    "score": score,
                    "path": new_path,
                    "old_path": old_path,
                    "status_raw": status,
                }
            )
        else:
            if idx >= len(parts):
                break
            path = parts[idx].decode("utf-8", errors="replace")
            idx += 1
            entries.append(
                {
                    "status": code,
                    "score": score,
                    "path": path,
                    "old_path": None if code != "D" else path,
                    "status_raw": status,
                }
            )
    return entries


def parse_numstat_entries(commit_hash: str) -> Dict[str, Dict[str, Optional[int]]]:
    output = run_git([
        "diff-tree",
        "--root",
        "--no-commit-id",
        "-r",
        "--numstat",
        "-M",
        "-z",
        commit_hash,
    ])
    parts = [p for p in output.split(b"\x00") if p]
    stats: Dict[str, Dict[str, Optional[int]]] = {}
    idx = 0
    while idx < len(parts):
        entry = parts[idx].decode("utf-8", errors="replace")
        idx += 1
        fields = entry.split("\t")
        if len(fields) < 2:
            continue
        add_str, del_str = fields[0], fields[1]
        additions = None if add_str == "-" else int(add_str)
        deletions = None if del_str == "-" else int(del_str)
        if len(fields) > 2 and fields[2]:
            path = "\t".join(fields[2:])
            stats[path] = {
                "additions": additions,
                "deletions": deletions,
                "old_path": None,
            }
        else:
            # Rename/Copies encode paths in the following entries.
            if idx + 1 >= len(parts):
                break
            old_path = parts[idx].decode("utf-8", errors="replace")
            new_path = parts[idx + 1].decode("utf-8", errors="replace")
            idx += 2
            stats[new_path] = {
                "additions": additions,
                "deletions": deletions,
                "old_path": old_path,
            }
    return stats


def extract_symbols(path: str, content: str) -> List[Tuple[str, str]]:
    suffix = Path(path).suffix.lower()
    regex_map = SYMBOL_REGEXES.get(suffix)
    if not regex_map:
        return []
    results: List[Tuple[str, str]] = []
    seen = set()
    for line in content.splitlines():
        for kind, regex in regex_map.items():
            match = regex.match(line)
            if match:
                name = match.group(1)
                if name and name not in seen:
                    seen.add(name)
                    results.append((name, kind))
    return results


def fetch_file_content(commit_hash: str, path: str) -> Optional[str]:
    try:
        data = run_git(["show", f"{commit_hash}:{path}"])
    except subprocess.CalledProcessError:
        return None
    try:
        return data.decode("utf-8")
    except UnicodeDecodeError:
        return None


def build_history_index() -> None:
    conn = reset_db()
    ensure_schema(conn)

    commit_hashes = (
        run_git(["rev-list", "--reverse", "HEAD"]).decode("utf-8").strip().splitlines()
    )
    total = len(commit_hashes)
    for idx, commit_hash in enumerate(commit_hashes, start=1):
        metadata = parse_commit_metadata(commit_hash)
        conn.execute(
            """
            INSERT INTO commits(
                hash, parents, author_name, author_email, authored_at,
                committer_name, committer_email, committed_at, subject, body, document_summary
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                metadata["hash"],
                metadata["parents"],
                metadata["author_name"],
                metadata["author_email"],
                metadata["authored_at"],
                metadata["committer_name"],
                metadata["committer_email"],
                metadata["committed_at"],
                metadata["subject"],
                metadata["body"],
                None,
            ),
        )
        commit_id = conn.execute("SELECT id FROM commits WHERE hash = ?", (metadata["hash"],)).fetchone()[0]

        status_entries = parse_status_entries(commit_hash)
        numstat_entries = parse_numstat_entries(commit_hash)

        summary = {
            "added": [],
            "modified": [],
            "deleted": [],
            "renamed": [],
            "copied": [],
        }

        for entry in status_entries:
            path = entry["path"]
            change_type = entry["status_raw"]
            old_path = entry.get("old_path")
            stats = numstat_entries.get(path)
            additions = stats.get("additions") if stats else None
            deletions = stats.get("deletions") if stats else None

            file_path_for_index = path if entry["status"] != "D" else (old_path or path)
            file_id = get_or_create_file_id(conn, file_path_for_index)
            conn.execute(
                """
                INSERT OR REPLACE INTO commit_files(
                    commit_id, file_id, change_type, old_path, additions, deletions
                ) VALUES (?, ?, ?, ?, ?, ?)
                """,
                (
                    commit_id,
                    file_id,
                    change_type,
                    old_path,
                    additions,
                    deletions,
                ),
            )

            if entry["status"] == "A":
                summary["added"].append(path)
            elif entry["status"] == "M":
                summary["modified"].append(path)
            elif entry["status"] == "D":
                summary["deleted"].append(old_path or path)
            elif entry["status"] == "R":
                summary["renamed"].append({"from": old_path, "to": path})
            elif entry["status"] == "C":
                summary["copied"].append({"from": old_path, "to": path})

            # Symbol extraction for files present in this commit.
            if entry["status"] != "D":
                content = fetch_file_content(commit_hash, path)
                if content is None:
                    continue
                for symbol_name, symbol_kind in extract_symbols(path, content):
                    conn.execute(
                        """
                        INSERT OR IGNORE INTO commit_symbols(
                            commit_id, file_id, symbol_name, symbol_kind
                        ) VALUES (?, ?, ?, ?)
                        """,
                        (commit_id, file_id, symbol_name, symbol_kind),
                    )

        conn.execute(
            "UPDATE commits SET document_summary = ? WHERE id = ?",
            (json.dumps(summary, sort_keys=True), commit_id),
        )
        conn.commit()

        if idx % 25 == 0 or idx == total:
            print(f"Processed {idx}/{total} commits", flush=True)


if __name__ == "__main__":
    build_history_index()
