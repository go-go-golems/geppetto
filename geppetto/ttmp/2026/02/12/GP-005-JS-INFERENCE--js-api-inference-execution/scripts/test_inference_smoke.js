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
  "./cmd/examples/simple-inference",
  "simple-inference",
  "Reply with exactly READY.",
  "--pinocchio-profile",
  PROFILE,
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
  if (!/Final Turn/i.test(output)) {
    console.error(output);
    throw new Error("inference command succeeded but output did not include 'Final Turn'");
  }
  console.log("PASS: inference smoke test completed");
  process.exit(0);
}

if (isLikelyCredentialOrQuotaIssue(output)) {
  console.log("SKIP: inference smoke test blocked by credentials/quota");
  console.log(output.split("\n").slice(0, 20).join("\n"));
  process.exit(0);
}

console.error(output);
throw new Error(`inference smoke test failed with exit code ${res.status}`);
