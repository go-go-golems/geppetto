const gp = require("geppetto");
const plugins = require("geppetto/plugins");

function resolveAssistantText(out) {
  const blocks = (out && Array.isArray(out.blocks)) ? out.blocks : [];
  return blocks
    .filter((b) => b && b.payload && typeof b.payload.text === "string")
    .filter((b) => b.kind === gp.consts.BlockKind.LLM_TEXT || b.kind === "assistant")
    .map((b) => (b.payload.text || "").trim())
    .join("\n")
    .trim();
}

function normalizeEngineOptions(ctx, options) {
  const merged = (options && typeof options === "object") ? options : {};
  const ctxOpts = (ctx && typeof ctx === "object") ? ctx : {};
  const engineOptions = (merged.engineOptions && typeof merged.engineOptions === "object")
    ? merged.engineOptions
    : ((ctxOpts.engineOptions && typeof ctxOpts.engineOptions === "object") ? ctxOpts.engineOptions : null);

  return {
    profile: (typeof merged.profile === "string" && merged.profile.trim())
      ? merged.profile.trim()
      : ((typeof ctxOpts.profile === "string" && ctxOpts.profile.trim()) ? ctxOpts.profile.trim() : ""),
    engineOptions,
  };
}

module.exports = plugins.defineOptimizerPlugin({
  apiVersion: plugins.OPTIMIZER_PLUGIN_API_VERSION,
  kind: "optimizer",
  id: "example.toy_math",
  name: "Example: Toy math accuracy",

  create(ctx) {
    const dataset = [
      { question: "2+2", answer: "4" },
      { question: "10-3", answer: "7" },
      { question: "6*7", answer: "42" },
      { question: "12/4", answer: "3" },
      { question: "9+8", answer: "17" },
      { question: "100-25", answer: "75" },
    ];

    function datasetFn() {
      return dataset;
    }

    function evaluate(input, options) {
      const inObj = (input && typeof input === "object") ? input : {};
      const candidate = (inObj.candidate && typeof inObj.candidate === "object") ? inObj.candidate : {};
      const example = (inObj.example && typeof inObj.example === "object") ? inObj.example : {};

      const instruction = (typeof candidate.prompt === "string" && candidate.prompt.trim())
        ? candidate.prompt.trim()
        : "Answer the question. Respond with only the final answer.";

      const prompt = `${instruction}\n\nQuestion: ${String(example.question || "")}\nFinal answer:`;

      const resolved = normalizeEngineOptions(ctx, options);
      const engine = (resolved.engineOptions && Object.keys(resolved.engineOptions).length > 0)
        ? gp.engines.fromConfig(resolved.engineOptions)
        : gp.engines.fromProfile(resolved.profile || "", {});

      const builder = gp.createBuilder().withEngine(engine);
      const session = builder.buildSession();

      const seed = gp.turns.newTurn({
        blocks: [
          gp.turns.newUserBlock(prompt),
        ],
      });

      const out = session.run(seed, {});
      const text = resolveAssistantText(out);

      const expected = String(example.answer || "").trim();
      const got = String(text || "").trim();

      const ok = expected !== "" && got === expected;
      return {
        score: ok ? 1.0 : 0.0,
        output: { text: got },
        feedback: ok ? "Correct." : `Expected "${expected}" but got "${got}".`,
      };
    }

    return {
      dataset: datasetFn,
      evaluate,
    };
  },
});
