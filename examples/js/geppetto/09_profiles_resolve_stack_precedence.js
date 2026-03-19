const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });

assert(resolved.registrySlug === "user-overrides", "assistant should resolve from top-of-stack registry");
assert(resolved.profileSlug === "assistant", "resolved profile slug mismatch");
assert(resolved.inferenceSettings.chat.engine === "gpt-5-nano", "top-of-stack engine override mismatch");
assert(resolved.inferenceSettings.chat.api_type === "openai-responses", "stacked api_type should be inherited");

console.log("resolved:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  profileSlug: resolved.profileSlug,
  model: resolved.inferenceSettings.chat.engine,
}));
