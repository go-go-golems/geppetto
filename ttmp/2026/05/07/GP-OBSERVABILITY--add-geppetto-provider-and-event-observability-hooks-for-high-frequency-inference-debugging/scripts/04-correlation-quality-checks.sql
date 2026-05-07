-- 04-correlation-quality-checks.sql
-- Purpose: Summarize how well provider/geppetto/backend/frontend reasoning
-- deltas line up in one browser-driven recording.

.headers on
.mode column

WITH
provider_delta AS (
  SELECT row_number() OVER (ORDER BY record_id) AS rn,
         json_extract(object_json, '$.delta') AS provider_delta
    FROM geppetto_records
   WHERE stage = 'provider_normalize_delta'
     AND event_type = 'response.reasoning_summary_text.delta'
),
geppetto_delta AS (
  SELECT row_number() OVER (ORDER BY record_id) AS rn,
         json_extract(event_json, '$.delta') AS geppetto_delta
    FROM geppetto_records
   WHERE stage = 'geppetto_publish_done'
     AND event_type = 'partial-thinking'
),
backend_reasoning AS (
  SELECT row_number() OVER (ORDER BY CAST(br.ordinal AS INTEGER)) AS rn,
         json_extract(bpue.payload_json, '$.chunk') AS backend_chunk
    FROM backend_pipeline bp
    JOIN backend_records br ON br.id = bp.record_id
    JOIN backend_pipeline_ui_events bpue ON bpue.record_id = br.id
   WHERE bp.event_name = 'ChatReasoningDelta'
     AND bpue.source = 'uiEvents'
),
frontend_reasoning AS (
  SELECT row_number() OVER (ORDER BY CAST(fr.ordinal AS INTEGER)) AS rn,
         json_extract(fpf.frame_json, '$.payload.chunk') AS frontend_chunk
    FROM frontend_parsed_frames fpf
    JOIN frontend_records fr ON fr.id = fpf.record_id
   WHERE fpf.name = 'ChatReasoningAppended'
)
SELECT 'provider_to_frontend' AS comparison,
       COUNT(*) AS pairs,
       SUM(CASE WHEN pd.provider_delta = fr.frontend_chunk THEN 1 ELSE 0 END) AS exact_matches,
       SUM(CASE WHEN pd.provider_delta != fr.frontend_chunk THEN 1 ELSE 0 END) AS mismatches
  FROM provider_delta pd
  JOIN frontend_reasoning fr ON fr.rn = pd.rn
UNION ALL
SELECT 'geppetto_to_frontend',
       COUNT(*),
       SUM(CASE WHEN gd.geppetto_delta = fr.frontend_chunk THEN 1 ELSE 0 END),
       SUM(CASE WHEN gd.geppetto_delta != fr.frontend_chunk THEN 1 ELSE 0 END)
  FROM geppetto_delta gd
  JOIN frontend_reasoning fr ON fr.rn = gd.rn
UNION ALL
SELECT 'backend_to_frontend',
       COUNT(*),
       SUM(CASE WHEN br.backend_chunk = fr.frontend_chunk THEN 1 ELSE 0 END),
       SUM(CASE WHEN br.backend_chunk != fr.frontend_chunk THEN 1 ELSE 0 END)
  FROM backend_reasoning br
  JOIN frontend_reasoning fr ON fr.rn = br.rn;

WITH
provider_delta AS (
  SELECT row_number() OVER (ORDER BY record_id) AS rn,
         record_id,
         json_extract(object_json, '$.delta') AS provider_delta
    FROM geppetto_records
   WHERE stage = 'provider_normalize_delta'
     AND event_type = 'response.reasoning_summary_text.delta'
),
frontend_reasoning AS (
  SELECT row_number() OVER (ORDER BY CAST(fr.ordinal AS INTEGER)) AS rn,
         fr.ordinal,
         json_extract(fpf.frame_json, '$.payload.chunk') AS frontend_chunk
    FROM frontend_parsed_frames fpf
    JOIN frontend_records fr ON fr.id = fpf.record_id
   WHERE fpf.name = 'ChatReasoningAppended'
)
SELECT pd.rn,
       pd.record_id AS provider_record_id,
       quote(pd.provider_delta) AS provider_delta,
       fr.ordinal AS frontend_ordinal,
       quote(fr.frontend_chunk) AS frontend_chunk
  FROM provider_delta pd
  JOIN frontend_reasoning fr ON fr.rn = pd.rn
 WHERE pd.provider_delta != fr.frontend_chunk
 ORDER BY pd.rn;
