const gp = require("geppetto");

const session = gp.createSession({
  engine: gp.engines.echo({ reply: "READY" })
});

session.append(
  gp.turns.newTurn({
    id: "t1",
    blocks: [gp.turns.newUserBlock("reply with READY")]
  })
);

const out = session.run();
const last = out.blocks[out.blocks.length - 1];

assert(last.kind === "llm_text", "missing llm_text output");
assert(last.payload.text === "READY", "unexpected assistant text");
assert(session.turnCount() === 1, "turnCount mismatch");
assert(session.getTurn(0).id === "t1", "history lookup mismatch");

console.log("assistant:", last.payload.text);
console.log("turnCount:", session.turnCount());

