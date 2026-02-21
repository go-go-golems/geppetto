#!/usr/bin/env node

const { spawnSync } = require("node:child_process");

const res = spawnSync(
  "go",
  [
    "test",
    "./pkg/js/modules/geppetto",
    "-run",
    "TestToolLoopHooksMutationRetryAbortAndHookPolicy",
    "-count=1",
    "-v",
  ],
  {
    cwd: process.cwd(),
    encoding: "utf8",
    timeout: 240000,
    maxBuffer: 10 * 1024 * 1024,
  },
);

const output = `${res.stdout || ""}\n${res.stderr || ""}`;
if (res.status !== 0) {
  console.error(output);
  throw new Error(`toolloop hooks smoke failed with exit code ${res.status}`);
}
if (!/PASS: TestToolLoopHooksMutationRetryAbortAndHookPolicy/.test(output)) {
  console.error(output);
  throw new Error("toolloop hooks smoke succeeded but expected test marker missing");
}

console.log("PASS: toolloop hooks smoke test completed");
