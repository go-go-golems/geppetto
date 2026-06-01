// Phase 5 hard-cut example: schema/tool/toolRegistry wrappers plus multimodal turn construction.
const gp = require("geppetto");

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

const turn = gp.turn()
  .system("You can inspect images.")
  .user(m => m
    .text("What is in this image?")
    .imageURL("https://example.invalid/screenshot.png"))
  .build();

const snapshot = turn.toJSON();
if (snapshot.blocks[1].payload.images[0].url !== "https://example.invalid/screenshot.png") {
  throw new Error("multimodal image URL missing");
}

console.log(JSON.stringify({
  tool: registry.list()[0].name,
  result: called.echoed,
  images: snapshot.blocks[1].payload.images.length,
}, null, 2));
