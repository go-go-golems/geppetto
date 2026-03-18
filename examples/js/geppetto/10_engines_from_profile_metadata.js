const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
const fromResolved = gp.engines.fromResolvedProfile(resolved);
const fromProfile = gp.engines.fromProfile({ profileSlug: "assistant" });

assert(fromResolved.metadata.profileSlug === "assistant", "fromResolvedProfile metadata mismatch");
assert(fromProfile.metadata.registrySlug === "user-overrides", "fromProfile metadata mismatch");

console.log("engines:", JSON.stringify({
  fromResolved: fromResolved.metadata,
  fromProfile: fromProfile.metadata,
}));
