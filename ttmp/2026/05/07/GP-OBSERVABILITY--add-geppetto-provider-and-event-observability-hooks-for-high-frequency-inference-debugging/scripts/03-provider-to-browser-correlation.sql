-- 03-provider-to-browser-correlation.sql
-- Purpose: Correlate decoded provider reasoning summary deltas to Geppetto
-- published partial-thinking deltas, backend Sessionstream reasoning events,
-- frontend parsed browser events, frontend UI mutations, and timeline entities.
--
-- This uses row_number ordering because provider IDs are not yet propagated into
-- frontend ReasoningUpdate payloads. The next schema improvement should add
-- provider response/item/output/summary IDs to ReasoningUpdate for direct joins.

.headers on
.mode column

WITH
provider_delta AS (
  SELECT row_number() OVER (ORDER BY record_id) AS rn,
         record_id AS provider_record_id,
         response_id,
         item_id,
         output_index,
         summary_index,
         json_extract(object_json, '$.delta') AS provider_delta
    FROM geppetto_records
   WHERE stage = 'provider_normalize_delta'
     AND event_type = 'response.reasoning_summary_text.delta'
),
geppetto_delta AS (
  SELECT row_number() OVER (ORDER BY record_id) AS rn,
         record_id AS geppetto_event_record_id,
         json_extract(event_json, '$.delta') AS geppetto_delta,
         message_id AS geppetto_message_id
    FROM geppetto_records
   WHERE stage = 'geppetto_publish_done'
     AND event_type = 'partial-thinking'
),
backend_reasoning AS (
  SELECT row_number() OVER (ORDER BY CAST(br.ordinal AS INTEGER)) AS rn,
         br.ordinal AS backend_ordinal,
         bp.event_name AS backend_event_name,
         json_extract(bpue.payload_json, '$.messageId') AS backend_message_id,
         json_extract(bpue.payload_json, '$.chunk') AS backend_chunk
    FROM backend_pipeline bp
    JOIN backend_records br ON br.id = bp.record_id
    JOIN backend_pipeline_ui_events bpue ON bpue.record_id = br.id
   WHERE bp.event_name = 'ChatReasoningDelta'
     AND bpue.source = 'uiEvents'
),
frontend_reasoning AS (
  SELECT row_number() OVER (ORDER BY CAST(fr.ordinal AS INTEGER)) AS rn,
         fr.ordinal AS frontend_ordinal,
         fpf.name AS frontend_event_name,
         json_extract(fpf.frame_json, '$.payload.messageId') AS frontend_message_id,
         json_extract(fpf.frame_json, '$.payload.chunk') AS frontend_chunk
    FROM frontend_parsed_frames fpf
    JOIN frontend_records fr ON fr.id = fpf.record_id
   WHERE fpf.name = 'ChatReasoningAppended'
),
frontend_mutation AS (
  SELECT fr.ordinal,
         fui.name AS frontend_ui_event_name,
         fui.message_id AS ui_mutation_message_id
    FROM frontend_ui_events fui
    JOIN frontend_records fr ON fr.id = fui.record_id
   WHERE fui.name = 'ChatReasoningAppended'
)
SELECT pd.rn,
       pd.provider_record_id,
       pd.response_id,
       pd.item_id AS provider_item_id,
       pd.summary_index,
       quote(pd.provider_delta) AS provider_delta,
       gd.geppetto_event_record_id,
       quote(gd.geppetto_delta) AS geppetto_delta,
       br.backend_ordinal,
       br.backend_message_id,
       fr.frontend_ordinal,
       fr.frontend_message_id,
       fm.ui_mutation_message_id,
       te.entity_id AS timeline_entity_id,
       te.created_ordinal AS timeline_created_ordinal,
       te.last_event_ordinal AS timeline_last_event_ordinal
  FROM provider_delta pd
  JOIN geppetto_delta gd ON gd.rn = pd.rn
  JOIN backend_reasoning br ON br.rn = pd.rn
  JOIN frontend_reasoning fr ON fr.rn = pd.rn
  LEFT JOIN frontend_mutation fm ON fm.ordinal = fr.frontend_ordinal
  LEFT JOIN timeline_entities te ON te.entity_id = fr.frontend_message_id
 ORDER BY pd.rn
 LIMIT 80;
