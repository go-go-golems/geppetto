const gp = require("geppetto");

const engine = gp.engines.fromFunction((turn) => {
  turn.blocks.push(gp.turns.newAssistantBlock("ok"));
  return turn;
});

const session = gp
  .createBuilder()
  .withEngine(engine)
  .useGoMiddleware("systemPrompt", { prompt: "SYSTEM" })
  .useMiddleware(
    gp.middlewares.fromJS((turn, next) => {
      const out = next(turn);
      out.metadata = out.metadata || {};
      out.metadata.trace_id = "js-mw";
      return out;
    }, "trace")
  )
  .buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("ping")] }));
const out = session.run();

assert(out.blocks[0].kind === "system", "go middleware did not inject system block");
assert(out.blocks[0].payload.text === "SYSTEM", "system prompt text mismatch");
assert(out.metadata.trace_id === "js-mw", "js middleware metadata missing");

console.log("roles:", out.blocks.map((b) => b.kind));
console.log("trace_id:", out.metadata.trace_id);

