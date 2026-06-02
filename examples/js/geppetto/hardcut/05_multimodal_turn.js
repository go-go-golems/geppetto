const gp = require("geppetto");
const source = globalThis.GEPPETTO_PHASE123_PROFILE || "examples/js/geppetto/profiles/50-hardcut-phase123.yaml";
const registry = gp.inferenceProfiles.load(source);
const settings = registry.resolve("assistant");
const agent = gp.agent().inference(settings).build();
const session = agent.session().id("hardcut-multimodal-demo").build();
const builder = session.next()
  .system("You inspect images.")
  .user(m => m.text("What is in this image?").imageURL("https://example.invalid/image.png"));
if (typeof builder.run !== "function") throw new Error("session turn builder missing run");
console.log(JSON.stringify({ session: session.id(), multimodalInput: true }, null, 2));
registry.close();
