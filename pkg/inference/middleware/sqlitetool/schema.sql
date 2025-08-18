-- Example schema for the SQLite Tool middleware
-- Includes a `_prompts` table read by the middleware and a small demo schema

PRAGMA foreign_keys = ON;

-- Table read by middleware; its contents are appended as system prompts
CREATE TABLE IF NOT EXISTS _prompts (
  prompt TEXT NOT NULL
);

-- Demo application tables
CREATE TABLE IF NOT EXISTS authors (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS books (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  author_id INTEGER NOT NULL,
  title TEXT NOT NULL,
  published_year INTEGER,
  FOREIGN KEY (author_id) REFERENCES authors(id)
);

-- Optional: indexes to speed up joins/filters
CREATE INDEX IF NOT EXISTS idx_books_author_id ON books(author_id);


