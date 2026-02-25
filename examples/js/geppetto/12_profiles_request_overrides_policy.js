const gp = require("geppetto");

const resolved = gp.profiles.resolve({
  profileSlug: "mutable",
  requestOverrides: {
    system_prompt: "One-shot system override"
  }
});
assert(
  resolved.effectiveRuntime.system_prompt === "One-shot system override",
  "system_prompt override should apply for allowed key"
);

let denied = false;
try {
  gp.profiles.resolve({
    profileSlug: "mutable",
    requestOverrides: {
      middlewares: []
    }
  });
} catch (e) {
  denied = /not allowed|policy violation|override key/i.test(String(e));
}
assert(denied, "middlewares override should be rejected by profile policy");

console.log("override policy checks: PASS");
