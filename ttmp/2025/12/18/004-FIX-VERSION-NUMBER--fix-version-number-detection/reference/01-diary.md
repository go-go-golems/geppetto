---
Title: Diary
Ticket: 004-FIX-VERSION-NUMBER
Status: active
Topics:
    - versioning
    - git
    - releases
DocType: reference
Intent: long-term
Owners: []
RelatedFiles: []
ExternalSources: []
Summary: "Step-by-step investigation of version detection issues"
LastUpdated: 2025-12-18T14:31:21.920174096-05:00
---

# Diary

## Goal

Document the step-by-step investigation of why `svu current` shows v0.5.5 instead of v0.5.7, and why GitHub shows v0.4.21 as the latest release despite newer tags existing.

## Step 1: Initial Investigation - Checking Tag Existence

I started by verifying that tags v0.5.6 and v0.5.7 actually exist, and checking what `svu current` reports. The initial hypothesis was that this might be a worktree issue since geppetto is checked out as a git worktree.

**Commit (code):** N/A — investigation only

### What I did
- Listed git tags: `git tag --list | grep -E '^v[0-9]' | sort -V | tail -10`
- Ran `svu current` to see what it reports
- Checked git worktree status: `git worktree list`
- Verified git directory location: `git rev-parse --git-dir`

### Why
- Needed to confirm tags exist before investigating why they're not detected
- Worktree hypothesis needed verification since worktrees can have tag visibility issues

### What worked
- Confirmed tags v0.5.6 and v0.5.7 exist in the repository
- Confirmed this is a worktree checkout (git dir at `/home/manuel/code/wesen/corporate-headquarters/.git/modules/geppetto/worktrees/geppetto25`)
- `svu current` returned `v0.5.5` despite newer tags existing

### What didn't work
- Initial assumption that worktree was the issue — tags are visible, so worktree isn't the problem

### What I learned
- Tags exist but aren't being detected by `svu current`
- Worktree setup doesn't prevent tag visibility (tags are shared across worktrees)

### What was tricky to build
- N/A — investigation step

### What warrants a second pair of eyes
- N/A — investigation step

### What should be done in the future
- N/A — investigation step

### Code review instructions
- N/A — investigation step

### Technical details
```bash
$ git tag --list | grep v0.5 | tail -5
v0.5.3
v0.5.4
v0.5.5
v0.5.6
v0.5.7

$ svu current
v0.5.5

$ git describe --tags --abbrev=0
v0.5.5
```

### What I'd do differently next time
- N/A — investigation step

## Step 2: Understanding How svu Works

I researched how `svu current` determines the version and discovered it uses `git describe` internally. This led me to check tag reachability from the current HEAD.

**Commit (code):** N/A — investigation only

### What I did
- Searched web for "svu current git tags how does it work"
- Checked what commit tags point to: `git show-ref --tags | grep v0.5.7`
- Checked current HEAD: `git log --oneline --decorate -10`
- Checked if tagged commit is reachable: `git merge-base --is-ancestor f39a118 HEAD`
- Listed tags containing HEAD: `git tag --contains HEAD`

### Why
- Needed to understand the mechanism `svu` uses to determine current version
- Suspected tags might not be reachable from current branch

### What worked
- Confirmed `svu` uses `git describe --tags --abbrev=0` internally
- Discovered tags v0.5.6 and v0.5.7 point to commit `f39a118`
- Confirmed `f39a118` is NOT an ancestor of current HEAD (`30e1c9c`)
- Confirmed no v0.5.x tags contain the current HEAD

### What didn't work
- N/A — investigation successful

### What I learned
- `git describe` only considers tags reachable from the current commit
- Tags on parallel branches won't be detected by `git describe` when on a different branch
- This is the root cause: tags v0.5.6/v0.5.7 are on branch `task/fix-pinocchio-profiles` which hasn't been merged to the current branch

### What was tricky to build
- Understanding git's tag reachability semantics — tags must be in the ancestry chain to be considered by `git describe`

### What warrants a second pair of eyes
- Verification that tags should be on main branch commits, not feature branch commits

### What should be done in the future
- Establish tagging policy: tags should be placed on main branch commits (or after merging to main)
- Consider branch-aware version detection if feature branch tags are intentional

### Code review instructions
- Review git log output showing branch structure:
```bash
$ git log --oneline --all --decorate --graph | grep v0.5
| | * f39a118 (tag: v0.5.7, tag: v0.5.6, task/fix-pinocchio-profiles)
* 33b3329 (tag: v0.5.5, task/revive-jesus, ...)
```

### Technical details
```bash
$ git show-ref --tags | grep v0.5.7
f39a1183a014e11e5a16f231e40a80cddcfac185 refs/tags/v0.5.7

$ git merge-base --is-ancestor f39a118 HEAD && echo "ancestor" || echo "NOT ancestor"
NOT ancestor

$ git tag --contains HEAD | grep v0.5
(no output - no v0.5.x tags contain HEAD)
```

### What I'd do differently next time
- Check tag reachability earlier in the investigation

## Step 3: Investigating GitHub Release Discrepancy

I investigated why GitHub shows v0.4.21 as the latest release when tags v0.5.x exist. This required understanding the difference between git tags and GitHub releases.

**Commit (code):** N/A — investigation only

### What I did
- Navigated to GitHub releases page: https://github.com/go-go-golems/geppetto/releases
- Checked remote tags: `git ls-remote --tags origin | grep v0.5`
- Searched web for information about GitHub releases vs tags
- Examined git log to see where tags are in branch history

### Why
- Needed to understand why GitHub doesn't recognize newer tags as releases
- User mentioned "doesn't pushing a tag handle that?" — needed to clarify the difference

### What worked
- Confirmed tags v0.5.6 and v0.5.7 exist on remote (`git ls-remote` showed them)
- Discovered GitHub releases are separate from git tags
- Confirmed GitHub shows v0.4.21 as latest release (from browser snapshot)

### What didn't work
- N/A — investigation successful

### What I learned
- **Git tags ≠ GitHub releases**: Pushing a tag creates a tag in the repository, but does NOT create a GitHub release
- GitHub releases must be created manually via GitHub UI or API
- GitHub's "latest release" is determined by the most recent **release**, not the most recent **tag**
- Releases can be manually marked as "latest" regardless of tag dates

### What was tricky to build
- Understanding the distinction between git tags (version markers) and GitHub releases (public announcements with release notes)

### What warrants a second pair of eyes
- Verification that v0.5.x versions should have GitHub releases created

### What should be done in the future
- Create GitHub releases for all v0.5.x tags that should be public releases
- Establish release process: tag + create GitHub release as part of release workflow
- Consider automating release creation via GitHub Actions when tags are pushed

### Code review instructions
- Review `.github/workflows/release.yml` to see current release process
- Check if release workflow should be triggered on tag push

### Technical details
```bash
$ git ls-remote --tags origin | grep v0.5.7
f39a1183a014e11e5a16f231e40a80cddcfac185	refs/tags/v0.5.7
```

GitHub releases page shows:
- Latest release: v0.4.21
- No releases for v0.5.x versions (tags exist but no releases)

### What I'd do differently next time
- Check GitHub releases API or UI earlier to confirm the discrepancy

## Step 4: Creating Analysis Document and Diary

Created structured documentation of findings using docmgr, following the project's documentation workflow.

**Commit (code):** N/A — documentation only

### What I did
- Created ticket: `docmgr ticket create-ticket --ticket 004-FIX-VERSION-NUMBER`
- Created analysis document: `docmgr doc add --doc-type analysis`
- Created diary document: `docmgr doc add --doc-type reference --title "Diary"`
- Wrote analysis document with root cause analysis, evidence, and solutions
- Wrote diary entries documenting each investigation step

### Why
- User requested analysis document creation
- Following project's docmgr workflow for structured documentation
- Diary format captures step-by-step investigation process

### What worked
- docmgr created proper directory structure and frontmatter
- Analysis document captures all findings clearly
- Diary entries follow the required format with all sections

### What didn't work
- N/A — documentation creation successful

### What I learned
- docmgr workflow provides good structure for investigation documentation
- Diary format forces reflection on each step, making the investigation process transparent

### What was tricky to build
- Ensuring diary entries follow the exact format with all required sections
- Balancing detail with conciseness in analysis document

### What warrants a second pair of eyes
- Review analysis document for completeness and accuracy
- Verify solutions proposed are appropriate for the project's workflow

### What should be done in the future
- Update changelog when solutions are implemented
- Create tasks for implementing recommended solutions
- Determine if v0.5.6/v0.5.7 tags should be moved to main branch commits

### Code review instructions
- Review analysis document: `geppetto/ttmp/2025/12/18/004-FIX-VERSION-NUMBER--fix-version-number-detection/analysis/01-version-detection-analysis.md`
- Review diary: `geppetto/ttmp/2025/12/18/004-FIX-VERSION-NUMBER--fix-version-number-detection/reference/01-diary.md`

### Technical details
- Ticket created at: `geppetto/ttmp/2025/12/18/004-FIX-VERSION-NUMBER--fix-version-number-detection/`
- Analysis document: `analysis/01-version-detection-analysis.md`
- Diary document: `reference/01-diary.md`

### What I'd do differently next time
- Relate files immediately after creating documents

## Step 5: Relating Files and Updating Changelog

Related relevant files to the documentation and updated the changelog to complete the documentation workflow.

**Commit (code):** N/A — documentation only

### What I did
- Related Makefile and release.yml to ticket index: `docmgr doc relate --ticket 004-FIX-VERSION-NUMBER`
- Related same files to analysis document: `docmgr doc relate --doc analysis/01-version-detection-analysis.md`
- Updated changelog: `docmgr changelog update --ticket 004-FIX-VERSION-NUMBER`

### Why
- Following docmgr workflow: relate files that are relevant to the investigation
- Changelog documents what was done in this ticket

### What worked
- Files successfully related to both ticket index and analysis document
- Changelog updated with summary of work

### What didn't work
- N/A — all operations successful

### What I learned
- docmgr allows relating files to both ticket index and specific documents
- File notes should explain why the file matters (not just "touched")

### What was tricky to build
- N/A — straightforward docmgr operations

### What warrants a second pair of eyes
- Verify file relationships are appropriate
- Review changelog entry for completeness

### What should be done in the future
- Review related files to verify they're correctly linked
- Create tasks for implementing solutions proposed in analysis document

### Code review instructions
- Check ticket index for related files
- Review analysis document frontmatter for RelatedFiles section

### Technical details
```bash
$ docmgr doc relate --ticket 004-FIX-VERSION-NUMBER \
  --file-note "/abs/path/to/Makefile:Uses svu current" \
  --file-note "/abs/path/to/release.yml:Release workflow"
```

### What I'd do differently next time
- N/A — workflow followed correctly

## Step 6: Analyze GitHub Workflows for Release Triggering

This step focused on understanding why GitHub shows an old “latest release” and why new tags don’t automatically appear as releases. The key was to separate “tag pushes” from “GitHub Releases” and then confirm whether any workflow still creates releases on tag events.

**Commit (code):** N/A — investigation only

### What I did
- Listed workflows under `geppetto/.github/workflows/`
- Read `geppetto/.github/workflows/release.yml` to confirm its `on:` trigger configuration
- Read `geppetto/.github/workflows/lint.yml` and `push.yml` to see what still triggers on tag pushes

### Why
- The most plausible explanation for “last release is old” is that no automation is currently creating GitHub Releases on tag pushes.

### What worked
- Confirmed `release.yml` is currently `workflow_dispatch` only (tag trigger is commented out).
- Confirmed `lint.yml` still triggers on `push` tags matching `v*`, so tag pushes do run CI.

### What didn't work
- N/A — investigation successful

### What I learned
- Tags can exist and CI can run, yet GitHub “Releases” can remain stale if no workflow (or human) creates releases.

### What was tricky to build
- N/A — mostly reading config, but required careful differentiation between “tags” and “releases”.

### What warrants a second pair of eyes
- Verify we actually want a GitHub Release created for every `v*` tag, or only some tags.

### What should be done in the future
- Decide policy: “tags imply GitHub release” vs “tags are Go module versions only”.

### Code review instructions
- Start with `geppetto/.github/workflows/release.yml` and confirm the commented out tag trigger.
- Compare with `geppetto/.github/workflows/lint.yml` which still runs on tags.

### Technical details
Key snippet (current state):

```yaml
on:
  workflow_dispatch:
#  push:
#    tags:
#      - '*'
```

## Step 7: Check Git History for Why Tag Releases Were Disabled

This step drilled into the git history of `release.yml` to find the exact point where tag-triggered releases were disabled, and whether there was any hint of build/release friction (e.g. OSX cross compilation).

**Commit (code):** N/A — investigation only

### What I did
- Ran `git log -- .github/workflows/release.yml` and inspected key commits
- Inspected commit `70b2a2b` which disables the tag trigger
- Noted that the same commit also removed a `lefthook` pre-push `release` command

### Why
- The “why” is usually embedded in commit history when workflows change intentionally.

### What worked
- Found the exact commit that disabled the trigger: `70b2a2b` (“Disable github release action”).
- Found follow-up commit `5bfcee1` that reintroduced the workflow in a “safe” manual-only state.

### What didn't work
- The commit message doesn’t explicitly explain the failure mode (only that it was disabled).

### What I learned
- This wasn’t an accidental regression: it was intentionally turned off.

### What was tricky to build
- N/A — mostly git archaeology.

### What warrants a second pair of eyes
- Confirm whether the original motivation was “avoid broken GoReleaser builds” or “avoid publishing binaries”.

### What should be done in the future
- Capture an explicit rationale in docs/commit message when disabling release automation (helps future debugging).

### Code review instructions
- Review commit `70b2a2b` diff for `release.yml` and `lefthook.yml`.

### Technical details
Relevant diff excerpt:

```diff
-on:
-  push:
+#on:
+#  push:
     # run only against tags
-    tags:
-      - '*'
+#    tags:
+#      - '*'
```

## Step 8: Determine Whether Missing Binaries Prevent Releases (GoReleaser Config Review)

This step answered the “is it because we don’t have a binary?” part by checking what GoReleaser is configured to build and whether that matches the repo’s actual `cmd/` structure.

**Commit (code):** N/A — investigation only

### What I did
- Read `geppetto/.goreleaser.yaml`
- Listed `geppetto/cmd/` directories to see what main packages exist
- Searched for any `cmd/geppetto` directory (none found)

### Why
- If GoReleaser is configured to build a non-existent binary/main package, tag-triggered releases would fail even if re-enabled.

### What worked
- Confirmed the repo has multiple `package main` entrypoints (e.g. `cmd/llm-runner`, `cmd/turnsdatalint`, `cmd/geppetto-lint`) but **no** `cmd/geppetto`.
- Confirmed `.goreleaser.yaml` does not specify a `main:` path, which is suspicious for a multi-cmd repo.

### What didn't work
- N/A — investigation successful, but it indicates the current GoReleaser config likely isn’t aligned with the repo.

### What I learned
- The “no binary” hypothesis is partially right: there is no `geppetto` CLI main package, and the current GoReleaser config doesn’t clearly target an existing main package.

### What was tricky to build
- N/A — just correlating config with directory layout.

### What warrants a second pair of eyes
- Decide which binaries (if any) should be shipped as release assets:
  - `llm-runner`?
  - `turnsdatalint`?
  - `geppetto-lint`?
  - none (release notes only)?

### What should be done in the future
- If we want binary releases: update `.goreleaser.yaml` to define builds with explicit `main: ./cmd/...` entries.

### Code review instructions
- Review `geppetto/.goreleaser.yaml` and compare to `geppetto/cmd/*` structure.

### Technical details
Observed `cmd/` layout:
- `geppetto/cmd/llm-runner/`
- `geppetto/cmd/turnsdatalint/`
- `geppetto/cmd/geppetto-lint/`
- No `geppetto/cmd/geppetto/`

## Step 9: Implement Artifact-Free GitHub Release Creation on Tags (and Manual Backfill)

To answer “can we still trigger some kind of artifact free release?”, I added a lightweight workflow that creates a GitHub Release from a tag **without building/uploading binaries**. This keeps the GitHub Releases page aligned with tags, even if GoReleaser stays manual/off.

**Commit (code):** N/A — local change only (workflow added)

### What I did
- Added `geppetto/.github/workflows/tag-release-notes.yml`
  - Triggers on `push` of tags `v*`
  - Also supports `workflow_dispatch` with a `tag` input for manual backfill of existing tags
  - Uses `softprops/action-gh-release` with `generate_release_notes: true`

### Why
- GitHub “Releases” requires explicit release creation; tags alone don’t populate the Releases page.
- This avoids GoReleaser complexity (OSXCross, signing, build issues) when we just want release metadata.

### What worked
- The workflow is minimal and should work with only `contents: write` and `GITHUB_TOKEN`.

### What didn't work
- N/A — implementation is straightforward, not executed here (needs GitHub Actions runtime).

### What I learned
- “Artifact-free release” is a clean separation: tags + Go module versioning can remain unchanged while GitHub Releases is kept in sync.

### What was tricky to build
- Avoiding accidental coupling to GoReleaser: we keep the existing `release.yml` manual-only and add a separate workflow.

### What warrants a second pair of eyes
- Confirm the workflow expression for `tag_name` is acceptable in GitHub Actions YAML:
  - `${{ github.event_name == 'workflow_dispatch' && inputs.tag || github.ref_name }}`

### What should be done in the future
- Optionally backfill existing tags (e.g. `v0.5.7`) by running the workflow manually via GitHub UI.
- If binary releases are desired later, revisit `.goreleaser.yaml` + `release.yml`.

### Code review instructions
- Review new workflow: `geppetto/.github/workflows/tag-release-notes.yml`.
- Confirm it does not upload assets and only creates the GitHub Release entry.

### Technical details
Workflow behavior:
- **Tag push**: creates a Release for that tag with autogenerated notes
- **Manual dispatch**: creates a Release for the provided tag name (useful for `v0.5.x` backfill)
