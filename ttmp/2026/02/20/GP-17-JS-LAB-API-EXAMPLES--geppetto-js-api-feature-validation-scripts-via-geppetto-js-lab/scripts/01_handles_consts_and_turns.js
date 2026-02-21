const gp = require("geppetto");

assert(!!gp.consts, "gp.consts must exist");
assert(gp.consts.ToolChoice.AUTO === "auto", "ToolChoice.AUTO mismatch");
assert(gp.consts.BlockKind.TOOL_CALL === "tool_call", "BlockKind.TOOL_CALL mismatch");
assert(gp.consts.TurnDataKeys.TOOL_CONFIG === "tool_config", "TurnDataKeys.TOOL_CONFIG mismatch");
assert(gp.consts.TurnMetadataKeys.SESSION_ID === "session_id", "TurnMetadataKeys.SESSION_ID mismatch");
assert(gp.consts.BlockMetadataKeys.MIDDLEWARE === "middleware", "BlockMetadataKeys.MIDDLEWARE mismatch");
assert(gp.consts.RunMetadataKeys.TRACE_ID === "trace_id", "RunMetadataKeys.TRACE_ID mismatch");
assert(gp.consts.PayloadKeys.ENCRYPTED_CONTENT === "encrypted_content", "PayloadKeys.ENCRYPTED_CONTENT mismatch");
assert(gp.consts.EventType.TOOL_RESULT === "tool-result", "EventType.TOOL_RESULT mismatch");

const engine = gp.engines.echo({ reply: "OK-HANDLE" });
const keys = Object.keys(engine);
assert(!keys.includes("__geppetto_ref"), "__geppetto_ref should be hidden");

const serialized = JSON.stringify(engine);
assert(!serialized.includes("__geppetto_ref"), "__geppetto_ref should not serialize");

// Attempt to overwrite hidden ref; should not break behavior.
engine.__geppetto_ref = 42;

const session = gp.createSession({ engine });
session.append(
  gp.turns.newTurn({
    id: "t-handle-1",
    blocks: [gp.turns.newUserBlock("hello from handles script")],
    metadata: {
      [gp.consts.TurnMetadataKeys.SESSION_ID]: "session-handles"
    }
  })
);

const out = session.run();
assert(Array.isArray(out.blocks), "output blocks must be array");
const last = out.blocks[out.blocks.length - 1];
assert(last.kind === gp.consts.BlockKind.LLM_TEXT, "expected llm_text output");
assert(last.payload.text === "OK-HANDLE", "unexpected echo output");

console.log("keys(engine):", JSON.stringify(keys));
console.log("last.kind:", last.kind);
console.log("last.payload.text:", last.payload.text);
