# Release Guide

This repository uses Git tags and the `deploy` workflow for production releases.

## Versioning

Use Semantic Versioning labels in the form `vMAJOR.MINOR.PATCH`.

- `PATCH` for bug fixes, operational tweaks, and other backward-compatible changes
- `MINOR` for backward-compatible feature additions or larger behavior changes
- `MAJOR` for breaking changes that require coordination or migration

Examples:

- `v1.0.1` for a small fix
- `v1.1.0` for a larger but compatible feature release
- `v2.0.0` for a breaking change

If the project is still moving quickly, keep the tag history simple and bump only the part that changed. Do not invent pre-release labels unless the team needs them.

## Release Flow

1. Merge the change into `main`.
2. Pick the next version number.
3. Create an annotated tag on the release commit.
4. Push the tag to `origin`.
5. Watch the `deploy` workflow run, or start it manually with `workflow_dispatch` if you need to rerun a release.

Example:

```bash
git tag -a v1.0.1 -m "Release v1.0.1"
git push origin v1.0.1
```

## Manual Deploy

Use `workflow_dispatch` when you need to redeploy the current code without creating a new tag. That is useful for recovery, environment refreshes, or rerunning a failed deploy.
