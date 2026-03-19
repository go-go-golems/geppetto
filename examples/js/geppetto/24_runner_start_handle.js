const gp = require("geppetto");

const registry = gp.tools.createRegistry();
registry.register({
  name: "echo_args",
  description: "echo tool input",
  handler: ({ value }) => ({ value }),
});

const runtime = gp.runner.resolveRuntime({
  systemPrompt: "Stream events and keep going.",
  runtimeKey: "stream-demo",
});

const handle = gp.runner.start({
  engine: gp.engines.fromFunction((turn) => {
    const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
    if (!hasToolUse) {
      turn.blocks.push(gp.turns.newToolCallBlock("call-1", "echo_args", { value: "stream" }));
      return turn;
    }
    turn.blocks.push(gp.turns.newAssistantBlock("done"));
    return turn;
  }),
  runtime,
  tools: registry,
  toolLoop: { enabled: true, maxIterations: 3, toolErrorHandling: "continue" },
  prompt: "start streaming",
}, {
  timeoutMs: 1000,
  tags: { mode: "runner-start-example" },
});

let seen = 0;
handle.on("*", () => {
  seen++;
});

assert(handle && typeof handle === "object", "runner.start should return an object");
assert(handle.promise && typeof handle.promise.then === "function", "runner.start handle missing promise");
assert(typeof handle.cancel === "function", "runner.start handle missing cancel()");
assert(typeof handle.on === "function", "runner.start handle missing on()");
assert(handle.session && handle.session.turnCount() === 1, "runner.start handle should expose the prepared session");
assert(handle.runtime && handle.runtime.runtimeKey === "stream-demo", "runner.start handle should expose runtime");

console.log(JSON.stringify({
  ok: true,
  runtimeKey: handle.runtime.runtimeKey,
  subscribed: seen,
}));
