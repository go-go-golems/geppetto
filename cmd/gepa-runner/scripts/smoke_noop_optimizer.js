const plugins = require("geppetto/plugins");

module.exports = plugins.defineOptimizerPlugin({
  apiVersion: plugins.OPTIMIZER_PLUGIN_API_VERSION,
  kind: "optimizer",
  id: "example.smoke_noop",
  name: "Example: Smoke Noop Optimizer",
  create() {
    return {
      dataset() {
        return [
          { id: "a", answer: "ok" },
          { id: "b", answer: "ok" },
        ];
      },
      evaluate(input) {
        const candidate = (input && typeof input === "object" && input.candidate && typeof input.candidate === "object")
          ? input.candidate
          : {};
        const prompt = String(candidate.prompt || "");
        const good = prompt.toLowerCase().includes("ok");
        return {
          score: good ? 1.0 : 0.0,
          output: { promptLength: prompt.length },
          feedback: good ? "prompt contains ok" : "prompt missing ok",
        };
      },
    };
  },
});
