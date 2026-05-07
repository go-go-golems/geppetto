-- 02-geppetto-reasoning-sequence.sql
-- Purpose: Inspect low-level provider reasoning records and Geppetto emitted
-- reasoning/info events in timestamp/record order.

.headers on
.mode column

SELECT record_id,
       ts,
       stage,
       event_type,
       info_message,
       response_id,
       item_id,
       output_index,
       summary_index,
       delta_len,
       normalized_delta_len,
       buffer_len,
       error
  FROM geppetto_reasoning_sequence
 ORDER BY ts, record_id
 LIMIT 80;

SELECT COUNT(*) AS summary_records_without_item_id
  FROM geppetto_summary_without_item_id;

SELECT record_id,
       geppetto_event_type,
       info_message,
       item_id,
       substr(event_json, 1, 240) AS event_json_preview
  FROM geppetto_emitted_events
 WHERE info_message = 'reasoning-summary'
 ORDER BY record_id
 LIMIT 10;
