// Phase 1-2 hard-cut example: load a Geppetto registry and resolve a
// Go-owned, read-only InferenceSettings wrapper.
const gp = require("geppetto");

const source = globalThis.GEPPETTO_PHASE123_PROFILE ||
  "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";

const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve("assistant");

const snapshot = settings.toJSON();
if (snapshot.chat.engine !== "gpt-4o-mini") {
  throw new Error(`unexpected model: ${snapshot.chat.engine}`);
}
if (JSON.stringify(snapshot).includes("example-secret")) {
  throw new Error("settings snapshot leaked a raw API key");
}

const clone = settings.clone();
const cloneSnapshot = clone.toJSON();
if (cloneSnapshot.provenance.profileSlug !== "assistant") {
  throw new Error("clone did not preserve provenance");
}

console.log(JSON.stringify({
  profile: snapshot.provenance.profileSlug,
  registry: snapshot.provenance.registrySlug,
  model: snapshot.chat.engine,
  apiKeys: Object.keys(snapshot.api.api_keys || {}),
}, null, 2));

registry.close();
