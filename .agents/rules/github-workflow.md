# GitHub Workflow Rules

Use these rules when GitHub state matters.

## Canonical Repository

- Use `25MPOOL/leetdaily` for `gh` commands.
- Do not treat the current `origin` URL as the source of truth if it differs from the canonical slug.

## Pull Request Checks

- After every PR creation or update, inspect all three surfaces:
- PR reviews: `gh api repos/25MPOOL/leetdaily/pulls/<number>/reviews`
- PR line comments: `gh api repos/25MPOOL/leetdaily/pulls/<number>/comments`
- PR issue comments: `gh api repos/25MPOOL/leetdaily/issues/<number>/comments`
- Treat walkthrough comments, skipped-review notices, and generic bot summaries as informational unless they contain a concrete request.

## Stacked PRs

- Stacked PRs that target a non-default branch may not receive automatic bot review.
- A "Review skipped" comment is informational, not approval and not a request for changes.
- Check the base branch before waiting on automation.

## Merge Strategy

- Choose the merge method based on branch history quality, not only commit count.
- Prefer squash merge for a single clean commit or for noisy multi-commit history.
- Prefer a merge commit only when multiple commits are individually meaningful and worth preserving.
- Check the commit list before merging and pick the method that leaves `main` easiest to read later.
