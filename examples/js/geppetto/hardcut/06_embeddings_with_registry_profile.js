const gp = require("geppetto");
if (typeof gp.embeddings !== "function") {
  console.log(JSON.stringify({ skipped: true, reason: "gp.embeddings hard-cut wrapper is not implemented yet" }, null, 2));
} else {
  throw new Error("Update this example once gp.embeddings() is implemented.");
}
