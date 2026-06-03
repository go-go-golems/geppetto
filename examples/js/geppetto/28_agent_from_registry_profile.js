// Phase 4 hard-cut example: resolve registry settings and build a session-capable Agent.
// This intentionally only builds the provider-backed agent; it does not make a live network inference call.
const gp = require("geppetto");

const source = globalThis.GEPPETTO_PHASE123_PROFILE ||
  "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";

const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve("assistant");
const agent = gp.agent()
  .name("registry-agent")
  .inference(settings)
  .build();

if (!agent || typeof agent.session !== "function") {
  throw new Error("registry-backed agent did not expose session()");
}
if (Object.prototype.hasOwnProperty.call(agent, "run")) {
  throw new Error("agent.run must not be public; use agent.session().next().run()");
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
  publicExecution: "agent.session().next().run()",
}, null, 2));

registry.close();
