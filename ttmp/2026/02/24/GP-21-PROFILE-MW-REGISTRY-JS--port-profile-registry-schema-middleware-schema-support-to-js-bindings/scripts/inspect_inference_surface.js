const gp = require("geppetto");

const builder = gp.createBuilder().withEngine(gp.engines.echo({ reply: "OK" }));
const session = builder.buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hello")] }));
const handle = session.start(undefined, { timeoutMs: 500, tags: { probe: true } });

const payload = {
  topLevelKeys: Object.keys(gp).sort(),
  builderKeys: Object.keys(builder).sort(),
  sessionKeys: Object.keys(session).sort(),
  runHandleKeys: Object.keys(handle).sort(),
  hasRunAsync: typeof session.runAsync === "function",
  hasStart: typeof session.start === "function",
  hasCancelActive: typeof session.cancelActive === "function",
  hasToolLoopBuilderMethods:
    typeof builder.withTools === "function" && typeof builder.withToolLoop === "function",
};

console.log(payload);
