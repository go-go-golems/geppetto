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
  "./cmd/examples/generic-tool-calling",
  "generic-tool-calling",
  "Use tools to get weather for San Francisco and compute 2 + 2.",
  "--pinocchio-profile",
  PROFILE,
  "--max-iterations",
  "3",
  "--max-parallel-tools",
  "1",
  "--tool-choice",
  "auto",
  "--tools-enabled",
  "--timeout",
  "60",
];

const env = { ...process.env, PINOCCHIO_PROFILE: PROFILE };
const res = spawnSync("go", args, {
  cwd: process.cwd(),
  env,
  encoding: "utf8",
  timeout: 300000,
  maxBuffer: 10 * 1024 * 1024,
});

const output = `${res.stdout || ""}\n${res.stderr || ""}`;

if (res.status === 0) {
  if (!/completed successfully|registered_tools|Final/i.test(output)) {
    console.error(output);
    throw new Error("builder/tools command succeeded but did not emit expected completion/tool markers");
  }
  console.log("PASS: builder+tools smoke test completed");
  process.exit(0);
}

if (isLikelyCredentialOrQuotaIssue(output)) {
  console.log("SKIP: builder+tools smoke test blocked by credentials/quota");
  console.log(output.split("\n").slice(0, 25).join("\n"));
  process.exit(0);
}

console.error(output);
throw new Error(`builder/tools smoke test failed with exit code ${res.status}`);
