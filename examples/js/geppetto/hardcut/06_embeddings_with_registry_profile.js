const gp = require("geppetto");

const settings = gp.inferenceProfiles
  .load(globalThis.GEPPETTO_PHASE123_PROFILE)
  .resolve("assistant");
const embedder = gp.embeddings(settings);
const model = embedder.model();

if (model.name !== "text-embedding-3-small") {
  throw new Error(`unexpected embedding model name: ${model.name}`);
}
if (model.dimensions !== 4) {
  throw new Error(`unexpected embedding dimensions: ${model.dimensions}`);
}

JSON.stringify({ ok: true, model }, null, 2);
