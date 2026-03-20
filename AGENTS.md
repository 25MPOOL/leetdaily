# AGENTS

## Plan Workflow

- Work through the files in `.plans/README.md` order.
- Treat each plan file as exactly one branch and one pull request.
- Before starting a new plan, make sure the previous plan has already been validated and a pull request has been created.
- Do not batch multiple plans into a single branch or a single pull request.

## Per-Plan Execution Rules

- Read the target plan file first and use its `Branch` value as the branch name.
- Use the plan file's `PR Title` value for the pull request title.
- Keep implementation scoped to that single plan. Do not pull work from later plans unless the current plan explicitly requires it.

## Done Procedure

- When a plan is implemented, run the validations listed in that plan.
- If validation fails, fix the issues and re-run validation until it passes.
- If validation passes, stage and commit the changes for that plan.
- Push the branch to `origin`.
- Create a pull request before moving on to the next plan.
- Report the pull request URL/status to the user, then stop. Start the next plan only after that per-plan PR step is complete.

## Review Follow-Up

- After creating or updating a pull request, check its reviews, review comments, and issue comments before continuing.
- Separate actionable comments from bot walkthroughs, summaries, and other non-blocking status messages.
- For stacked pull requests, note that review bots may skip auto-review when the base branch is not the default branch.
- If a stacked pull request was skipped by automation, treat that as informational and do not confuse it with an approval or a requested change.

## GitHub Operations

- Prefer `gh` for pull request, issue, review, and repository inspection when GitHub state matters.
- Use `25MPOOL/leetdaily` as the canonical GitHub repo slug for `gh` commands, even if `git remote -v` still shows the old moved location.

## Merge Strategy

- Choose the merge method based on the quality of the branch history, not only the commit count.
- If a PR has a single clean commit, prefer squash merge.
- If a PR has multiple commits and each commit is meaningful, reviewable, and worth preserving, prefer a merge commit.
- If a PR has multiple commits but the history includes fixup, WIP, noisy, or otherwise low-signal commits, prefer squash merge instead of preserving that history.
- Before merging, quickly inspect the commit list and use the merge method that leaves the main branch history easiest to read later.

## If The Worktree Is Already Mixed

- If the current worktree already contains changes from multiple plans, do not continue implementing more plans blindly.
- Tell the user that the worktree is mixed and ask whether to split the work or accept a combined PR before proceeding.
