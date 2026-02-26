const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });

assert(resolved.registrySlug === "user-overrides", "assistant should resolve from top-of-stack registry");
assert(resolved.profileSlug === "assistant", "resolved profile slug mismatch");
assert(resolved.runtimeKey === "assistant", "default runtime key should match profile slug");
assert(typeof resolved.runtimeFingerprint === "string" && resolved.runtimeFingerprint.length > 8, "missing runtime fingerprint");

const runtime = resolved.effectiveRuntime || {};
assert(runtime.system_prompt === "User override assistant profile.", "resolved system prompt mismatch");

const patch = runtime.step_settings_patch || {};
const aiChat = patch["ai-chat"] || {};
assert(aiChat["ai-engine"] === "gpt-4.1-nano", "ai-engine should come from top profile layer");
assert(aiChat["ai-api-type"] === "openai", "ai-api-type should come from provider layer");

console.log("resolved:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  profileSlug: resolved.profileSlug,
  runtimeKey: resolved.runtimeKey
}));
