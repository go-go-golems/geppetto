const gp = require("geppetto");
const source = globalThis.GEPPETTO_PHASE123_PROFILE || "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";
const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve("assistant");
const engine = gp.engine().inference(settings).build();
console.log(JSON.stringify({ engine: engine.name, profile: engine.metadata.profileSlug }, null, 2));
registry.close();
