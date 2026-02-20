const gp = require("geppetto");

let middlewareCtx = null;
let toolCtx = null;
let hookCtx = null;

const tools = gp.tools.createRegistry();
tools.register({
  name: "ctx_echo",
  description: "Echo value and expose context",
  handler: ({ value }, ctx) => {
    toolCtx = ctx;
    return { echoed: value, callId: ctx && ctx.callId };
  }
});

const engine = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === gp.consts.BlockKind.TOOL_USE);
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("ctx-call-1", "ctx_echo", { value: "hello" }));
    return turn;
  }

  turn.blocks.push(gp.turns.newAssistantBlock("done"));
  return turn;
});

const session = gp
  .createBuilder()
  .withEngine(engine)
  .useMiddleware(
    gp.middlewares.fromJS((turn, next, ctx) => {
      middlewareCtx = ctx;
      return next(turn);
    }, "context-demo")
  )
  .withTools(tools, {
    enabled: true,
    maxIterations: 3,
    toolChoice: gp.consts.ToolChoice.AUTO,
    toolErrorHandling: gp.consts.ToolErrorHandling.RETRY,
    maxParallelTools: 1
  })
  .withToolHooks({
    beforeToolCall: (payload) => {
      hookCtx = payload;
    }
  })
  .buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("run context example")] }));
const out = session.run(undefined, {
  timeoutMs: 3000,
  tags: {
    demo: "context-and-constants",
    example: "07"
  }
});

assert(Array.isArray(out.blocks), "expected output blocks");
assert(!!middlewareCtx, "expected middleware context");
assert(!!middlewareCtx.sessionId, "expected middleware sessionId");
assert(!!middlewareCtx.inferenceId, "expected middleware inferenceId");
assert(!!middlewareCtx.deadlineMs, "expected middleware deadlineMs from run timeoutMs");
assert(middlewareCtx.tags && middlewareCtx.tags.example === "07", "expected middleware tags from run options");

assert(!!toolCtx, "expected tool handler context");
assert(toolCtx.callId === "ctx-call-1", "expected tool handler callId");
assert(!!toolCtx.sessionId && !!toolCtx.inferenceId, "expected tool handler session/inference IDs");

assert(!!hookCtx, "expected hook payload");
assert(!!hookCtx.sessionId && !!hookCtx.inferenceId, "expected hook context IDs");
assert(hookCtx.tags && hookCtx.tags.demo === "context-and-constants", "expected hook tags from run options");

console.log("middleware ctx:", JSON.stringify(middlewareCtx));
console.log("tool ctx:", JSON.stringify(toolCtx));
console.log("hook ctx:", JSON.stringify(hookCtx));
