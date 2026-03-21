# Repo facts for agents
Source of truth: `AGENTS.md`, `.plans/README.md`, `docs/release.md`, `docs/runbook.md`

This repo is organized around a staged maintainability plan for a Go service named `leetdaily`.

- The active plan list lives in `.plans/README.md`.
- The repo uses `25MPOOL/leetdaily` for `gh` commands.
- Release tags follow Semantic Versioning and feed the `deploy` workflow.
- `docs/release.md` owns release tagging and deploy flow details.
- `docs/runbook.md` owns local setup, deploy recovery, secret rotation, and incident checks.
- `infra/` contains the Terraform and deployment stack.
- `cmd/` and `internal/` contain the application code.

Use this file as the fast index. If a fact needs procedure-level detail, follow the source document instead.
