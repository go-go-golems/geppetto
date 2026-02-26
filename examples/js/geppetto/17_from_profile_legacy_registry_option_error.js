const gp = require("geppetto");

let threw = false;
try {
  gp.engines.fromProfile("assistant", { registrySlug: "team-agent" });
} catch (e) {
  threw = /registryslug has been removed/i.test(String(e));
}

assert(threw, "legacy engines.fromProfile registrySlug option should throw migration error");
console.log("legacy registrySlug migration error: PASS");
