const gp = require("geppetto");

const resolved = gp.profiles.resolve({
  registrySlug: "team-agent",
  profileSlug: "assistant"
});

assert(resolved.registrySlug === "team-agent", "explicit registry resolve mismatch");
assert(resolved.profileSlug === "assistant", "explicit profile resolve mismatch");

const runtime = resolved.effectiveRuntime || {};
assert(runtime.system_prompt === "You are the team assistant.", "team assistant system prompt mismatch");

const middlewares = runtime.middlewares || [];
assert(middlewares.some((mw) => mw.name === "retry"), "team assistant should include retry middleware");

console.log("explicit resolve:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  profileSlug: resolved.profileSlug,
  middlewareCount: middlewares.length
}));
