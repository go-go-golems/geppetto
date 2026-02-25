const gp = require("geppetto");

const rows = gp.schemas.listExtensions();
assert(Array.isArray(rows), "listExtensions should return an array");
assert(rows.length >= 3, "expected codec and host extension schemas");

const keys = rows.map((r) => r.key).sort();
assert(keys.includes("demo.analytics@v1"), "missing demo.analytics@v1 schema");
assert(keys.includes("demo.safety@v1"), "missing demo.safety@v1 schema");
assert(keys.includes("host.notes@v1"), "missing host.notes@v1 schema");

const analytics = rows.find((r) => r.key === "demo.analytics@v1");
assert(analytics.displayName === "Demo Analytics", "unexpected analytics display name");
assert(analytics.schema && analytics.schema.type === "object", "analytics schema payload missing");

console.log("extension schemas:", JSON.stringify(keys));
