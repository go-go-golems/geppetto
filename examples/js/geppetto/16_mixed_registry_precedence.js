const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
assert(
  resolved.registrySlug === "workspace-db",
  "assistant should resolve from top sqlite registry in mixed stack"
);

const engine = gp.engines.fromProfile("assistant");
assert(engine.name === "profile:workspace-db/assistant", "engine should resolve to sqlite registry assistant profile");
assert(engine.metadata.profileRegistry === "workspace-db", "engine metadata should reflect sqlite registry");

console.log("mixed precedence:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  engineName: engine.name
}));
