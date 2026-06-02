// Multi-turn EventEmitter example for `agent.runAsync(...)`.
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
// asynchronous runs from the same agent. The second turn explicitly continues
// from the first output turn, preserving the hard-cut explicit-turn model
// without hidden agent state.

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

  const system = "You are participating in a deterministic integration smoke test. Follow the requested output format exactly.";

  const turn1 = gp.turn()
    .system(system)
    .user("Turn 1: Reply with exactly this token and no extra words: ASYNC_ALPHA_GEPPETTO")
    .build();

  const result1 = await agent.runAsync(turn1).promise;
  const text1 = oneLine(result1.text());
  if (!text1) throw new Error("turn 1 returned empty text");

  const turn2 = gp.turn(result1.outputTurn())
    .user("Turn 2: What exact token did you return in the previous assistant message? Reply in the form ASYNC_BETA_GEPPETTO:<token> and no extra words.")
    .build();

  const result2 = await agent.runAsync(turn2).promise;
  const text2 = oneLine(result2.text());
  if (!text2) throw new Error("turn 2 returned empty text");

  console.log(JSON.stringify({
    profile,
    registry: settingsSnapshot.provenance && settingsSnapshot.provenance.registrySlug,
    model: settingsSnapshot.chat && settingsSnapshot.chat.engine,
    turn1: text1,
    turn2: text2,
    totalEvents: allEvents.length,
    eventTypes: Array.from(new Set(allEvents)),
    deltaRuns: deltasByRun.map(parts => ({ deltaCount: parts.length, textFromDeltas: parts.join("") })),
  }, null, 2));
})();
