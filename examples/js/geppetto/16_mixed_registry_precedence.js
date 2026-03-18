const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
assert(
  resolved.registrySlug === "workspace-db",
  "assistant should resolve from top sqlite registry in mixed stack"
);
assert(resolved.effectiveRuntime.system_prompt === "You are the workspace assistant profile.", "runtime payload mismatch");

console.log("mixed precedence:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  runtimeKey: resolved.runtimeKey
}));
