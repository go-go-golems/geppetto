const gp = require("geppetto");

const tools = gp.tools.createRegistry();
tools.register({
  name: "js_add",
  description: "Add two numbers",
  handler: ({ a, b }) => ({ sum: a + b })
});

const engine = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("call-1", "js_add", { a: 2, b: 3 }));
    return turn;
  }

  turn.blocks.push(gp.turns.newAssistantBlock("done"));
  return turn;
});

const session = gp
  .createBuilder()
  .withEngine(engine)
  .withTools(tools, {
    enabled: true,
    maxIterations: 3,
    toolChoice: "auto",
    maxParallelTools: 1
  })
  .buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("compute")] }));
const out = session.run();

const toolUse = out.blocks.find((b) => b.kind === "tool_use");
assert(!!toolUse, "expected tool_use block");
const resultText = String(toolUse.payload && toolUse.payload.result);
assert(resultText.includes("sum"), "tool result missing sum");

console.log("tool result:", resultText);

