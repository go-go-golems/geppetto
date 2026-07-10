#!/usr/bin/env python3
"""Render a Markdown history report from the SQLite history database."""

from __future__ import annotations

import argparse
import sqlite3
from pathlib import Path


def q(conn: sqlite3.Connection, sql: str, params=()):
    conn.row_factory = sqlite3.Row
    return conn.execute(sql, params).fetchall()


def table(rows) -> str:
    if not rows:
        return "_No rows._\n"
    headers = rows[0].keys()
    out = ["| " + " | ".join(headers) + " |", "| " + " | ".join("---" for _ in headers) + " |"]
    for r in rows:
        out.append("| " + " | ".join(str(r[h]) if r[h] is not None else "" for h in headers) + " |")
    return "\n".join(out) + "\n"


def scalar(conn, sql: str):
    return conn.execute(sql).fetchone()[0]


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--since", default="2026-03-14")
    ap.add_argument("--until", default="2026-06-14")
    args = ap.parse_args()

    conn = sqlite3.connect(args.db)
    conn.row_factory = sqlite3.Row

    overall = q(conn, """
        SELECT COUNT(*) AS commits, SUM(is_merge) AS merge_commits,
               SUM(CASE WHEN is_merge = 0 THEN 1 ELSE 0 END) AS non_merge_commits,
               SUM(files_changed) AS files_changed, SUM(insertions) AS insertions,
               SUM(deletions) AS deletions, MIN(author_day) AS first_day, MAX(author_day) AS last_day
        FROM commits
    """)[0]
    tickets = scalar(conn, "SELECT COUNT(*) FROM docmgr_tickets")
    docs = scalar(conn, "SELECT COUNT(*) FROM docmgr_docs")
    words = scalar(conn, "SELECT COALESCE(SUM(word_count), 0) FROM docmgr_docs")

    month_category = q(conn, """
        SELECT substr(author_day, 1, 7) AS month, category, COUNT(*) AS commits,
               SUM(insertions) AS insertions, SUM(deletions) AS deletions
        FROM commits GROUP BY month, category ORDER BY month, commits DESC
    """)
    months = q(conn, """
        SELECT substr(author_day, 1, 7) AS month, COUNT(*) AS commits,
               SUM(files_changed) AS files_changed, SUM(insertions) AS insertions, SUM(deletions) AS deletions
        FROM commits GROUP BY month ORDER BY month
    """)
    top_paths = q(conn, """
        SELECT path, COUNT(*) AS commits, SUM(COALESCE(insertions,0)) AS insertions,
               SUM(COALESCE(deletions,0)) AS deletions
        FROM commit_files GROUP BY path ORDER BY commits DESC, insertions + deletions DESC LIMIT 20
    """)
    largest_tickets = q(conn, """
        SELECT t.ticket, t.ticket_day, t.title, COUNT(d.path) AS docs, SUM(d.word_count) AS words
        FROM docmgr_tickets t LEFT JOIN docmgr_docs d ON d.ticket = t.ticket
        GROUP BY t.ticket ORDER BY words DESC LIMIT 15
    """)
    doc_months = q(conn, """
        SELECT month, COUNT(*) AS tickets, SUM(doc_count) AS docs, SUM(words) AS words
        FROM (
          SELECT substr(t.ticket_day, 1, 7) AS month, t.ticket,
                 COUNT(d.path) AS doc_count, COALESCE(SUM(d.word_count), 0) AS words
          FROM docmgr_tickets t LEFT JOIN docmgr_docs d ON d.ticket = t.ticket
          GROUP BY t.ticket
        )
        GROUP BY month ORDER BY month
    """)
    recent = q(conn, """
        SELECT author_day, short_sha, category, subject
        FROM commits ORDER BY author_date DESC LIMIT 30
    """)

    narrative = []
    narrative.append("# Last Three Months History\n")
    narrative.append("## Goal\n")
    narrative.append(
        f"This document summarizes repository activity from **{args.since}** through **{args.until}** using git history and docmgr ticket metadata loaded into a local SQLite database.\n"
    )
    narrative.append("## Data set\n")
    narrative.append(
        f"The SQLite database contains **{overall['commits']} commits** ({overall['non_merge_commits']} non-merge, {overall['merge_commits']} merge), "
        f"touching **{overall['files_changed']} file entries** with **{overall['insertions']} insertions** and **{overall['deletions']} deletions**. "
        f"The docmgr side contains **{tickets} tickets**, **{docs} markdown documents**, and roughly **{words} words** in tickets dated in the same window.\n"
    )
    narrative.append("## High-level read\n")
    narrative.append(
        "The period was dense and documentation-heavy. Git activity clusters around provider/runtime work, Goja and JavaScript integration, image and multimodal input support, reasoning/streaming compatibility, embeddings exposure, and repeated provider correctness fixes. Docmgr activity shows substantial parallel research and implementation planning, with large tickets acting as evidence stores for provider audits, Gemini modernization, image input, and related implementation diaries.\n"
    )
    narrative.append("## Git activity by month\n")
    narrative.append(table(months))
    narrative.append("## Git activity by month and category\n")
    narrative.append(table(month_category))
    narrative.append("## Most frequently touched paths\n")
    narrative.append(table(top_paths))
    narrative.append("## Docmgr activity by month\n")
    narrative.append(table(doc_months))
    narrative.append("## Largest docmgr tickets\n")
    narrative.append(table(largest_tickets))
    narrative.append("## Recent notable commits\n")
    narrative.append(table(recent))
    narrative.append("## Interpretation\n")
    narrative.append(
        "1. **Provider compatibility was a dominant theme.** Commit subjects mention Claude extended thinking, Gemini function-call thought signatures, Gemini SDK modernization, image input mapping, and provider API gap audits.\n"
        "2. **Runtime and scripting integration expanded.** The history includes Goja runtime flags, JavaScript embeddings exposure, provider resource closer registration, and host service contributions.\n"
        "3. **The project used docmgr as an engineering memory layer.** Large research and diary tickets account for a substantial amount of written context, making the commit stream easier to interpret than git alone.\n"
        "4. **Correctness fixes followed feature additions quickly.** Several commits after larger feature/documentation pushes are targeted fixes around preserving settings, avoiding dynamic SQL, nil API panic handling, session filters, and reasoning block signing.\n"
    )
    narrative.append("## Reproduction\n")
    narrative.append(
        "Regenerate the database and report from the ticket root with:\n\n"
        "```bash\n"
        "python3 scripts/01_build_history_db.py --repo ../../../../.. --since 2026-03-14 --db various/history.sqlite\n"
        "sqlite3 various/history.sqlite < scripts/02_summary_queries.sql > various/01-history-summary.md\n"
        "python3 scripts/03_render_history_report.py --db various/history.sqlite --out analysis/01-last-three-months-history.md --since 2026-03-14 --until 2026-06-14\n"
        "```\n"
    )

    out = Path(args.out)
    existing = out.read_text() if out.exists() else ""
    if existing.startswith("---\n"):
        end = existing.find("\n---\n", 4)
        if end != -1:
            content = existing[: end + 5] + "\n\n" + "\n".join(narrative)
        else:
            content = "\n".join(narrative)
    else:
        content = "\n".join(narrative)
    out.write_text(content)
    print(f"wrote {out}")


if __name__ == "__main__":
    main()
