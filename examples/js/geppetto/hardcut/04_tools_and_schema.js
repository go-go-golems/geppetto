const gp = require("geppetto");
const input = gp.schema.object().property("value", gp.schema.string()).required("value").build();
const tool = gp.tool("echo_value").description("Echo").input(input).handler(args => ({ echoed: args.value })).build();
const registry = gp.toolRegistry().add(tool);
const result = registry.call("echo_value", { value: "hardcut" });
console.log(JSON.stringify({ tool: registry.list()[0].name, result: result.echoed }, null, 2));
