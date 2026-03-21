# Repo glossary for agents
Source of truth: `AGENTS.md`, `.plans/*.md`, `docs/runbook.md`, `docs/release.md`

Keep these terms aligned with the repo's actual workflow.

- `plan`: one implementation unit from `.plans/*.md` with its own branch and PR.
- `validation`: the checks listed in the target plan before commit and PR creation.
- `PR`: the reviewable change for one plan, created after validation passes.
- `mixed worktree`: a working tree that already contains unrelated changes and should not be extended blindly.
- `canonical slug`: `25MPOOL/leetdaily`, the GitHub repository slug to use with `gh`.
- `runbook`: the operational guide in `docs/runbook.md`.
- `release`: the versioning and tag flow in `docs/release.md`.

Use the term that matches the file that owns the truth. Do not redefine terms in multiple places.
