const gp = require("geppetto");

const engine = gp.engines.fromProfile("assistant");
assert(engine.name === "profile:user-overrides/assistant", "engine name should include resolved registry/profile");
assert(engine.metadata, "engine metadata should be present");
assert(engine.metadata.profileRegistry === "user-overrides", "engine metadata profileRegistry mismatch");
assert(engine.metadata.profileSlug === "assistant", "engine metadata profileSlug mismatch");
assert(typeof engine.metadata.runtimeFingerprint === "string", "missing runtimeFingerprint metadata");

console.log("engine:", JSON.stringify({
  name: engine.name,
  profileRegistry: engine.metadata.profileRegistry,
  profileSlug: engine.metadata.profileSlug
}));
