const gp = require("geppetto");

const resolved = gp.profiles.resolve({
  profileSlug: "assistant",
  runtimeKeyFallback: "assistant-runtime"
});
assert(
  resolved.runtimeKey === "assistant-runtime",
  "runtimeKeyFallback should set the resolved runtime key"
);
assert(
  resolved.effectiveRuntime.system_prompt === "User override assistant profile.",
  "resolve should return the profile runtime"
);

const legacyAliasIgnored = gp.profiles.resolve({
  profileSlug: "assistant",
  runtimeKey: "legacy-alias-should-not-apply"
});
assert(
  legacyAliasIgnored.runtimeKey === "assistant",
  "legacy runtimeKey alias should no longer affect profiles.resolve"
);

console.log("profiles.resolve runtime key checks: PASS");
