#!/usr/bin/env node

const { spawnSync } = require("node:child_process");

const args = [
  "test",
  "./pkg/js/modules/geppetto",
  "-run",
  "TestSessionHistoryInspectionAndSnapshotImmutability",
  "-count=1",
  "-v",
];

const res = spawnSync("go", args, {
  cwd: process.cwd(),
  encoding: "utf8",
  timeout: 180000,
  maxBuffer: 8 * 1024 * 1024,
});

const output = `${res.stdout || ""}\n${res.stderr || ""}`;
if (res.status !== 0) {
  console.error(output);
  throw new Error(`session history smoke failed with exit code ${res.status}`);
}
if (!/PASS: TestSessionHistoryInspectionAndSnapshotImmutability/.test(output)) {
  console.error(output);
  throw new Error("session history smoke succeeded but expected test marker missing");
}

console.log("PASS: session history smoke test completed");
