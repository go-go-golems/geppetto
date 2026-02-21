#!/usr/bin/env node

const { spawnSync } = require("node:child_process");

const PROFILE = "gemini-2.5-flash-lite";

function run(cmd, args, timeout = 240000) {
  return spawnSync(cmd, args, {
    cwd: process.cwd(),
    env: { ...process.env, PINOCCHIO_PROFILE: PROFILE },
    encoding: "utf8",
    timeout,
    maxBuffer: 10 * 1024 * 1024,
  });
}

function isLikelyCredentialOrQuotaIssue(output) {
  return /(api key|missing.*key|unauthorized|permission denied|invalid api key|quota|rate limit|429)/i.test(
    output,
  );
}

const unit = run("go", [
  "test",
  "./pkg/js/modules/geppetto",
  "-run",
  "TestEnginesFromProfileAndFromConfigResolution",
  "-count=1",
  "-v",
]);

const unitOutput = `${unit.stdout || ""}\n${unit.stderr || ""}`;
if (unit.status !== 0 || !/PASS: TestEnginesFromProfileAndFromConfigResolution/.test(unitOutput)) {
  console.error(unitOutput);
  throw new Error(`engine profile unit smoke failed with exit code ${unit.status}`);
}

const inf = run("go", [
  "run",
  "./cmd/examples/simple-inference",
  "simple-inference",
  "Reply with exactly READY.",
  "--pinocchio-profile",
  PROFILE,
  "--timeout",
  "60",
], 300000);

const infOutput = `${inf.stdout || ""}\n${inf.stderr || ""}`;
if (inf.status === 0) {
  if (!/Final Turn/i.test(infOutput)) {
    console.error(infOutput);
    throw new Error("inference command succeeded but output did not include 'Final Turn'");
  }
  console.log("PASS: engine profile smoke test completed");
  process.exit(0);
}

if (isLikelyCredentialOrQuotaIssue(infOutput)) {
  console.log("SKIP: engine profile smoke inference blocked by credentials/quota");
  console.log(infOutput.split("\n").slice(0, 20).join("\n"));
  process.exit(0);
}

console.error(infOutput);
throw new Error(`engine profile smoke inference failed with exit code ${inf.status}`);
