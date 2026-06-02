// Phase 5 hard-cut example: schema/tool/toolRegistry wrappers plus multimodal session input.
const gp = require("geppetto");

const source = globalThis.GEPPETTO_PHASE123_PROFILE ||
  "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";
const registryFile = gp.inferenceProfiles.load(source);
const settings = registryFile.resolve("assistant");

const input = gp.schema.object()
  .property("value", gp.schema.string().description("Value to echo"))
  .required("value")
  .build();

const echo = gp.tool("echo_value")
  .description("Echo a value")
  .input(input)
  .handler((args, ctx) => ({ echoed: args.value, toolName: ctx.toolName }))
  .build();

const registry = gp.toolRegistry().add(echo);
const called = registry.call("echo_value", { value: "phase5" });
if (called.echoed !== "phase5") {
  throw new Error("tool call failed");
}

const agent = gp.agent().inference(settings).tool(registry).build();
const session = agent.session().id("multimodal-demo").build();
const builder = session.next()
  .system("You can inspect images.")
  .user(m => m
    .text("What is in this image?")
    .imageURL("https://example.invalid/screenshot.png"));

if (!builder || typeof builder.run !== "function") {
  throw new Error("session.next() did not create a runnable turn builder");
}

console.log(JSON.stringify({
  tool: registry.list()[0].name,
  result: called.echoed,
  sessionId: session.id(),
  multimodalInput: true,
}, null, 2));

registryFile.close();
