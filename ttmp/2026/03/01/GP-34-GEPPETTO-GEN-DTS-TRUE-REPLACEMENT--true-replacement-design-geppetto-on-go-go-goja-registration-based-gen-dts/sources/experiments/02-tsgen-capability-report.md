# tsgen Capability Report

## spec.Module Shape

- Module fields: Name, Description, Functions, RawDTS
- Supports first-class interfaces field: False
- Supports first-class type aliases field: False
- Supports first-class constants field: False

## TypeRef Coverage

- Defined TypeKind values: any, array, boolean, named, never, number, object, string, union, unknown, void
- Renderer switch cases: any, array, boolean, named, never, number, object, string, union, unknown, void
- Validator switch cases: array, named, object, union

## Renderer Hooks

- Renders function declarations directly: True
- Supports raw passthrough lines (RawDTS): True
- Has dedicated renderInterface/renderTypeAlias renderer paths: False
- Can model interface/type alias/const declarations without RawDTS passthrough: False
