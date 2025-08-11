-- Example seed data for Agent Mode middleware (SQLite)
-- Insert a few modes with prompts and allowed tools

INSERT INTO agent_modes (name, prompt, allowed_tools) VALUES
  ('chat',
   'You are in chat mode; prefer concise and helpful answers.',
   'echo'),
  ('clock',
   'You are in clock mode; you may use time_now when necessary.',
   'time_now'),
  ('research',
   'You are in research mode; be thorough and cite sources when appropriate.',
   'web_search,fetch_url');

-- Example: record a couple of changes
INSERT INTO agent_mode_changes (run_id, turn_id, from_mode, to_mode, analysis, at) VALUES
  ('run-1', 'turn-1', NULL, 'chat', 'Initial mode selection for a greeting.', datetime('now')),
  ('run-1', 'turn-2', 'chat', 'clock', 'User asked for the current time; switching to clock.', datetime('now'));


