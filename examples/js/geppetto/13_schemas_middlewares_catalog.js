const gp = require("geppetto");

const rows = gp.schemas.listMiddlewares();
assert(Array.isArray(rows), "listMiddlewares should return an array");
assert(rows.length >= 3, "expected demo middleware schema definitions");

const names = rows.map((r) => r.name).sort();
assert(names.includes("agentmode"), "missing agentmode schema");
assert(names.includes("retry"), "missing retry schema");
assert(names.includes("telemetry"), "missing telemetry schema");

const retry = rows.find((r) => r.name === "retry");
assert(retry && retry.schema && retry.schema.type === "object", "retry schema should be an object");

console.log("middleware schemas:", JSON.stringify(names));
