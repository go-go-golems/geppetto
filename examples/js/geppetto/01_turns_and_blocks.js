const gp = require("geppetto");

const turn = gp.turns.newTurn({
  id: "turn-1",
  blocks: [
    gp.turns.newUserBlock("hello"),
    gp.turns.newToolCallBlock("call-1", "js_add", { a: 2, b: 3 })
  ],
  metadata: {
    session_id: "s-1"
  },
  data: {
    tool_config: { enabled: true }
  }
});

assert(turn.id === "turn-1", "turn id mismatch");
assert(Array.isArray(turn.blocks), "blocks must be an array");
assert(turn.blocks.length === 2, "expected two blocks");
assert(turn.blocks[0].kind === "user", "first block must be user");
assert(turn.blocks[1].kind === "tool_call", "second block must be tool_call");
assert(turn.metadata.session_id === "s-1", "metadata.session_id mismatch");

const normalized = gp.turns.normalize(turn);
assert(normalized.blocks.length === 2, "normalize changed block count unexpectedly");

console.log("turn id:", normalized.id);
console.log("block kinds:", normalized.blocks.map((b) => b.kind));

