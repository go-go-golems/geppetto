const gp = require("geppetto");

const resolved = gp.profiles.resolve({
  registrySlug: "team-agent",
  profileSlug: "assistant",
});

assert(resolved.registrySlug === "team-agent", "explicit registry resolve mismatch");
assert(resolved.profileSlug === "assistant", "explicit profile resolve mismatch");
assert(resolved.inferenceSettings.chat.engine === "gpt-5-mini", "team assistant engine mismatch");
assert(resolved.inferenceSettings.chat.api_type === "openai-responses", "team assistant api_type mismatch");

console.log("explicit resolve:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  profileSlug: resolved.profileSlug,
  model: resolved.inferenceSettings.chat.engine,
}));
