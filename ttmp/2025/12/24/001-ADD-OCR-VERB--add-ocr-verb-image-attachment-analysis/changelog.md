# Changelog

## 2026-03-16

- Moved the canonical ticket workspace from `pinocchio/ttmp` to `geppetto/ttmp`.
- Reason: the root bug and the multimodal turn payload contract are Geppetto-owned even though the CLI entrypoint is in Pinocchio.

## 2025-12-24

- Initial workspace created


## 2025-12-24

Step 1: Initial exploration - confirmed images parsed from CLI but not reaching LLM providers. Identified bug: imagePaths extracted but never passed to turn builder.


## 2025-12-24

Step 2: Completed image format conversion analysis. Identified conversion path: file paths → ImageContent → turn payload format. Created comprehensive analysis document.


## 2025-12-24

Analysis complete. Updated ticket index with summary of findings and fix requirements.


## 2025-12-24

Created comprehensive task list with 13 tasks covering implementation, testing, and documentation.


## 2025-12-24

Implemented fix: threaded --images via run.RunContext.ImagePaths, converted paths to turns image payload, attached to seed user block. Added unit tests; make build/test passed (pinocchio+geppetto); CLI now describes the image.
