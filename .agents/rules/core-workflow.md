# Core Workflow Rules

Use these rules for all repository changes.

## Explore First

- Inspect the relevant files before changing anything.
- Prefer `rg` and `rg --files` when searching the repo.
- Read the smallest set of files that establishes the current behavior, existing patterns, and affected docs.

## Protect Existing Work

- Assume the worktree may already contain unrelated user changes.
- Do not revert, rewrite, or clean up changes outside the task scope.
- If a change would touch an unrelated area, stop and confirm the scope instead of widening it silently.

## Safe Editing

- Use `apply_patch` for manual edits.
- Avoid destructive git operations such as `git reset --hard` and `git checkout --`.
- Prefer small, targeted edits that preserve surrounding behavior.
- If a change is likely to conflict with uncommitted work, stop and reassess before editing.
