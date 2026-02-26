const gp = require("geppetto");

const resolved = gp.profiles.resolve({
  registrySlug: "team-agent",
  profileSlug: "assistant"
});

assert(resolved.registrySlug === "team-agent", "explicit registry resolve mismatch");
assert(resolved.profileSlug === "assistant", "explicit profile resolve mismatch");

const runtime = resolved.effectiveRuntime || {};
const patch = runtime.step_settings_patch || {};
const aiChat = patch["ai-chat"] || {};
assert(aiChat["ai-engine"] === "gpt-4.1-mini", "team assistant model should be gpt-4.1-mini");
assert(aiChat["ai-api-type"] === "openai", "team assistant should inherit openai api type from provider layer");

const middlewares = runtime.middlewares || [];
assert(middlewares.some((mw) => mw.name === "retry"), "team assistant should include retry middleware");

console.log("explicit resolve:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  profileSlug: resolved.profileSlug,
  middlewareCount: middlewares.length
}));
