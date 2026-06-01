// Real provider multi-turn smoke example for the hard-cut Geppetto JS API.
//
// Run from the repository root with:
//
//   go run ./cmd/examples/geppetto-js-run run \
//     --script examples/js/geppetto/30_real_provider_multiturn.js \
//     --profile-registries "$HOME/.config/pinocchio/profiles.yaml" \
//     --profile default \
//     --timeout-ms 120000
//
// This performs two real provider calls. The second explicit Turn includes the
// first user message and first assistant reply, so the provider receives real
// multi-turn conversational context through the new Go-owned Turn wrapper API.

const gp = require("geppetto");

const cfg = globalThis.GEPPETTO_EXAMPLE || {};
const profile = cfg.profile || "default";
const timeoutMs = cfg.timeoutMs || 120000;

function oneLine(s) {
  return String(s || "").replace(/\s+/g, " ").trim();
}

const settings = gp.inferenceProfiles.resolve(profile);
const settingsSnapshot = settings.toJSON();

const agent = gp.agent()
  .name("real-provider-multiturn")
  .inference(settings)
  .runDefaults({ timeoutMs, tags: { example: "real-provider-multiturn" } })
  .build();

const system = "You are participating in a deterministic integration smoke test. Follow the requested output format exactly.";

const turn1 = gp.turn()
  .system(system)
  .user("Turn 1: Reply with exactly this token and no extra words: ALPHA_GEPPETTO")
  .build();

const result1 = agent.run(turn1);
const text1 = oneLine(result1.text());
if (!text1) {
  throw new Error("turn 1 returned empty text");
}

const turn2 = gp.turn()
  .system(system)
  .user("Turn 1: Reply with exactly this token and no extra words: ALPHA_GEPPETTO")
  .assistant(text1)
  .user("Turn 2: What exact token did you return in the previous assistant message? Reply in the form BETA_GEPPETTO:<token> and no extra words.")
  .build();

const result2 = agent.run(turn2);
const text2 = oneLine(result2.text());
if (!text2) {
  throw new Error("turn 2 returned empty text");
}

console.log(JSON.stringify({
  profile,
  registry: settingsSnapshot.provenance && settingsSnapshot.provenance.registrySlug,
  model: settingsSnapshot.chat && settingsSnapshot.chat.engine,
  turn1: {
    text: text1,
    inputBlocks: result1.inputTurn().toJSON().blocks.length,
    effectiveBlocks: result1.effectiveTurn().toJSON().blocks.length,
    outputBlocks: result1.outputTurn().toJSON().blocks.length,
  },
  turn2: {
    text: text2,
    inputBlocks: result2.inputTurn().toJSON().blocks.length,
    effectiveBlocks: result2.effectiveTurn().toJSON().blocks.length,
    outputBlocks: result2.outputTurn().toJSON().blocks.length,
  },
}, null, 2));
