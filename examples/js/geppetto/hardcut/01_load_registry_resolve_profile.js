const gp = require("geppetto");
const source = globalThis.GEPPETTO_PHASE123_PROFILE || "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";
const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve("assistant");
console.log(JSON.stringify({ model: settings.toJSON().chat.engine, profile: settings.toJSON().provenance.profileSlug }, null, 2));
registry.close();
