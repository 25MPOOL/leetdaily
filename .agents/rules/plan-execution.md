# Plan Execution Rules

Use these rules when working from `.plans/README.md`.

## Plan Order

- Work through the files in `.plans/README.md` order.
- Treat each plan file as exactly one branch and one pull request.
- Do not batch multiple plans into a single branch or PR.

## Per-Plan Inputs

- Read the target plan file first.
- Use the plan file's `Branch` value as the branch name.
- Use the plan file's `PR Title` value for the pull request title.
- Keep implementation scoped to that plan only.

## Done Sequence

- Run the validations listed in the plan.
- Fix failures and rerun validation until it passes.
- Stage and commit the changes for that plan.
- Push the branch to `origin`.
- Create the pull request before moving on.
- Report the PR URL and status, then stop until that PR step is complete.
