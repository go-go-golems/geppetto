// Real provider EventEmitter + session.next().runAsync() smoke example.
//
// Run from the repository root with:
//
//   go run ./cmd/examples/geppetto-js-run run \
//     --script examples/js/geppetto/31_event_emitter_run_async.js \
//     --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
//     --profile default \
//     --timeout-ms 120000
//
// The script returns a Promise. The example runner waits for returned promises,
// so this can exercise live EventEmitter delivery through session.next().runAsync(...).

(async () => {
  const gp = require("geppetto");
  const EventEmitter = require("events");

  const cfg = globalThis.GEPPETTO_EXAMPLE || {};
  const profile = cfg.profile || "default";
  const timeoutMs = cfg.timeoutMs || 120000;

  const settings = gp.inferenceProfiles.resolve(profile);
  const settingsSnapshot = settings.toJSON();

  const events = new EventEmitter();
  const seen = [];

  events.on("event", ev => {
    seen.push(ev.type);
  });

  events.on("text-delta", ev => {
    if (ev.delta) {
      seen.push("text-delta:" + ev.delta);
    }
  });

  events.on("inference-error", ev => {
    console.error("inference-error", ev.message || ev.error);
  });

  const agent = gp.agent()
    .name("event-emitter-run-async")
    .inference(settings)
    .events(events)
    .runDefaults({ timeoutMs, tags: { example: "event-emitter-run-async" } })
    .build();

  const session = agent.session().id("event-emitter-run-async-smoke").build();
  const handle = session.next()
    .system("You are participating in a deterministic integration smoke test. Follow the requested output format exactly.")
    .user("Reply with exactly this token and no extra words: ASYNC_GEPPETTO")
    .runAsync();

  const result = await handle.promise;
  const text = String(result.text() || "").replace(/\s+/g, " ").trim();
  if (!text) {
    throw new Error("runAsync returned empty text");
  }

  console.log("\n" + JSON.stringify({
    profile,
    registry: settingsSnapshot.provenance && settingsSnapshot.provenance.registrySlug,
    model: settingsSnapshot.chat && settingsSnapshot.chat.engine,
    sessionId: session.id(),
    text,
    eventCount: seen.length,
    eventTypes: Array.from(new Set(seen)),
  }, null, 2));

  return result;
})();
