# Changelog

## 2026-03-18

- Initial workspace created
- Implemented the first runtime-metadata cleanup slice in `677f7a2`, adding internal helpers that materialize resolved profile runtime metadata into middlewares, filtered tool registries, and stamped turn metadata for JS session assembly.
- Implemented the public-facing follow-up in `01e2e89`, documenting `resolvedProfile` / `useResolvedProfile(...)`, updating the JS type surface, and adding an executable example that shows the intended execution path.
- Closed the ticket after GP-46 landed on top of this cleanup and the runtime-metadata boundary was no longer a separate open workstream.
