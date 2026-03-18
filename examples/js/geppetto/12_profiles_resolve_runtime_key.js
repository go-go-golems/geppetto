const gp = require("geppetto");

const resolved = gp.profiles.resolve({
  profileSlug: "assistant",
});

assert(resolved.profileSlug === "assistant", "profiles.resolve should keep the selected profile slug");
assert(resolved.inferenceSettings.chat.engine === "gpt-5-nano", "profiles.resolve should expose engine settings");

const ignoredRuntimeAlias = gp.profiles.resolve({
  profileSlug: "assistant",
  runtimeKey: "legacy-alias-should-not-apply",
});

assert(ignoredRuntimeAlias.profileSlug === "assistant", "legacy runtimeKey input should no longer affect profiles.resolve");
assert(ignoredRuntimeAlias.inferenceSettings.chat.engine === "gpt-5-nano", "legacy runtimeKey input should not affect inference settings");

console.log("profiles.resolve hard-cut checks: PASS");
