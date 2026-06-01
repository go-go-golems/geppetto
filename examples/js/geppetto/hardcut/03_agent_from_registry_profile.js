const gp = require("geppetto");
const source = globalThis.GEPPETTO_PHASE123_PROFILE || "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";
const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve("assistant");
const agent = gp.agent().name("assistant").inference(settings).build();
if (typeof agent.run !== "function") throw new Error("agent.run missing");
console.log(JSON.stringify({ agent: "assistant", profile: settings.toJSON().provenance.profileSlug }, null, 2));
registry.close();
