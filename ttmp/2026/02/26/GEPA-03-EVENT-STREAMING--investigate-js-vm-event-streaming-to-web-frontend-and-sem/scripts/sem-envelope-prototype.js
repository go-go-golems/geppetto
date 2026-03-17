#!/usr/bin/env node

/**
 * Minimal SEM envelope prototype used for this ticket analysis.
 * It demonstrates how JS VM script events could be normalized
 * before they enter the pinocchio/go-go-os pipeline.
 */

function makeSemEnvelope(eventType, data, meta = {}) {
  const now = new Date().toISOString();
  return {
    sem: true,
    event: {
      id: `${eventType}:${Math.random().toString(36).slice(2, 10)}`,
      type: eventType,
      ts: now,
      data,
      metadata: meta,
    },
  };
}

function validateSemEnvelope(envelope) {
  if (!envelope || envelope.sem !== true) return { ok: false, reason: 'missing sem=true' };
  if (!envelope.event || typeof envelope.event !== 'object') return { ok: false, reason: 'missing event object' };
  if (typeof envelope.event.type !== 'string' || envelope.event.type.length === 0) {
    return { ok: false, reason: 'missing event.type' };
  }
  if (typeof envelope.event.id !== 'string' || envelope.event.id.length === 0) {
    return { ok: false, reason: 'missing event.id' };
  }
  return { ok: true, reason: 'valid' };
}

function classifyFamily(type) {
  if (type.startsWith('llm.')) return 'llm';
  if (type.startsWith('tool.')) return 'tool';
  if (type.startsWith('timeline.')) return 'timeline';
  if (type.startsWith('ws.')) return 'ws';
  if (type.startsWith('hypercard.')) return 'hypercard';
  return 'other';
}

const samples = [
  makeSemEnvelope('gepa.script.progress', { step: 1, total: 8, label: 'collect-candidates' }, { run_id: 'run-001' }),
  makeSemEnvelope('timeline.upsert', {
    conv_id: 'conv-demo',
    version: 9,
    entity: {
      id: 'entity-gepa-progress-1',
      kind: 'gepa.progress',
      turn_id: 'turn-1',
      created_at_unix_ms: Date.now(),
      updated_at_unix_ms: Date.now(),
      role: 'system',
      title: 'GEPA Progress',
      props: { phase: 'mutation', generation: 3, candidate_index: 12 },
      stream_state: 'running',
    },
  }),
  { sem: false, event: { id: 'bad-1', type: 'gepa.bad' } },
  { sem: true, event: { id: '', type: '' } },
];

for (const sample of samples) {
  const verdict = validateSemEnvelope(sample);
  const type = sample?.event?.type || '<missing>';
  const family = classifyFamily(type);
  process.stdout.write(
    `${verdict.ok ? 'OK' : 'FAIL'} | type=${type} | family=${family} | reason=${verdict.reason}\n`
  );
}
