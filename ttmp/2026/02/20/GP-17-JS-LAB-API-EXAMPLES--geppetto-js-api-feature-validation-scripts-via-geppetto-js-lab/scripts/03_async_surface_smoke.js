const gp = require("geppetto");

// runAsync surface smoke
const asyncSession = gp.createSession({
  engine: gp.engines.echo({ reply: "ASYNC-OK" })
});
asyncSession.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("async run")] }));

const p = asyncSession.runAsync();
assert(!!p, "runAsync must return a promise-like object");
assert(typeof p.then === "function", "runAsync result must implement then()");

// start()/RunHandle surface smoke
const startSession = gp.createSession({
  engine: gp.engines.echo({ reply: "START-OK" })
});
startSession.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("start run")] }));

const handle = startSession.start(undefined, {
  timeoutMs: 1500,
  tags: {
    suite: "gp17",
    script: "03"
  }
});

assert(!!handle, "start() must return a handle");
assert(!!handle.promise, "RunHandle.promise missing");
assert(typeof handle.cancel === "function", "RunHandle.cancel missing");
assert(typeof handle.on === "function", "RunHandle.on missing");

let observedEventCount = 0;
handle.on(gp.consts.EventType.START, () => {
  observedEventCount += 1;
});
handle.on(gp.consts.EventType.FINAL, () => {
  observedEventCount += 1;
});
handle.on(gp.consts.EventType.ERROR, () => {
  observedEventCount += 1;
});

const cancelResult = handle.cancel();
const cancelType = typeof cancelResult;
assert(cancelType === "boolean" || cancelType === "undefined", "cancel() should be callable (boolean or undefined return)");
assert(typeof startSession.isRunning() === "boolean", "isRunning() should return boolean");

console.log("runAsyncThenType:", typeof p.then);
console.log("cancelResult:", cancelResult);
console.log("observedEventCountAtExit:", observedEventCount);
