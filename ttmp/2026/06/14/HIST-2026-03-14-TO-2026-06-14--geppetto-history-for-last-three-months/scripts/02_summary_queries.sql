-- Summary queries for the Geppetto last-three-months history database.
-- Run with:
--   sqlite3 various/history.sqlite < scripts/02_summary_queries.sql

.mode markdown
.headers on

.print '# Overall git activity'
SELECT
  COUNT(*) AS commits,
  SUM(is_merge) AS merge_commits,
  SUM(CASE WHEN is_merge = 0 THEN 1 ELSE 0 END) AS non_merge_commits,
  SUM(files_changed) AS files_changed,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions,
  MIN(author_day) AS first_day,
  MAX(author_day) AS last_day
FROM commits;

.print '\n# Commits by month and category'
SELECT
  substr(author_day, 1, 7) AS month,
  category,
  COUNT(*) AS commits,
  SUM(files_changed) AS files_changed,
  SUM(insertions) AS insertions,
  SUM(deletions) AS deletions
FROM commits
GROUP BY month, category
ORDER BY month, commits DESC;

.print '\n# Top touched paths'
SELECT
  path,
  COUNT(*) AS commits,
  SUM(COALESCE(insertions, 0)) AS insertions,
  SUM(COALESCE(deletions, 0)) AS deletions
FROM commit_files
GROUP BY path
ORDER BY commits DESC, insertions + deletions DESC
LIMIT 30;

.print '\n# Docmgr ticket count by month'
SELECT
  substr(ticket_day, 1, 7) AS month,
  COUNT(*) AS tickets,
  SUM((SELECT COUNT(*) FROM docmgr_docs d WHERE d.ticket = t.ticket)) AS docs,
  SUM((SELECT COALESCE(SUM(word_count), 0) FROM docmgr_docs d WHERE d.ticket = t.ticket)) AS words
FROM docmgr_tickets t
GROUP BY month
ORDER BY month;

.print '\n# Largest docmgr tickets by written words'
SELECT
  t.ticket,
  t.ticket_day,
  t.title,
  COUNT(d.path) AS docs,
  SUM(d.word_count) AS words
FROM docmgr_tickets t
LEFT JOIN docmgr_docs d ON d.ticket = t.ticket
GROUP BY t.ticket
ORDER BY words DESC
LIMIT 20;

.print '\n# Recent notable commits'
SELECT author_day, short_sha, category, subject
FROM commits
ORDER BY author_date DESC
LIMIT 40;
