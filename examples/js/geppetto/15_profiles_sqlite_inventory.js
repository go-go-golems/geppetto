const gp = require("geppetto");

const registries = gp.profiles.listRegistries();
assert(registries.length === 1, "expected single sqlite-backed registry");
assert(registries[0].slug === "workspace-db", "expected workspace-db registry");

const profiles = gp.profiles.listProfiles("workspace-db");
assert(Array.isArray(profiles), "listProfiles should return an array");
assert(profiles.length === 2, "expected two seeded sqlite profiles");

const slugs = profiles.map((p) => p.slug).sort();
assert(
  JSON.stringify(slugs) === JSON.stringify(["assistant", "default"]),
  `unexpected sqlite profile slugs: ${JSON.stringify(slugs)}`
);

const resolvedDefault = gp.profiles.resolve({ registrySlug: "workspace-db" });
assert(resolvedDefault.profileSlug === "default", "resolve() should use seeded default profile");

const assistant = gp.profiles.getProfile("assistant", "workspace-db");
assert(assistant.slug === "assistant", "getProfile should return assistant");
assert(
  assistant.runtime && assistant.runtime.system_prompt === "You are the workspace assistant profile.",
  "assistant runtime should match seeded sqlite registry data"
);

console.log("sqlite inventory checks: PASS");
