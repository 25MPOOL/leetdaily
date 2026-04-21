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

## Release Flow for Infra Changes

If the merged PR includes changes under `infra/terraform/**`, the Cloud Run Job (and any other managed resources) must be updated before the new image is deployed. Run Terraform first, then tag.

1. Merge the change into `main`.
2. Trigger `terraform-apply` from the Actions tab (or `gh workflow run terraform-apply.yaml --ref main`).
3. Open the `terraform-apply` run log and verify the `Plan:` line and the resources listed under `will be created / will be destroyed / will be updated` match what the PR intended. The `terraform-plan` workflow is skipped on PRs in this repo (see below), so this log is the only plan review step.
4. Once `Apply complete!` appears without errors, proceed with the normal tag-based release flow above.

If the apply reports unexpected resource drift, cancel the run before it finishes and investigate; do not tag a release on top of a questionable infra state.

## Why terraform-plan is skipped on PRs

The `terraform-plan` workflow requires `GCP_TERRAFORM_PLAN_WORKLOAD_IDENTITY_PROVIDER` to be set on the repository. Until that variable is provisioned, the workflow reports `configured=false` and exits early with a skipped status. The `terraform-apply` workflow uses a separate `GCP_TERRAFORM_APPLY_WORKLOAD_IDENTITY_PROVIDER` variable (which is configured) and runs plan + apply in the same job, so the plan output is available in the apply run log.

## Manual Deploy

Use `workflow_dispatch` when you need to redeploy the current code without creating a new tag. That is useful for recovery, environment refreshes, or rerunning a failed deploy.
