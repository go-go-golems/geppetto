const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
assert(resolved.registrySlug === "user-overrides", "resolved registry mismatch");
assert(resolved.profileSlug === "assistant", "resolved profile mismatch");
assert(typeof resolved.runtimeFingerprint === "string", "missing runtimeFingerprint metadata");

console.log("resolved:", JSON.stringify({
  profileRegistry: resolved.registrySlug,
  profileSlug: resolved.profileSlug,
  runtimeFingerprint: resolved.runtimeFingerprint
}));
