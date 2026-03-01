#!/usr/bin/env python3
"""Probe go-go-goja tsgen model and renderer capabilities."""

from __future__ import annotations

import argparse
import pathlib
import re
import sys


RE_TYPE_KIND = re.compile(r"TypeKind[A-Za-z0-9_]+\s+TypeKind\s*=\s*\"([a-z_]+)\"")
RE_RENDER_CASE = re.compile(r"case spec\.TypeKind([A-Za-z0-9_]+):")
RE_MODULE_FIELDS = re.compile(r"type Module struct \{\n([^}]*)\n\}", re.MULTILINE)
RE_STRUCT_FIELD = re.compile(r"^\s*([A-Za-z0-9_]+)\s+", re.MULTILINE)


def _extract_module_fields(types_go: str) -> list[str]:
    m = RE_MODULE_FIELDS.search(types_go)
    if not m:
        return []
    body = m.group(1)
    return RE_STRUCT_FIELD.findall(body)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--types-go", required=True)
    parser.add_argument("--renderer-go", required=True)
    parser.add_argument("--validator-go", required=True)
    args = parser.parse_args()

    types_text = pathlib.Path(args.types_go).read_text(encoding="utf-8")
    renderer_text = pathlib.Path(args.renderer_go).read_text(encoding="utf-8")
    validator_text = pathlib.Path(args.validator_go).read_text(encoding="utf-8")

    type_kinds = sorted(set(RE_TYPE_KIND.findall(types_text)))
    render_cases = sorted(set(x.lower() for x in RE_RENDER_CASE.findall(renderer_text)))
    validate_cases = sorted(set(x.lower() for x in RE_RENDER_CASE.findall(validator_text)))
    module_fields = _extract_module_fields(types_text)

    has_interface_field = "Interfaces" in module_fields
    has_type_alias_field = "TypeAliases" in module_fields
    has_const_field = "Consts" in module_fields or "Constants" in module_fields

    lines: list[str] = []
    lines.append("# tsgen Capability Report")
    lines.append("")
    lines.append("## spec.Module Shape")
    lines.append("")
    lines.append(f"- Module fields: {', '.join(module_fields)}")
    lines.append(f"- Supports first-class interfaces field: {has_interface_field}")
    lines.append(f"- Supports first-class type aliases field: {has_type_alias_field}")
    lines.append(f"- Supports first-class constants field: {has_const_field}")
    lines.append("")
    lines.append("## TypeRef Coverage")
    lines.append("")
    lines.append(f"- Defined TypeKind values: {', '.join(type_kinds)}")
    lines.append(f"- Renderer switch cases: {', '.join(render_cases)}")
    lines.append(f"- Validator switch cases: {', '.join(validate_cases)}")
    lines.append("")
    lines.append("## Renderer Hooks")
    lines.append("")
    lines.append(
        f"- Renders function declarations directly: {'renderFunction(' in renderer_text}"
    )
    lines.append(f"- Supports raw passthrough lines (RawDTS): {'module.RawDTS' in renderer_text}")
    lines.append(
        "- Has dedicated renderInterface/renderTypeAlias renderer paths: "
        + str(("renderInterface(" in renderer_text) or ("renderTypeAlias(" in renderer_text))
    )
    lines.append(
        "- Can model interface/type alias/const declarations without RawDTS passthrough: "
        + str(has_interface_field or has_type_alias_field or has_const_field)
    )
    lines.append("")

    sys.stdout.write("\n".join(lines))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
