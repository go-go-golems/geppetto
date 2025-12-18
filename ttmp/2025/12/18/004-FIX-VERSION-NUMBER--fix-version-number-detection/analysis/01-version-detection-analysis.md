---
Title: Version Detection Analysis
Ticket: 004-FIX-VERSION-NUMBER
Status: active
Topics:
    - versioning
    - git
    - releases
DocType: analysis
Intent: long-term
Owners: []
RelatedFiles:
    - Path: geppetto/.github/workflows/release.yml
      Note: GitHub Actions release workflow (currently manual trigger only)
    - Path: geppetto/.github/workflows/tag-release-notes.yml
      Note: New workflow for artifact-free releases
    - Path: geppetto/.goreleaser.yaml
      Note: GoReleaser config review for binary release feasibility
    - Path: geppetto/Makefile
      Note: Contains svu current usage in release target
ExternalSources: []
Summary: Analysis of why svu current shows v0.5.5 instead of v0.5.7, and why GitHub shows v0.4.21 as latest release
LastUpdated: 2025-12-18T14:31:20.605429509-05:00
---



# Version Detection Analysis

## Problem Statement

1. **svu current shows v0.5.5** when tags v0.5.6 and v0.5.7 exist
2. **GitHub shows v0.4.21 as latest release** despite newer tags existing

## Root Cause Analysis

### Issue 1: svu current Returns Wrong Version

**Root Cause:** Tags v0.5.6 and v0.5.7 point to commit `f39a118` which is on branch `task/fix-pinocchio-profiles`. This commit is **not an ancestor** of the current HEAD (`30e1c9c` on branch `task/add-turns-linting`).

**How svu works:**
- `svu current` uses `git describe --tags --abbrev=0` internally
- `git describe` only considers tags that are **reachable** from the current HEAD
- Since `f39a118` (v0.5.6/v0.5.7) is not in the ancestry chain of HEAD, git describe falls back to v0.5.5

**Evidence:**
```bash
$ git tag --list | grep v0.5 | tail -3
v0.5.5
v0.5.6
v0.5.7

$ git describe --tags --abbrev=0
v0.5.5

$ git merge-base --is-ancestor f39a118 HEAD && echo "ancestor" || echo "NOT ancestor"
NOT ancestor

$ git log --oneline --all --decorate | grep v0.5
f39a118 (tag: v0.5.7, tag: v0.5.6, task/fix-pinocchio-profiles) :ambulance: :sparkles: Load new config file format
33b3329 (tag: v0.5.5, task/revive-jesus, ...) Merge pull request #222
```

**Branch Structure:**
```
*   30e1c9c (HEAD -> task/add-turns-linting, origin/main)
|\  
| * [commits on task/add-turns-linting branch]
|/  
*   5545eff [merge base]
|\  
| * [other branches]
|/  
*   33b3329 (tag: v0.5.5) [on main branch]
*   ...
*   f39a118 (tag: v0.5.7, tag: v0.5.6) [on task/fix-pinocchio-profiles branch]
```

### Issue 2: GitHub Shows v0.4.21 as Latest Release

**Root Cause:** GitHub releases are **separate from git tags**. Simply pushing a tag does not create a GitHub release. The last GitHub release was manually created for v0.4.21, and no releases have been created for v0.5.x versions.

**Evidence:**
- GitHub releases page shows v0.4.21 as latest
- Tags v0.5.0 through v0.5.7 exist in the repository but are not associated with GitHub releases
- Tags exist but releases don't

**How GitHub Releases Work:**
1. Pushing a tag (`git push origin --tags`) creates a tag in the repository
2. Creating a GitHub release associates a tag with release notes and marks it as a release
3. GitHub's "latest release" is determined by the most recent **release**, not the most recent **tag**
4. Releases can be manually marked as "latest" regardless of tag dates

### Issue 2a: Why GitHub Releases Don’t Auto-Trigger Anymore (Workflow Analysis)

**Root Cause:** `geppetto/.github/workflows/release.yml` is currently **manual-only** (`workflow_dispatch`) and no longer triggers on `push` to tags.

**Evidence (git history):**
- Commit `70b2a2b` “Disable github release action” commented out the `on: push: tags:` section.
- Commit `5bfcee1` later reintroduced the workflow but kept it manual (`workflow_dispatch`) so it wouldn’t error while also not running automatically.

**Additional context from current workflows:**
- `geppetto/.github/workflows/lint.yml` still runs on `push` tags matching `v*`, so tags do trigger CI — just not the release publication workflow.

### Issue 2b: Is It Because We “Don’t Have a Binary”?

Not exactly — but **the current GoReleaser setup does not clearly define a releasable main package**:
- `geppetto/.goreleaser.yaml` defines a build output named `geppetto`, but does **not** specify which `main` package to build (and the repo does not currently have `cmd/geppetto/`).
- The existing workflow also tries to do cross-platform work (OSXCross + signing), which historically had build friction (commit history shows multiple OSX build workarounds).

So even if we re-enabled tag-triggered GoReleaser, it’s likely to be noisy until `.goreleaser.yaml` is updated to match the actual commands you want to ship (e.g. `cmd/llm-runner`, `cmd/turnsdatalint`, `cmd/geppetto-lint`, etc.).

### Artifact-Free Release Option (Recommended Quick Fix)

We can still keep GitHub’s “Releases” page up to date **without building binaries** by creating a release entry for a tag:
- This produces an “artifact-free” GitHub Release (no uploaded assets), and GitHub automatically provides “Source code (zip/tar.gz)”.
- A lightweight workflow can run on tag pushes and also support manual backfill for an existing tag.

Implementation added in this ticket:
- `geppetto/.github/workflows/tag-release-notes.yml`: Creates/updates a GitHub Release using `softprops/action-gh-release` with `generate_release_notes: true`.

## Worktree Context

The geppetto repository is checked out as a **git worktree**:
- Main git directory: `/home/manuel/code/wesen/corporate-headquarters/.git/modules/geppetto`
- Worktree location: `/home/manuel/workspaces/2025-12-01/integrate-moments-persistence/geppetto`
- Worktree git dir: `/home/manuel/code/wesen/corporate-headquarters/.git/modules/geppetto/worktrees/geppetto25`

This is **not** the cause of the issue - tags are properly shared across worktrees. The issue is purely about tag reachability from the current branch.

## Impact

1. **Version detection in CI/CD:** `svu current` will return incorrect version when run on branches that don't include v0.5.6/v0.5.7 in their history
2. **Release visibility:** Users looking at GitHub releases won't see v0.5.x versions
3. **Version consistency:** Different branches may report different "current" versions

## Solutions

### Solution 1: Fix Tag Placement (Recommended)

Ensure tags are placed on commits that are part of the main branch history:

1. Identify the merge base or main branch commit where v0.5.6/v0.5.7 should have been tagged
2. Delete and recreate tags on the correct commits:
   ```bash
   git tag -d v0.5.6 v0.5.7
   git tag v0.5.6 <correct-commit>
   git tag v0.5.7 <correct-commit>
   git push origin --tags --force
   ```

### Solution 2: Create GitHub Releases

For each v0.5.x tag that should be a release:
1. Go to GitHub Releases page
2. Click "Draft a new release"
3. Select the tag (e.g., v0.5.7)
4. Add release notes
5. Check "Set as the latest release" for the most recent one
6. Publish

### Solution 3: Use Branch-Specific Version Detection

If tags on feature branches are intentional, modify version detection to:
- Check tags on the current branch first
- Fall back to main branch tags
- Or explicitly specify which branch to use for version detection

## Recommendations

1. **Immediate:** Determine if v0.5.6/v0.5.7 tags are on the correct commits
2. **Short-term:** Create GitHub releases for v0.5.x versions that should be public releases
3. **Long-term:** Establish a release process that:
   - Tags commits on main branch (or after merging to main)
   - Creates GitHub releases for all public versions
   - Uses consistent versioning strategy across branches

## Related Files

- `geppetto/Makefile`: Contains `svu current` usage in release target
- `.github/workflows/release.yml`: GitHub Actions workflow for releases

