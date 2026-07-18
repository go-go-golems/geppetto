const gp = require("geppetto");

const settings = gp.inferenceProfiles
  .load(globalThis.GEPPETTO_PHASE123_PROFILE)
  .resolve("bge-reranker");

const reranker = gp.reranker(settings);

// model() returns the provider/model identity without making a network call.
const model = reranker.model();

if (model.provider !== "llama.cpp") {
  throw new Error(`unexpected reranker provider: ${model.provider}`);
}

JSON.stringify({ ok: true, model }, null, 2);
