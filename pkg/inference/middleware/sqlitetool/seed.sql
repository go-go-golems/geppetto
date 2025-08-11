-- Seed data for the SQLite Tool middleware demo

-- Prompts consumed by the middleware and injected into the Turn as system text
INSERT INTO _prompts (prompt) VALUES
  ('You can use the sql_query tool to run read-only queries. Return concise results.'),
  ('When uncertain about table names, first list tables from sqlite_master.');

-- Demo data
INSERT INTO authors (name) VALUES ('Ursula K. Le Guin'), ('Iain M. Banks');

INSERT INTO books (author_id, title, published_year) VALUES
  ((SELECT id FROM authors WHERE name = 'Ursula K. Le Guin'), 'A Wizard of Earthsea', 1968),
  ((SELECT id FROM authors WHERE name = 'Ursula K. Le Guin'), 'The Left Hand of Darkness', 1969),
  ((SELECT id FROM authors WHERE name = 'Iain M. Banks'), 'Consider Phlebas', 1987);


