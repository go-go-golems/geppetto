# Changelog

## 2026-06-02

- Initial workspace created


## 2026-06-02

Created session-centered JS API redesign guide and investigation diary.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/02/GP-JS-SESSION-API-2026-06-02--design-session-centered-geppetto-javascript-api/design-doc/01-session-centered-javascript-api-design-and-implementation-guide.md — Primary design and implementation guide
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/02/GP-JS-SESSION-API-2026-06-02--design-session-centered-geppetto-javascript-api/reference/01-investigation-diary.md — Investigation diary


## 2026-06-02

Uploaded session-centered JS API design guide bundle to reMarkable at /ai/2026/06/02/GP-JS-SESSION-API-2026-06-02.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/ttmp/2026/06/02/GP-JS-SESSION-API-2026-06-02--design-session-centered-geppetto-javascript-api/design-doc/01-session-centered-javascript-api-design-and-implementation-guide.md — Uploaded design guide


## 2026-06-02

Implemented JS session wrappers and hard-cut public execution to agent.session()/session.next() (commits 40fe7ec7, c4525da7).

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Updated API reference for session-centered execution
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_session.go — Session builder/session/turn-builder wrappers
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/module.go — Removed top-level gp.turn export


## 2026-06-02

Addressed PR #367 review comments: agent().goTool now falls back to host GoToolRegistry, and legacy provider registry config handling was removed.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/api_sessions.go — Go tool registry fallback
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/js/modules/geppetto/provider/provider.go — Legacy registry config removal


## 2026-06-02

Updated JS API docs after PR review follow-up to document agent().goTool(name) host-registry behavior; examples did not require changes because they do not use removed provider registry config.

### Related Files

- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/13-js-api-reference.md — Documents goTool host registry fallback
- /home/manuel/workspaces/2026-06-01/geppetto-js/geppetto/pkg/doc/topics/14-js-api-user-guide.md — Adds host Go tool selection guidance

