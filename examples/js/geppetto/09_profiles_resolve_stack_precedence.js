const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });

assert(resolved.registrySlug === "user-overrides", "assistant should resolve from top-of-stack registry");
assert(resolved.profileSlug === "assistant", "resolved profile slug mismatch");
assert(resolved.runtimeKey === "assistant", "default runtime key should match profile slug");
assert(typeof resolved.runtimeFingerprint === "string" && resolved.runtimeFingerprint.length > 8, "missing runtime fingerprint");

const runtime = resolved.effectiveRuntime || {};
assert(runtime.system_prompt === "User override assistant profile.", "resolved system prompt mismatch");
assert(Array.isArray(runtime.tools), "resolved tools payload missing");
assert(runtime.tools.includes("go_concat"), "tool inheritance mismatch");

console.log("resolved:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  profileSlug: resolved.profileSlug,
  runtimeKey: resolved.runtimeKey
}));
