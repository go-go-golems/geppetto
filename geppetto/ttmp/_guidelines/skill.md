# Guidelines: Skill Documents

## Purpose
Skills are **disciplined workflows** written as documents (`DocType: skill`) that teach LLMs (and humans) *how to work*, not just what exists. A good skill turns “best practice” into a repeatable, enforceable process.

Skills are meant to be discoverable via:
- `docmgr skill list` (filter by topics, file/dir, ticket)
- `docmgr skill show <query>` (load and apply a skill)

## Required Elements
- **Frontmatter contract**
  - `DocType: skill`
  - `Title`: Use the convention `Skill: <Name>` (recommended)
  - `Topics`: Choose topics that match how developers will search (e.g. `tdd`, `debugging`, `docs`)
  - `WhatFor`: 2–5 sentences describing the outcome this workflow ensures
  - `WhenToUse`: Clear trigger conditions (“Use when …”)
- **Body**
  - **Overview**: 2–5 sentences (why it matters; what it enforces)
  - **When to Use**: Concrete triggers + examples
  - **Process**: Step-by-step actions (copy/paste friendly commands where possible)
  - **Verification**: Checklist to prevent “I think I’m done” drift

## Best Practices
- **Be explicit**: Use strong modal language (“MUST”, “NEVER”, “STOP”) where appropriate.
- **Optimize for discovery**: Ensure `Topics`, `WhatFor`, and `WhenToUse` contain the words people will actually search for.
- **Prefer commands over prose**: If there’s a canonical command sequence, include it as a `bash` block.
- **Make validation unskippable**: Add a checklist and expected success criteria (tests/linters/scenario suites).

## References
- `pkg/doc/how-to-write-skills.md` — Full guidance on writing and enforcing skills
- `pkg/doc/using-skills.md` — How to discover and load skills via CLI


