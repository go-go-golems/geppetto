#!/usr/bin/env python3
"""Probe geppetto.d.ts surface complexity for true-replacement planning."""

from __future__ import annotations

import argparse
import pathlib
import re
import sys
from dataclasses import dataclass


RE_EXPORT = re.compile(
    r"(?m)^\s*export (const|interface|type|function) ([A-Za-z_][A-Za-z0-9_]*)"
)
RE_CONST_OBJECT = re.compile(
    r"(?m)^\s*export const ([A-Za-z_][A-Za-z0-9_]*)\s*:\s*\{"
)
RE_OBJECT_MEMBER = re.compile(r"(?m)^\s{8}([A-Za-z_][A-Za-z0-9_]*)\s*(?:\(|:)")


@dataclass
class SurfaceSummary:
    total_lines: int
    export_counts: dict[str, int]
    top_level_consts: list[str]
    top_level_interfaces: list[str]
    top_level_types: list[str]
    top_level_functions: list[str]
    grouped_exports: dict[str, list[str]]
    feature_counts: dict[str, int]


def _find_matching_brace(text: str, open_idx: int) -> int:
    depth = 0
    for i in range(open_idx, len(text)):
        c = text[i]
        if c == "{":
            depth += 1
        elif c == "}":
            depth -= 1
            if depth == 0:
                return i
    raise ValueError("unclosed brace in declaration content")


def summarize_surface(text: str) -> SurfaceSummary:
    exports: dict[str, list[str]] = {"const": [], "interface": [], "type": [], "function": []}
    for kind, name in RE_EXPORT.findall(text):
        exports[kind].append(name)

    grouped: dict[str, list[str]] = {}
    for m in RE_CONST_OBJECT.finditer(text):
        name = m.group(1)
        open_idx = m.end() - 1
        close_idx = _find_matching_brace(text, open_idx)
        body = text[open_idx + 1 : close_idx]
        members = sorted(set(RE_OBJECT_MEMBER.findall(body)))
        grouped[name] = members

    features = {
        "inline_arrow_callbacks": len(re.findall(r"=>", text)),
        "promises": len(re.findall(r"\bPromise<", text)),
        "record_types": len(re.findall(r"\bRecord<", text)),
        "partial_types": len(re.findall(r"\bPartial<", text)),
        "readonly_fields": len(re.findall(r"\breadonly\b", text)),
        "string_literal_unions": len(re.findall(r'"[^"]+"\s*\|\s*"[^"]+"', text)),
    }

    export_counts = {k: len(v) for k, v in exports.items()}
    return SurfaceSummary(
        total_lines=text.count("\n") + 1,
        export_counts=export_counts,
        top_level_consts=sorted(set(exports["const"])),
        top_level_interfaces=sorted(set(exports["interface"])),
        top_level_types=sorted(set(exports["type"])),
        top_level_functions=sorted(set(exports["function"])),
        grouped_exports=grouped,
        feature_counts=features,
    )


def _emit_markdown(summary: SurfaceSummary, dts_path: pathlib.Path) -> str:
    lines: list[str] = []
    lines.append(f"# d.ts Surface Report: `{dts_path}`")
    lines.append("")
    lines.append("## Totals")
    lines.append("")
    lines.append(f"- Lines: {summary.total_lines}")
    lines.append(f"- Export `const` count: {summary.export_counts['const']}")
    lines.append(f"- Export `interface` count: {summary.export_counts['interface']}")
    lines.append(f"- Export `type` count: {summary.export_counts['type']}")
    lines.append(f"- Export `function` count: {summary.export_counts['function']}")
    lines.append("")
    lines.append("## Top-Level Export Names")
    lines.append("")
    lines.append(f"- const: {', '.join(summary.top_level_consts)}")
    lines.append(
        "- interface sample (first 20): "
        + ", ".join(summary.top_level_interfaces[:20])
        + (" ..." if len(summary.top_level_interfaces) > 20 else "")
    )
    lines.append(f"- type: {', '.join(summary.top_level_types)}")
    lines.append(f"- function: {', '.join(summary.top_level_functions)}")
    lines.append("")
    lines.append("## Grouped Object Exports")
    lines.append("")
    for name in sorted(summary.grouped_exports.keys()):
        members = summary.grouped_exports[name]
        lines.append(f"- {name}: {len(members)} members")
        if members:
            lines.append("  - " + ", ".join(members))
    lines.append("")
    lines.append("## Feature Signals")
    lines.append("")
    for key in sorted(summary.feature_counts.keys()):
        lines.append(f"- {key}: {summary.feature_counts[key]}")
    lines.append("")
    return "\n".join(lines)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dts", required=True, help="Path to geppetto.d.ts")
    args = parser.parse_args()

    dts_path = pathlib.Path(args.dts)
    text = dts_path.read_text(encoding="utf-8")
    summary = summarize_surface(text)
    sys.stdout.write(_emit_markdown(summary, dts_path))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
