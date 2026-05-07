-- 06-geppetto-reasoning-to-frontend-view.sql
-- Purpose: Use the built-in SQLite view added by GP-OBSERVABILITY to inspect
-- provider -> Geppetto -> backend -> frontend -> timeline reasoning correlation.

.headers on
.mode column

SELECT rn,
       provider_record_id,
       provider_item_id,
       quote(provider_delta) AS provider_delta,
       geppetto_event_record_id,
       quote(geppetto_delta) AS geppetto_delta,
       backend_ordinal,
       backend_message_id,
       frontend_ordinal,
       frontend_message_id,
       ui_mutation_message_id,
       timeline_entity_id,
       timeline_created_ordinal,
       timeline_last_event_ordinal
  FROM geppetto_reasoning_to_frontend
 ORDER BY rn
 LIMIT 80;

SELECT COUNT(*) AS correlated_rows,
       SUM(CASE WHEN geppetto_delta = frontend_chunk THEN 1 ELSE 0 END) AS geppetto_frontend_exact_matches,
       SUM(CASE WHEN geppetto_delta != frontend_chunk THEN 1 ELSE 0 END) AS geppetto_frontend_mismatches,
       SUM(CASE WHEN provider_delta = frontend_chunk THEN 1 ELSE 0 END) AS provider_frontend_exact_matches,
       SUM(CASE WHEN provider_delta != frontend_chunk THEN 1 ELSE 0 END) AS provider_frontend_mismatches
  FROM geppetto_reasoning_to_frontend;
