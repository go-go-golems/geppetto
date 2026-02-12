const gp = require("geppetto");

const tools = gp.tools.createRegistry();
tools.useGoTools(["go_double", "go_concat"]);

const direct = tools.call("go_double", { n: 21 });
assert(direct.value === 42, "direct go tool call mismatch");

const engine = gp.engines.fromFunction((turn) => {
  const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
  if (!hasToolUse) {
    turn.blocks.push(gp.turns.newToolCallBlock("call-go", "go_double", { n: 5 }));
    return turn;
  }

  turn.blocks.push(gp.turns.newAssistantBlock("go tools done"));
  return turn;
});

const session = gp
  .createBuilder()
  .withEngine(engine)
  .withTools(tools, { enabled: true, maxIterations: 3, toolChoice: "auto" })
  .buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("double 5")] }));
const out = session.run();

const toolUse = out.blocks.find((b) => b.kind === "tool_use");
assert(!!toolUse, "expected tool_use block from go tool");
const resultText = String(toolUse.payload && toolUse.payload.result);
assert(resultText.includes("10"), "go tool result missing doubled value");

console.log("direct go_double:", JSON.stringify(direct));
console.log("toolloop go_double result:", resultText);

