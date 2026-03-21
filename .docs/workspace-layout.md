# Workspace layout for agents
Source of truth: repository structure, `AGENTS.md`, `.plans/README.md`

Use this to map the repo before making changes.

- `.plans/` contains the ordered implementation plans and the canonical branch / PR metadata.
- `.agents/` contains agent-facing behavior rules and workflow guidance.
- `.docs/` contains short agent-facing summaries and navigation aids.
- `docs/` contains human-facing operational docs such as `docs/release.md` and `docs/runbook.md`.
- `infra/` contains Terraform and deployment infrastructure.
- `cmd/` and `internal/` contain the Go service code.

Do not duplicate detailed procedures across layers. Keep summaries in `.docs/` and detailed instructions in `docs/`.
