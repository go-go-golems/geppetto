const gp = require("geppetto");

let middlewareCtx = null;
let toolCtx = null;
let hookPayload = null;

const tools = gp.tools.createRegistry();
tools.register({
  name: "ctx_echo",
  description: "echo tool for context validation",
  handler: ({ text }, ctx) => {
    toolCtx = ctx;
    return { echoed: text };
  }
});

const engine = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === gp.consts.BlockKind.TOOL_USE);
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("ctx-call-1", "ctx_echo", { text: "abc" }));
    return turn;
  }
  turn.blocks.push(gp.turns.newAssistantBlock("context-ok"));
  return turn;
});

const session = gp
  .createBuilder()
  .withEngine(engine)
  .useMiddleware(
    gp.middlewares.fromJS((turn, next, ctx) => {
      middlewareCtx = ctx;
      return next(turn);
    }, "ctx-mw")
  )
  .withTools(tools, {
    enabled: true,
    maxIterations: 3,
    toolChoice: gp.consts.ToolChoice.AUTO,
    toolErrorHandling: gp.consts.ToolErrorHandling.RETRY
  })
  .withToolHooks({
    beforeToolCall: (payload) => {
      hookPayload = payload;
    }
  })
  .buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("run context")] }));
const out = session.run(undefined, {
  timeoutMs: 2000,
  tags: {
    suite: "gp17",
    script: "02"
  }
});

assert(Array.isArray(out.blocks), "expected blocks");
assert(!!middlewareCtx, "middleware context missing");
assert(!!middlewareCtx.sessionId, "middleware sessionId missing");
assert(!!middlewareCtx.inferenceId, "middleware inferenceId missing");
assert(!!middlewareCtx.deadlineMs, "middleware deadlineMs missing");
assert(middlewareCtx.tags && middlewareCtx.tags.script === "02", "middleware tags missing");

assert(!!toolCtx, "tool context missing");
assert(toolCtx.callId === "ctx-call-1", "tool callId mismatch");
assert(!!toolCtx.sessionId, "tool sessionId missing");
assert(!!toolCtx.inferenceId, "tool inferenceId missing");

assert(!!hookPayload, "hook payload missing");
assert(!!hookPayload.sessionId, "hook sessionId missing");
assert(!!hookPayload.inferenceId, "hook inferenceId missing");
assert(hookPayload.tags && hookPayload.tags.suite === "gp17", "hook tags missing");

console.log("middlewareCtx:", JSON.stringify(middlewareCtx));
console.log("toolCtx:", JSON.stringify(toolCtx));
console.log("hookPayload:", JSON.stringify(hookPayload));
