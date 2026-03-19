const gp = require("geppetto");

const resolved = gp.profiles.resolve({ profileSlug: "assistant" });
assert(
  resolved.registrySlug === "workspace-db",
  "assistant should resolve from top sqlite registry in mixed stack",
);
assert(
  resolved.inferenceSettings.chat.engine === "gpt-5-mini",
  "mixed-stack engine settings mismatch",
);

console.log("mixed precedence:", JSON.stringify({
  registrySlug: resolved.registrySlug,
  model: resolved.inferenceSettings.chat.engine,
}));
