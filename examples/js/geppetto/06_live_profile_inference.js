const gp = require("geppetto");

const apiKey = ENV.GEMINI_API_KEY || ENV.GOOGLE_API_KEY || "";
if (!apiKey) {
  console.log("SKIP: set GEMINI_API_KEY or GOOGLE_API_KEY to run live inference");
} else {
  const session = gp.createSession({
    engine: gp.engines.fromConfig({
      apiType: "gemini",
      model: "gemini-2.5-flash-lite",
      apiKey
    })
  });

  session.append(
    gp.turns.newTurn({
      blocks: [gp.turns.newUserBlock("Reply with exactly READY.")]
    })
  );

  const out = session.run();
  assert(Array.isArray(out.blocks), "expected output blocks");
  assert(out.blocks.length >= 2, "expected at least system+assistant/user+assistant blocks");

  const last = out.blocks[out.blocks.length - 1];
  console.log("last block kind:", last.kind);
  console.log("last block payload:", JSON.stringify(last.payload || {}));
}
