const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "mutable" });
const runtime = gp.runner.resolveRuntime({
  systemPrompt: "App-owned prompt for runner profile composition.",
  runtimeKey: resolved.profileSlug,
  metadata: {
    profileSlug: resolved.profileSlug,
    profileRegistry: resolved.registrySlug,
    model: resolved.inferenceSettings.chat.engine,
  },
});

const out = gp.runner.run({
  engine: gp.engines.fromFunction((turn) => {
    const runtimeMeta = turn.metadata.runtime || {};
    return gp.turns.newTurn({
      blocks: [
        gp.turns.newAssistantBlock(JSON.stringify({
          firstKind: turn.blocks[0].kind,
          firstText: turn.blocks[0].payload.text,
          runtimeKey: runtimeMeta.runtime_key,
          profileSlug: resolved.profileSlug,
          registrySlug: resolved.registrySlug,
          model: resolved.inferenceSettings.chat.engine,
        })),
      ],
    });
  }),
  runtime,
  prompt: "hello from explicit runner runtime",
});

const payload = JSON.parse(out.blocks[0].payload.text);
assert(payload.firstKind === "system", "runner runtime should materialize the app-owned system prompt");
assert(payload.firstText === "App-owned prompt for runner profile composition.", "runner runtime system prompt mismatch");
assert(payload.runtimeKey === "mutable", "runner runtime should stamp the explicit runtime key");
assert(payload.registrySlug === "user-overrides", "resolved profile registry mismatch");
assert(payload.model === "gpt-4.1-mini", "resolved profile engine settings mismatch");

console.log(JSON.stringify({
  ok: true,
  runtimeKey: payload.runtimeKey,
  registrySlug: payload.registrySlug,
  model: payload.model,
}));
