const gp = require("geppetto");

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

const resolved = gp.profiles.resolve({ profileSlug: "mutable" });
const runtime = gp.runner.resolveRuntime({
  systemPrompt: "App-owned prompt for the resolved-profile session example.",
  runtimeKey: "resolved-profile-demo",
  metadata: {
    profileSlug: resolved.profileSlug,
    profileRegistry: resolved.registrySlug,
    model: resolved.inferenceSettings.chat.engine,
  },
});

const prepared = gp.runner.prepare({
  engine: gp.engines.fromFunction((turn) => {
    const runtimeMeta = turn.metadata.runtime || {};
    return gp.turns.newTurn({
      blocks: [
        gp.turns.newAssistantBlock(JSON.stringify({
          firstKind: turn.blocks[0].kind,
          firstText: turn.blocks[0].payload.text,
          runtimeKey: runtimeMeta.runtime_key,
          model: resolved.inferenceSettings.chat.engine,
          profileSlug: resolved.profileSlug,
          registrySlug: resolved.registrySlug,
        })),
      ],
    });
  }),
  runtime,
  prompt: "hello from the resolved-profile example",
});

assert(prepared.session.turnCount() === 1, "prepared session should contain the seed turn");

const out = prepared.run();

const payload = JSON.parse(out.blocks[0].payload.text);
assert(payload.firstKind === "system", "session should apply the explicit app-owned system prompt");
assert(payload.firstText === "App-owned prompt for the resolved-profile session example.", "system prompt mismatch");
assert(payload.runtimeKey === "resolved-profile-demo", "runtime metadata should stay app-owned");
assert(payload.model === "gpt-4.1-mini", "resolved profile should expose engine settings");
assert(payload.profileSlug === "mutable", "resolved profile slug mismatch");
assert(payload.registrySlug === "user-overrides", "resolved profile registry mismatch");

console.log(JSON.stringify({
  ok: true,
  runtimeKey: payload.runtimeKey,
  model: payload.model,
}));
