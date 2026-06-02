// Host-backed turn-store persistence example for the session-centered JS API.
//
// This script requires an xgoja/Go host that configures require("geppetto") with
// turn storage enabled (for example, a Pinocchio-style turns DSN adapter). The
// plain geppetto-js-run helper configures profiles but does not open a turn
// store by itself, so this example is documentation-first until a host adapter
// is installed.

const gp = require("geppetto");

const profile = globalThis.GEPPETTO_PROFILE || "default";
const sessionId = globalThis.GEPPETTO_TURNSTORE_SESSION_ID || "geppetto-js-turnstore-demo";
const timeoutMs = Number(globalThis.GEPPETTO_TIMEOUT_MS || 120000);

let store;
try {
  store = gp.turnStores.default();
} catch (err) {
  throw new Error(`turn store is not configured by this host: ${err}`);
}

const settings = gp.inferenceProfiles.resolve(profile);
const agent = gp.agent()
  .name("turn-store-persistence-example")
  .inference(settings)
  .store(store)
  .runDefaults({ timeoutMs, tags: { example: "turn-store-persistence", sessionId } })
  .build();

const session = agent.session()
  .id(sessionId)
  .store(store)
  .metadata("example", "turn-store-persistence")
  .build();

const result = session.next()
  .system("Answer in exactly one short sentence.")
  .user("Explain why explicit turn persistence is useful.")
  .run();

const latest = store.loadLatest({ sessionId, phase: "final" });

console.log(JSON.stringify({
  profile,
  sessionId,
  finalText: result.text(),
  turnCount: session.turnCount(),
  latest: latest ? {
    turnId: latest.turnId,
    sessionId: latest.sessionId,
    inferenceId: latest.inferenceId,
    phase: latest.phase,
    text: latest.turn ? latest.turn.toJSON().blocks
      .filter(block => block.role === "assistant" || block.kind === "llm_text")
      .map(block => block.payload && block.payload.text)
      .filter(Boolean)
      .join("\n") : "",
  } : null,
}, null, 2));
