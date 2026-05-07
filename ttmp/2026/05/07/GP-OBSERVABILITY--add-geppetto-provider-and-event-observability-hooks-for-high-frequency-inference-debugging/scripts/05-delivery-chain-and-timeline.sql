-- 05-delivery-chain-and-timeline.sql
-- Purpose: Inspect browser-visible delivery and persisted timeline state after
-- provider/Geppetto reasoning records have been correlated to frontend messages.

.headers on
.mode column

SELECT ordinal,
       pipeline_event,
       transport_fanout,
       frontend_parsed
  FROM delivery_chain
 WHERE frontend_parsed = 'yes'
 ORDER BY CAST(ordinal AS INTEGER)
 LIMIT 80;

SELECT kind,
       entity_id,
       created_ordinal,
       last_event_ordinal,
       tombstone,
       substr(payload_json, 1, 300) AS payload_preview
  FROM timeline_entities
 WHERE entity_id LIKE '%thinking%'
 ORDER BY created_ordinal;
