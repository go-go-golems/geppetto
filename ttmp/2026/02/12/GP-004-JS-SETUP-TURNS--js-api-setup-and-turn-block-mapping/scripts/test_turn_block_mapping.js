#!/usr/bin/env node

const fs = require("node:fs");
const path = require("node:path");

function assert(cond, msg) {
  if (!cond) {
    throw new Error(msg);
  }
}

const repoRoot = process.cwd();
const kindsPath = path.join(repoRoot, "pkg/turns/block_kind_gen.go");
const keysPath = path.join(repoRoot, "pkg/turns/keys_gen.go");

assert(fs.existsSync(kindsPath), `missing generated kinds file: ${kindsPath}`);
assert(fs.existsSync(keysPath), `missing generated keys file: ${keysPath}`);

const kinds = fs.readFileSync(kindsPath, "utf8");
const keys = fs.readFileSync(keysPath, "utf8");

assert(kinds.includes("func (k BlockKind) String() string"), "missing BlockKind String mapper");
assert(kinds.includes("func (k *BlockKind) UnmarshalYAML"), "missing BlockKind YAML decode mapper");
assert(kinds.includes("BlockKindOther"), "missing BlockKindOther fallback");
assert(kinds.includes("case \"user\":"), "missing canonical user kind mapping");

assert(keys.includes("const ("), "missing generated key constants block");
assert(keys.includes("KeyTurnMetaSessionID"), "missing generated KeyTurnMetaSessionID");
assert(keys.includes("AgentModeAllowedToolsValueKey"), "missing generated AgentModeAllowedToolsValueKey");
assert(keys.includes("ResponsesServerToolsValueKey"), "missing generated ResponsesServerToolsValueKey");
assert(keys.includes("TurnMetaK["), "missing TurnMeta typed key builder usage");
assert(keys.includes("BlockMetaK["), "missing BlockMeta typed key builder usage");

const kindConstCount = (kinds.match(/BlockKind[A-Za-z0-9_]+\s+BlockKind/g) || []).length;
const keyConstCount = (keys.match(/ValueKey\s*=/g) || []).length;

assert(kindConstCount >= 6, `unexpectedly low generated block kind count: ${kindConstCount}`);
assert(keyConstCount >= 10, `unexpectedly low generated key constant count: ${keyConstCount}`);

console.log("PASS: turn/block generated mapper contract checks");
console.log(`INFO: kind constants=${kindConstCount}, key constants=${keyConstCount}`);
