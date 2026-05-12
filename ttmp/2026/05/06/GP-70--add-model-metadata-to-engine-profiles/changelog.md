# Changelog

## 2026-05-06

- Initial workspace created


## 2026-05-06

Step 1: Created GP-70 ticket, wrote comprehensive design document with type sketches, pseudocode, merge semantics, YAML format, 6-phase implementation plan, data flow diagrams, and testing strategy. Removed full Go implementation code per user request, keeping only pseudocode and prose to leave implementor freedom.


## 2026-05-11

Replaced rough task list with detailed phased implementation tasks (6 phases + cross-cutting, ~22 tasks) covering every file and concern from the design doc


## 2026-05-11

Implemented GP-70 model metadata across Geppetto and Pinocchio: typed ModelInfo, profile merge/YAML tests, reasoning decisions, JS exposure, cost stamping, profile API/UI exposure, and task checklist updates.


## 2026-05-11

Updated permanent Geppetto and Pinocchio documentation for model_info profile metadata and JS/web-chat exposure.


## 2026-05-11

Fixed PR #351 CI generated-file drift by regenerating and committing pkg/doc/types/geppetto.d.ts after the ModelInfo JS API template changes.

