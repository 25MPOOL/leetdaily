# Plan workflow for agents
Source of truth: `AGENTS.md`, `.plans/README.md`, `.plans/*.md`

This repo is plan-driven. Work one plan at a time.

- Follow `.plans/README.md` in order.
- Treat each plan file as exactly one branch and one pull request.
- Read the target plan file first, then use its `Branch` value as the branch name.
- Use the plan file's `PR Title` value for the pull request title.
- Keep implementation scoped to that plan unless the plan explicitly reaches into later work.
- Run the plan's validation commands before commit, push, and PR creation.
- After PR creation or update, check reviews, review comments, and issue comments.

The plan file is the source for execution details; this file is only a navigation summary.
