#!/usr/bin/env node

const { spawnSync } = require("node:child_process");

const PROFILE = "gemini-2.5-flash-lite";

function isLikelyCredentialOrQuotaIssue(output) {
  return /(api key|missing.*key|unauthorized|permission denied|invalid api key|quota|rate limit|429)/i.test(
    output,
  );
}

const args = [
  "run",
  "./cmd/examples/middleware-inference",
  "middleware-inference",
  "Respond with exactly middleware check.",
  "--pinocchio-profile",
  PROFILE,
  "--with-uppercase",
  "--timeout",
  "60",
];

const env = { ...process.env, PINOCCHIO_PROFILE: PROFILE };
const res = spawnSync("go", args, {
  cwd: process.cwd(),
  env,
  encoding: "utf8",
  timeout: 240000,
  maxBuffer: 8 * 1024 * 1024,
});

const output = `${res.stdout || ""}\n${res.stderr || ""}`;

if (res.status === 0) {
  if (!/Applied middleware: uppercase/i.test(output)) {
    console.error(output);
    throw new Error("middleware command succeeded but did not report uppercase middleware");
  }
  console.log("PASS: middleware smoke test completed");
  process.exit(0);
}

if (isLikelyCredentialOrQuotaIssue(output)) {
  console.log("SKIP: middleware smoke test blocked by credentials/quota");
  console.log(output.split("\n").slice(0, 20).join("\n"));
  process.exit(0);
}

console.error(output);
throw new Error(`middleware smoke test failed with exit code ${res.status}`);
