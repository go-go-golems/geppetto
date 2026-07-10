#!/usr/bin/env python3
"""Build a SQLite history database from git and docmgr ticket files.

The database is intentionally local to this docmgr ticket.  It records the last
three months of git commits plus ticket/doc metadata under ttmp/ so the report
can be regenerated with SQL instead of hand-counting.
"""

from __future__ import annotations

import argparse
import datetime as dt
import os
import re
import sqlite3
import subprocess
from pathlib import Path
from typing import Iterable


FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n", re.S)
TICKET_DIR_RE = re.compile(r"^(?P<ticket>[^/]+?)--(?P<slug>.+)$")


def run(repo: Path, args: list[str]) -> str:
    return subprocess.check_output(args, cwd=repo, text=True)


def git_lines(repo: Path, args: list[str]) -> list[str]:
    out = run(repo, ["git", *args])
    return out.splitlines()


def parse_iso_date(value: str) -> dt.date:
    return dt.datetime.fromisoformat(value.replace("Z", "+00:00")).date()


def parse_simple_frontmatter(text: str) -> dict[str, str]:
    match = FRONTMATTER_RE.match(text)
    if not match:
        return {}
    data: dict[str, str] = {}
    current_key: str | None = None
    list_values: list[str] = []
    for raw in match.group(1).splitlines():
        line = raw.rstrip()
        if not line:
            continue
        if line.startswith("    - ") and current_key:
            list_values.append(line[6:].strip().strip('"'))
            data[current_key] = ", ".join(list_values)
            continue
        if ":" in line and not line.startswith(" "):
            key, val = line.split(":", 1)
            current_key = key.strip()
            list_values = []
            data[current_key] = val.strip().strip('"')
    return data


def init_db(conn: sqlite3.Connection) -> None:
    conn.executescript(
        """
        PRAGMA journal_mode=WAL;
        DROP TABLE IF EXISTS commits;
        DROP TABLE IF EXISTS commit_files;
        DROP TABLE IF EXISTS docmgr_tickets;
        DROP TABLE IF EXISTS docmgr_docs;
        DROP TABLE IF EXISTS changelog_entries;

        CREATE TABLE commits (
          sha TEXT PRIMARY KEY,
          short_sha TEXT NOT NULL,
          author_date TEXT NOT NULL,
          author_day TEXT NOT NULL,
          author_name TEXT NOT NULL,
          subject TEXT NOT NULL,
          body TEXT NOT NULL,
          files_changed INTEGER NOT NULL DEFAULT 0,
          insertions INTEGER NOT NULL DEFAULT 0,
          deletions INTEGER NOT NULL DEFAULT 0,
          is_merge INTEGER NOT NULL DEFAULT 0,
          category TEXT NOT NULL
        );

        CREATE TABLE commit_files (
          sha TEXT NOT NULL,
          path TEXT NOT NULL,
          insertions INTEGER,
          deletions INTEGER,
          PRIMARY KEY (sha, path),
          FOREIGN KEY (sha) REFERENCES commits(sha)
        );

        CREATE TABLE docmgr_tickets (
          ticket TEXT PRIMARY KEY,
          path TEXT NOT NULL,
          ticket_day TEXT NOT NULL,
          title TEXT,
          status TEXT,
          topics TEXT,
          summary TEXT
        );

        CREATE TABLE docmgr_docs (
          path TEXT PRIMARY KEY,
          ticket TEXT,
          title TEXT,
          doc_type TEXT,
          status TEXT,
          topics TEXT,
          last_updated TEXT,
          word_count INTEGER NOT NULL,
          FOREIGN KEY (ticket) REFERENCES docmgr_tickets(ticket)
        );

        CREATE TABLE changelog_entries (
          id INTEGER PRIMARY KEY AUTOINCREMENT,
          ticket TEXT NOT NULL,
          path TEXT NOT NULL,
          entry TEXT NOT NULL,
          FOREIGN KEY (ticket) REFERENCES docmgr_tickets(ticket)
        );
        """
    )


def classify(subject: str) -> str:
    s = subject.lower()
    if s.startswith("merge"):
        return "merge"
    if "doc" in s or "diary" in s or "guide" in s or "audit" in s or "plan" in s:
        return "docs/research"
    if "test" in s or "smoke" in s or "coverage" in s:
        return "tests"
    if "bump" in s or "depend" in s or "upgrade" in s:
        return "dependencies"
    if "fix" in s or "bug" in s or "panic" in s or "preserve" in s:
        return "fixes"
    if "add" in s or "support" in s or "expose" in s or "implement" in s:
        return "features"
    return "maintenance"


def load_git(conn: sqlite3.Connection, repo: Path, since: str, until: str | None) -> None:
    args = [
        "log",
        f"--since={since}",
        "--date=iso-strict",
        "--pretty=format:%H%x1f%aI%x1f%an%x1f%s%x1f%b%x1e",
    ]
    if until:
        args.insert(2, f"--until={until}")
    raw = run(repo, ["git", *args])
    shas: list[str] = []
    for rec in raw.strip("\n\x1e").split("\x1e"):
        rec = rec.strip("\n")
        if not rec:
            continue
        parts = rec.split("\x1f", 4)
        if len(parts) != 5:
            continue
        sha, author_date, author_name, subject, body = parts
        shas.append(sha)
        conn.execute(
            """
            INSERT INTO commits(sha, short_sha, author_date, author_day, author_name, subject, body, is_merge, category)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                sha,
                sha[:12],
                author_date,
                parse_iso_date(author_date).isoformat(),
                author_name,
                subject.strip(),
                body.strip(),
                1 if subject.lower().startswith("merge") else 0,
                classify(subject),
            ),
        )

    for sha in shas:
        numstat = git_lines(repo, ["show", "--numstat", "--format=", sha])
        files_changed = insertions = deletions = 0
        for line in numstat:
            parts = line.split("\t")
            if len(parts) < 3:
                continue
            ins_raw, del_raw, path = parts[0], parts[1], parts[2]
            ins = None if ins_raw == "-" else int(ins_raw)
            dels = None if del_raw == "-" else int(del_raw)
            files_changed += 1
            insertions += ins or 0
            deletions += dels or 0
            conn.execute(
                "INSERT OR REPLACE INTO commit_files(sha, path, insertions, deletions) VALUES (?, ?, ?, ?)",
                (sha, path, ins, dels),
            )
        conn.execute(
            "UPDATE commits SET files_changed=?, insertions=?, deletions=? WHERE sha=?",
            (files_changed, insertions, deletions, sha),
        )


def ticket_dirs(ttmp: Path, since_day: dt.date) -> Iterable[Path]:
    for p in sorted(ttmp.glob("20[0-9][0-9]/*/*/*--*")):
        if not p.is_dir():
            continue
        try:
            day = dt.date(int(p.parts[-4]), int(p.parts[-3]), int(p.parts[-2]))
        except (ValueError, IndexError):
            continue
        if day >= since_day:
            yield p


def load_docmgr(conn: sqlite3.Connection, repo: Path, since: str) -> None:
    ttmp = repo / "ttmp"
    since_day = dt.date.fromisoformat(since[:10])
    for tdir in ticket_dirs(ttmp, since_day):
        m = TICKET_DIR_RE.match(tdir.name)
        if not m:
            continue
        ticket = m.group("ticket")
        ticket_day = f"{tdir.parts[-4]}-{tdir.parts[-3]}-{tdir.parts[-2]}"
        index = tdir / "index.md"
        meta = parse_simple_frontmatter(index.read_text(errors="replace")) if index.exists() else {}
        rel = str(tdir.relative_to(repo))
        conn.execute(
            """
            INSERT OR REPLACE INTO docmgr_tickets(ticket, path, ticket_day, title, status, topics, summary)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            """,
            (ticket, rel, ticket_day, meta.get("Title"), meta.get("Status"), meta.get("Topics"), meta.get("Summary")),
        )
        for md in sorted(tdir.rglob("*.md")):
            text = md.read_text(errors="replace")
            fm = parse_simple_frontmatter(text)
            words = len(re.findall(r"\b\w+\b", FRONTMATTER_RE.sub("", text)))
            conn.execute(
                """
                INSERT OR REPLACE INTO docmgr_docs(path, ticket, title, doc_type, status, topics, last_updated, word_count)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?)
                """,
                (
                    str(md.relative_to(repo)),
                    ticket,
                    fm.get("Title"),
                    fm.get("DocType"),
                    fm.get("Status"),
                    fm.get("Topics"),
                    fm.get("LastUpdated"),
                    words,
                ),
            )
        changelog = tdir / "changelog.md"
        if changelog.exists():
            for line in changelog.read_text(errors="replace").splitlines():
                stripped = line.strip()
                if stripped.startswith("- "):
                    conn.execute(
                        "INSERT INTO changelog_entries(ticket, path, entry) VALUES (?, ?, ?)",
                        (ticket, str(changelog.relative_to(repo)), stripped[2:]),
                    )


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--repo", default=".")
    ap.add_argument("--since", default="2026-03-14")
    ap.add_argument("--until", default=None)
    ap.add_argument("--db", required=True)
    args = ap.parse_args()

    repo = Path(args.repo).resolve()
    db = Path(args.db).resolve()
    db.parent.mkdir(parents=True, exist_ok=True)
    if db.exists():
        db.unlink()
    conn = sqlite3.connect(db)
    try:
        init_db(conn)
        load_git(conn, repo, args.since, args.until)
        load_docmgr(conn, repo, args.since)
        conn.commit()
    finally:
        conn.close()
    print(f"wrote {db}")


if __name__ == "__main__":
    main()
