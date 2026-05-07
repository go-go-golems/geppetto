-- 01-meta-and-counts.sql
-- Purpose: Confirm the reconcile artifact contains the expected frontend, backend,
-- and Geppetto record volumes for one debug session.

.headers on
.mode column

SELECT key, value
  FROM meta
 WHERE key IN ('session_id', 'backend_record_count', 'frontend_record_count', 'geppetto_record_count')
 ORDER BY key;

SELECT 'frontend_records' AS table_name, COUNT(*) AS rows FROM frontend_records
UNION ALL SELECT 'backend_records', COUNT(*) FROM backend_records
UNION ALL SELECT 'geppetto_records', COUNT(*) FROM geppetto_records
UNION ALL SELECT 'geppetto_provider_events', COUNT(*) FROM geppetto_provider_events
UNION ALL SELECT 'geppetto_emitted_events', COUNT(*) FROM geppetto_emitted_events
UNION ALL SELECT 'geppetto_reasoning_sequence', COUNT(*) FROM geppetto_reasoning_sequence
UNION ALL SELECT 'geppetto_summary_without_item_id', COUNT(*) FROM geppetto_summary_without_item_id
UNION ALL SELECT 'geppetto_publish_errors', COUNT(*) FROM geppetto_publish_errors
UNION ALL SELECT 'delivery_chain', COUNT(*) FROM delivery_chain;
