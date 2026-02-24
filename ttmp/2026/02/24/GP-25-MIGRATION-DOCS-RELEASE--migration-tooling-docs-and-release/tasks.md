# Tasks

## Migration Command Quality

- [ ] Audit `profiles_migrate_legacy` command flags and defaults for final UX.
- [ ] Add tests for legacy map input detection.
- [ ] Add tests for single-registry input detection.
- [ ] Add tests for canonical multi-registry input passthrough.
- [ ] Add tests for invalid registry slug handling.
- [ ] Add tests for `--in-place` behavior.
- [ ] Add tests for default output path behavior.
- [ ] Add tests for summary output metrics.
- [ ] Add tests for idempotent repeated conversion.

## Migration Fixture Corpus

- [ ] Add representative legacy fixtures with minimal profile fields.
- [ ] Add fixtures with multiple profiles and mixed middleware/tool sections.
- [ ] Add fixtures with extension payload examples.
- [ ] Add malformed YAML fixtures for error-path testing.
- [ ] Add fixtures with comments/format variations to test parser resilience.

## Geppetto Documentation

- [ ] Update `geppetto` profile topic with registry-first model details.
- [ ] Update profile topic with extension key conventions and examples.
- [ ] Update migration playbook with latest command examples.
- [ ] Add troubleshooting section for common migration failures.
- [ ] Validate geppetto help-page frontmatter for updated docs.

## Pinocchio Documentation

- [ ] Add/update pinocchio profile registry help page.
- [ ] Add migration playbook in pinocchio docs for legacy profile conversion.
- [ ] Add migration section for removed aliases and renamed symbols.
- [ ] Add before/after API examples for CRUD and profile selection endpoints.
- [ ] Validate pinocchio help-page frontmatter and discoverability.

## Third-Party Migration Guidance

- [ ] Document all removed aliases with replacements in one table.
- [ ] Document removed compatibility env vars and their replacements.
- [ ] Provide code snippet migration guide for common third-party usage paths.
- [ ] Add risk notes for integrations pinned to previous DTO shapes.
- [ ] Publish transition checklist for downstream maintainers.

## Release Notes and Communication

- [ ] Draft release notes section for profile registry cutover.
- [ ] Add explicit breaking-changes section with required actions.
- [ ] Add upgrade matrix (`old -> new -> action`) for operators.
- [ ] Add compatibility floor (minimum versions across repos).
- [ ] Add rollback guidance and known limitations section.

## Validation and QA

- [ ] Run migration command on fixture corpus and capture outputs.
- [ ] Run migration command on one real legacy file sample.
- [ ] Start Pinocchio with migrated registry and verify profile CRUD/select flows.
- [ ] Start Go-Go-OS with migrated registry and verify profile CRUD/select flows.
- [ ] Verify docs command snippets execute as written.
- [ ] Capture QA transcript in ticket changelog.

## Operational Artifacts

- [ ] Add concise migration checklist page for on-call/deploy operators.
- [ ] Add FAQ section for common migration questions.
- [ ] Add support escalation notes (what logs/files to collect).
- [ ] Add final sign-off checklist for release approval.

## Closeout

- [ ] Run `docmgr doctor` and `docmgr validate frontmatter` for ticket docs.
- [ ] Ensure changelog includes migration command and docs updates.
- [ ] Link GP-21..GP-24 completion status in final release summary.
