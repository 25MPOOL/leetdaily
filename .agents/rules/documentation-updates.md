# Documentation Update Rules

Use these rules when adding or changing docs.

## Keep Truth Single

- Prefer existing locations over creating new top-level docs.
- Do not duplicate the same fact in multiple files unless one copy is clearly a summary and the other is the primary source.
- Keep human-facing detail in `docs/` and agent-facing summaries in `.docs/`.

## `.docs` Policy

- Treat `.docs` as concise summaries and navigation aids.
- Each `.docs` file should point to its source of truth.
- Start `.docs` files with a short purpose line and a `Source of truth:` line.
- Use `.docs` to explain where to look, not to replace the primary documentation.

## Update Discipline

- If a change affects release or runbook behavior, update the primary document first.
- If the change introduces a new repo fact, add it to the narrowest existing doc layer that owns that fact.
- Before creating a new top-level documentation file, confirm that the content cannot fit into an existing `docs/` or `.docs/` file.
