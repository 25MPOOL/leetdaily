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
- If validation passes, stage and commit the changes for that plan.
- Push the branch to `origin`.
- Create a pull request before moving on to the next plan.
- Report the pull request URL/status to the user, then stop. Start the next plan only after that per-plan PR step is complete.

## If The Worktree Is Already Mixed

- If the current worktree already contains changes from multiple plans, do not continue implementing more plans blindly.
- Tell the user that the worktree is mixed and ask whether to split the work or accept a combined PR before proceeding.
