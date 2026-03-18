const gp = require("geppetto");

const runtime = gp.runner.resolveRuntime({
  systemPrompt: "Answer in one short line.",
  runtimeKey: "local-demo",
  runtimeFingerprint: "local-demo-fingerprint",
  profileVersion: 1,
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
          runtimeFingerprint: runtimeMeta.runtime_fingerprint,
          profileVersion: runtimeMeta["profile.version"],
        })),
      ],
    });
  }),
  runtime,
  prompt: "say hello",
});

const payload = JSON.parse(out.blocks[0].payload.text);
assert(payload.firstKind === "system", "runner.run should prepend the direct system prompt");
assert(payload.firstText === "Answer in one short line.", "runner.run system prompt mismatch");
assert(payload.runtimeKey === "local-demo", "runner.run should stamp runtime_key");
assert(payload.runtimeFingerprint === "local-demo-fingerprint", "runner.run should stamp runtime_fingerprint");
assert(payload.profileVersion === 1, "runner.run should stamp profile.version");

console.log(JSON.stringify({ ok: true, runtimeKey: payload.runtimeKey }));
