// Multi-turn EventEmitter example for `session.next().runAsync(...)`.
//
// Run from the repository root with:
//
//   go run ./cmd/examples/geppetto-js-run run \
//     --script examples/js/geppetto/33_event_emitter_multiturn_run_async.js \
//     --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
//     --profile default \
//     --timeout-ms 120000
//
// A builder-level EventEmitter is attached once and receives events for both
// asynchronous runs from the same session. The second session.next() call
// continues from the latest session turn without exposing public turn-run input.

(async () => {
  const gp = require("geppetto");
  const EventEmitter = require("events");

  const cfg = globalThis.GEPPETTO_EXAMPLE || {};
  const profile = cfg.profile || "default";
  const timeoutMs = cfg.timeoutMs || 120000;

  function oneLine(s) {
    return String(s || "").replace(/\s+/g, " ").trim();
  }

  const settings = gp.inferenceProfiles.resolve(profile);
  const settingsSnapshot = settings.toJSON();

  const events = new EventEmitter();
  const allEvents = [];
  const deltasByRun = [];
  let currentRun = -1;

  events.on("event", ev => {
    allEvents.push(ev.type);
  });

  events.on("provider-call-started", () => {
    currentRun += 1;
    deltasByRun[currentRun] = [];
  });

  events.on("text-delta", ev => {
    if (currentRun >= 0 && ev.delta) {
      deltasByRun[currentRun].push(ev.delta);
    }
  });

  events.on("inference-error", ev => {
    console.error("inference-error", ev.message || ev.error);
  });

  const agent = gp.agent()
    .name("event-emitter-multiturn-run-async")
    .inference(settings)
    .events(events)
    .runDefaults({ timeoutMs, tags: { example: "event-emitter-multiturn-run-async" } })
    .build();

  const session = agent.session().id("event-emitter-multiturn-smoke").build();
  const system = "You are participating in a deterministic integration smoke test. Follow the requested output format exactly.";

  const result1 = await session.next()
    .system(system)
    .user("Turn 1: Reply with exactly this token and no extra words: ASYNC_ALPHA_GEPPETTO")
    .runAsync()
    .promise;
  const text1 = oneLine(result1.text());
  if (!text1) throw new Error("turn 1 returned empty text");

  const result2 = await session.next()
    .user("Turn 2: What exact token did you return in the previous assistant message? Reply in the form ASYNC_BETA_GEPPETTO:<token> and no extra words.")
    .runAsync()
    .promise;
  const text2 = oneLine(result2.text());
  if (!text2) throw new Error("turn 2 returned empty text");

  console.log(JSON.stringify({
    profile,
    registry: settingsSnapshot.provenance && settingsSnapshot.provenance.registrySlug,
    model: settingsSnapshot.chat && settingsSnapshot.chat.engine,
    sessionId: session.id(),
    turnCount: session.turnCount(),
    turn1: text1,
    turn2: text2,
    totalEvents: allEvents.length,
    eventTypes: Array.from(new Set(allEvents)),
    deltaRuns: deltasByRun.map(parts => ({ deltaCount: parts.length, textFromDeltas: parts.join("") })),
  }, null, 2));
})();
