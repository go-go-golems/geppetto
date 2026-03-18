const gp = require("geppetto");

function assert(cond, msg) {
  if (!cond) throw new Error(msg);
}

const resolved = gp.profiles.resolve({ profileSlug: "mutable" });

const session = gp.createBuilder()
  .withEngine(gp.engines.fromFunction((turn) => {
    const runtime = turn.metadata.runtime || {};
    return gp.turns.newTurn({
      blocks: [
        gp.turns.newAssistantBlock(JSON.stringify({
          firstKind: turn.blocks[0].kind,
          firstText: turn.blocks[0].payload.text,
          runtimeKey: runtime.runtime_key,
          runtimeFingerprint: runtime.runtime_fingerprint,
          profileSlug: runtime["profile.slug"],
          registrySlug: runtime["profile.registry"],
        })),
      ],
    });
  }))
  .useResolvedProfile(resolved)
  .buildSession();

const out = session.run(gp.turns.newTurn({
  blocks: [gp.turns.newUserBlock("hello from the resolved-profile example")],
}));

const payload = JSON.parse(out.blocks[0].payload.text);
assert(payload.firstKind === "system", "resolvedProfile should materialize the system prompt middleware");
assert(payload.firstText === "Mutable profile baseline.", "resolvedProfile system prompt mismatch");
assert(payload.runtimeKey === "mutable", "resolvedProfile should stamp runtime_key");
assert(typeof payload.runtimeFingerprint === "string" && payload.runtimeFingerprint.length > 8, "resolvedProfile should stamp runtime_fingerprint");
assert(payload.profileSlug === "mutable", "resolvedProfile should stamp profile.slug");
assert(payload.registrySlug === "user-overrides", "resolvedProfile should stamp profile.registry");

console.log(JSON.stringify({
  ok: true,
  runtimeKey: payload.runtimeKey,
  registrySlug: payload.registrySlug,
}));
