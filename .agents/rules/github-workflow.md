# GitHub Workflow Rules

## Canonical Repository

- Use `25MPOOL/leetdaily` for `gh` commands.
- Do not rely on the current `origin` URL as the source of truth because GitHub reports that the repository has moved from `nkoji21/leetdaily`.

## PR Review Checks

- After every PR creation or update, inspect all three surfaces:
- Pull request reviews: `gh api repos/25MPOOL/leetdaily/pulls/<number>/reviews`
- Pull request line comments: `gh api repos/25MPOOL/leetdaily/pulls/<number>/comments`
- Pull request issue comments: `gh api repos/25MPOOL/leetdaily/issues/<number>/comments`
- Treat walkthrough comments, skipped-review notices, and generic bot summaries as informational unless they contain a concrete request.

## Stacked PR Caveat

- Stacked PRs that target a non-default branch may not receive automatic bot review.
- CodeRabbit currently leaves a "Review skipped" issue comment on such PRs instead of creating actual review findings.
- Do not wait for auto-review on stacked PRs forever. Check whether the base branch is the default branch and interpret the comment accordingly.
