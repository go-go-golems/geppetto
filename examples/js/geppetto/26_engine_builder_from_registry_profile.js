// Phase 3 hard-cut example: build an Engine from registry-resolved
// InferenceSettings. This constructs a provider engine but does not make a
// network inference call.
const gp = require("geppetto");

const source = globalThis.GEPPETTO_PHASE123_PROFILE ||
  "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";

const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve({ profile: "assistant" });

let rejectedPlainObject = false;
try {
  gp.engine().inference({ chat: { engine: "gpt-4o-mini" } }).build();
} catch (err) {
  rejectedPlainObject = /InferenceSettings wrapper/i.test(String(err));
}
if (!rejectedPlainObject) {
  throw new Error("engine().inference(...) accepted a plain JS object");
}

const engine = gp.engine().inference(settings).build();
if (engine.metadata.profileSlug !== "assistant") {
  throw new Error("engine did not preserve profile provenance");
}
if (!engine.modelInfo || engine.modelInfo.contextWindow !== 128000) {
  throw new Error("engine did not expose modelInfo from registry settings");
}

console.log(JSON.stringify({
  engineName: engine.name,
  registry: engine.metadata.registrySlug,
  profile: engine.metadata.profileSlug,
  contextWindow: engine.modelInfo.contextWindow,
}, null, 2));

registry.close();
