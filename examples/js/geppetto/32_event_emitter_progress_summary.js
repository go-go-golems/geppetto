// EventEmitter progress summary example for `agent.runAsync(...)`.
//
// Run from the repository root with:
//
//   go run ./cmd/examples/geppetto-js-run run \
//     --script examples/js/geppetto/32_event_emitter_progress_summary.js \
//     --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
//     --profile default \
//     --timeout-ms 120000
//
// The example attaches one EventEmitter at agent-builder time and records both
// the general "event" channel and selected type-specific channels. Some
// providers emit token/text deltas; others only emit lifecycle/final events.

(async () => {
  const gp = require("geppetto");
  const EventEmitter = require("events");

  const cfg = globalThis.GEPPETTO_EXAMPLE || {};
  const profile = cfg.profile || "default";
  const timeoutMs = cfg.timeoutMs || 120000;

  const settings = gp.inferenceProfiles.resolve(profile);
  const settingsSnapshot = settings.toJSON();

  const events = new EventEmitter();
  const counts = Object.create(null);
  const lifecycle = [];
  const textDeltas = [];
  const reasoningDeltas = [];
  const errors = [];

  events.on("event", ev => {
    counts[ev.type] = (counts[ev.type] || 0) + 1;
  });

  events.on("provider-call-started", ev => {
    lifecycle.push({ type: ev.type, inferenceId: ev.inferenceId || null });
  });

  events.on("provider-call-finished", ev => {
    lifecycle.push({ type: ev.type, inferenceId: ev.inferenceId || null });
  });

  events.on("text-delta", ev => {
    if (ev.delta) textDeltas.push(ev.delta);
  });

  events.on("reasoning-delta", ev => {
    if (ev.delta) reasoningDeltas.push(ev.delta);
  });

  events.on("inference-error", ev => {
    errors.push(ev.message || ev.error || String(ev));
  });

  const agent = gp.agent()
    .name("event-emitter-progress-summary")
    .inference(settings)
    .events(events)
    .runDefaults({ timeoutMs, tags: { example: "event-emitter-progress-summary" } })
    .build();

  const turn = gp.turn()
    .system("Answer with one short sentence.")
    .user("Write a six-word sentence about reliable JavaScript event streams.")
    .build();

  const handle = agent.runAsync(turn);
  const result = await handle.promise;

  console.log(JSON.stringify({
    profile,
    registry: settingsSnapshot.provenance && settingsSnapshot.provenance.registrySlug,
    model: settingsSnapshot.chat && settingsSnapshot.chat.engine,
    finalText: result.text(),
    eventCounts: counts,
    lifecycle,
    textDeltaCount: textDeltas.length,
    textFromDeltas: textDeltas.join(""),
    reasoningDeltaCount: reasoningDeltas.length,
    errors,
  }, null, 2));
})();
