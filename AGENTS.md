# AGENTS

Top-level contract for this repo. Keep detailed procedures in `.agents/rules` and repo facts in `.docs`.

## Plan Workflow

- Work through the files in `.plans/README.md` order.
- Treat each plan file as exactly one branch and one pull request.
- Before starting a new plan, confirm the previous plan has been validated and a pull request has been created.
- Do not batch multiple plans into a single branch or a single pull request.

## Per-Plan Execution

- Read the target plan file first and use its `Branch` value as the branch name.
- Use the plan file's `PR Title` value for the pull request title.
- Keep implementation scoped to that single plan.
- When a plan is implemented, run the validations listed in that plan, fix failures, then commit, push, and open a PR.
- Report the pull request URL/status to the user, then stop until that PR step is complete.

## Review Follow-Up

- After creating or updating a pull request, check reviews, review comments, and issue comments before continuing.
- Separate actionable comments from bot walkthroughs, summaries, and other non-blocking status messages.
- Treat skipped-review notices on stacked pull requests as informational unless they contain a concrete request.

## GitHub Operations

- Prefer `gh` for pull request, issue, review, and repository inspection when GitHub state matters.
- Use `25MPOOL/leetdaily` as the canonical GitHub repo slug for `gh` commands, even if `git remote -v` still shows the old moved location.
- Keep release tagging and deploy behavior details in [docs/release.md](docs/release.md).

## Safety

- If a tool is needed to reproduce the development environment and it is available in `aqua packages`, add it to the repository-managed `aqua` configuration instead of relying on ad hoc local installation.
- If the current worktree already contains changes from multiple plans, do not continue blindly; tell the user and ask whether to split the work or accept a combined PR.
