const gp = require("geppetto");

const registries = gp.profiles.listRegistries();
assert(Array.isArray(registries), "listRegistries should return an array");
assert(registries.length >= 3, "expected stacked YAML registries");

const slugs = registries.map((r) => r.slug).sort();
assert(slugs.includes("provider-openai"), "missing provider-openai registry");
assert(slugs.includes("team-agent"), "missing team-agent registry");
assert(slugs.includes("user-overrides"), "missing user-overrides registry");

const defaultRegistry = gp.profiles.getRegistry();
assert(defaultRegistry.slug === "user-overrides", "top-of-stack registry should be user-overrides");

const teamProfiles = gp.profiles.listProfiles("team-agent");
const teamProfileSlugs = teamProfiles.map((p) => p.slug);
assert(teamProfileSlugs.includes("assistant"), "team-agent/assistant missing");
assert(teamProfileSlugs.includes("analyst"), "team-agent/analyst missing");

console.log("registries:", JSON.stringify(slugs));
console.log("team profiles:", JSON.stringify(teamProfileSlugs.sort()));
