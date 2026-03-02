const gp = require("geppetto");

const registry = gp.tools.createRegistry();
registry.register({
	name: "echo_args",
	description: "echo tool input",
	handler: ({ value }) => ({ value }),
});

const engine = gp.engines.fromFunction((turn) => {
	const hasToolUse = turn.blocks.some((b) => b.kind === "tool_use");
	if (!hasToolUse) {
		turn.blocks.push(gp.turns.newToolCallBlock("call-1", "echo_args", { value: "x" }));
		return turn;
	}
	turn.blocks.push(gp.turns.newAssistantBlock("done"));
	return turn;
});

const seen = {
	all: 0,
	execute: 0,
	result: 0,
};

const sink = gp.events.collector()
	.on("*", (ev) => {
		seen.all++;
		if (ev && ev.type === "tool-call-execution-result") {
			seen.result++;
		}
	})
	.on("tool-call-execute", () => {
		seen.execute++;
	});

const session = gp.createBuilder()
	.withEngine(engine)
	.withTools(registry, { enabled: true, maxIterations: 3, toolErrorHandling: "continue" })
	.withEventSink(sink)
	.buildSession();

session.append(gp.turns.newTurn({ blocks: [gp.turns.newUserBlock("hello")] }));
const out = session.run();
const last = out.blocks[out.blocks.length - 1];
assert(last && last.kind === "llm_text", "expected assistant output");
assert(seen.all > 0, "expected collector wildcard callbacks");
assert(seen.execute > 0, "expected tool-call-execute callback");
assert(seen.result > 0, "expected tool-call-execution-result callback");
