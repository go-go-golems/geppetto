const gp = require("geppetto");

const sources = [
  "examples/js/geppetto/profiles/10-provider-openai.yaml",
  "examples/js/geppetto/profiles/20-team-agent.yaml",
  "examples/js/geppetto/profiles/30-user-overrides.yaml",
];

const connected = gp.profiles.connectStack(sources);
assert(Array.isArray(connected.sources), "connectStack should return source list");
assert(connected.sources.length === 3, "connectStack source count mismatch");

const active = gp.profiles.getConnectedSources();
assert(active.length === 3, "getConnectedSources should return connected source stack");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
assert(resolved.registrySlug === "user-overrides", "runtime connectStack should honor top-of-stack precedence");

gp.profiles.disconnectStack();
assert(gp.profiles.getConnectedSources().length === 0, "disconnectStack should clear runtime stack");

let threw = false;
try {
  gp.profiles.listRegistries();
} catch (e) {
  threw = /configured profile registry/i.test(String(e));
}
assert(threw, "profiles API should require configured registry after disconnect");

console.log("connected sources:", JSON.stringify(connected.sources));
console.log("resolved assistant registry:", resolved.registrySlug);
