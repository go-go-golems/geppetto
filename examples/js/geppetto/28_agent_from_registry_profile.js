// Phase 4 hard-cut example: resolve registry settings and build an Agent from
// them. This intentionally only builds the provider-backed agent; it does not
// make a live network inference call.
const gp = require("geppetto");

const source = globalThis.GEPPETTO_PHASE123_PROFILE ||
  "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";

const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve("assistant");
const agent = gp.agent()
  .name("registry-agent")
  .inference(settings)
  .build();

if (!agent || typeof agent.run !== "function") {
  throw new Error("registry-backed agent did not expose run(turn)");
}
if (Object.prototype.hasOwnProperty.call(agent, "ask")) {
  throw new Error("agent.ask must not exist");
}
if (Object.prototype.hasOwnProperty.call(agent, "system")) {
  throw new Error("agent.system must not exist");
}

console.log(JSON.stringify({
  agent: "registry-agent",
  profile: settings.toJSON().provenance.profileSlug,
  model: settings.toJSON().chat.engine,
  explicitTurnRequired: true,
}, null, 2));

registry.close();
