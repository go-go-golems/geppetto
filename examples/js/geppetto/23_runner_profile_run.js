const gp = require("geppetto");

const runtime = gp.runner.resolveRuntime({
  profile: { profileSlug: "mutable" },
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
          registrySlug: runtimeMeta["profile.registry"],
        })),
      ],
    });
  }),
  runtime,
  prompt: "hello from profile runtime",
});

const payload = JSON.parse(out.blocks[0].payload.text);
assert(payload.firstKind === "system", "profile runtime should materialize the system prompt middleware");
assert(payload.firstText === "Mutable profile baseline.", "profile runtime system prompt mismatch");
assert(payload.runtimeKey === "mutable", "profile runtime should stamp runtime_key");
assert(payload.registrySlug === "user-overrides", "profile runtime should stamp profile.registry");

console.log(JSON.stringify({ ok: true, runtimeKey: payload.runtimeKey, registrySlug: payload.registrySlug }));
