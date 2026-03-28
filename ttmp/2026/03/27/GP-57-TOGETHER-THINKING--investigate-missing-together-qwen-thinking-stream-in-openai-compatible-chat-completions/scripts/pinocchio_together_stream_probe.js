const gp = require("geppetto");

const resolved = gp.profiles.resolve({});
const engine = gp.engines.fromResolvedProfile(resolved);

console.log(JSON.stringify({
  registrySlug: resolved.registrySlug || null,
  profileSlug: resolved.profileSlug || null,
  model: resolved.inferenceSettings?.chat?.engine || null,
  apiType: resolved.inferenceSettings?.chat?.api_type || null,
}, null, 2));

const handle = gp.runner.start({
  engine,
  prompt: "What is 17 * 23? Think step by step, then give a short final answer.",
});

const seen = {};

handle.on("*", (ev) => {
  const type = ev && ev.type ? ev.type : "unknown";
  seen[type] = (seen[type] || 0) + 1;
  if (type === "reasoning-text-delta") {
    console.log("REASONING_DELTA", JSON.stringify(ev.delta));
    return;
  }
  if (type === "partial-thinking") {
    console.log("PARTIAL_THINKING", JSON.stringify(ev.delta));
    return;
  }
  if (type === "partial") {
    console.log("TEXT_DELTA", JSON.stringify(ev.delta));
    return;
  }
  if (type === "info") {
    console.log("INFO", JSON.stringify(ev.message));
    return;
  }
  if (type === "final") {
    console.log("FINAL", JSON.stringify(ev.text));
    return;
  }
  console.log("EVENT", type, JSON.stringify(ev));
});

handle.promise.then((turn) => {
  console.log("EVENT_COUNTS", JSON.stringify(seen, null, 2));
  console.log("TURN", JSON.stringify(turn, null, 2));
}).catch((err) => {
  console.error("RUN_ERROR", String(err && err.stack ? err.stack : err));
  throw err;
});

// Note: pinocchio's current `js` command runs the script synchronously and does not
// await the returned promise. This file is preserved as an experiment artifact, but
// the Go probe is the authoritative streaming debugger for now.
