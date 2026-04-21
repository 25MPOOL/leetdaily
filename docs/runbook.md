# Runbook

## Local setup

1. Install tools with `aqua i`.
2. Prepare `config.json`, `guilds.json`, `state.json`, and `problems.json` locally or use GCS-backed objects.
3. Set required env vars:
   - `DISCORD_BOT_TOKEN`
   - `LEETDAILY_RUNTIME=job` for the normal one-shot execution path
   - `LEETDAILY_RUNTIME=http` only when you need the optional local `/run` and `/healthz` endpoints
   - `PORT` when using HTTP mode
   - `GCS_BUCKET` plus `CONFIG_OBJECT`, `GUILDS_OBJECT`, `STATE_OBJECT`, `PROBLEMS_OBJECT` for production-style storage
4. Run `go test ./...` and `go build ./cmd/leetdaily`.
5. Start locally with `go run ./cmd/leetdaily`.

### Manual guild configuration

1. Edit `guilds.json` or the configured GCS guild object directly.
2. Set the single server's `guild_id`, `forum_channel_id`, `notification_channel_id`, `enabled`, and `start_problem_number`.
3. Confirm the JSON object contains the expected channel IDs before the next run.

## Deploy

Production deploys are triggered by pushing a `v*` tag. The `deploy` workflow builds and pushes the container image, then runs `gcloud run jobs update` against the Cloud Run Job. Infra changes are applied separately by the `terraform-apply` workflow.

1. If the release includes changes under `infra/terraform/**`, trigger `terraform-apply` first and confirm `Apply complete!` in the run log.
2. Create an annotated `v*` tag on the release commit and push it to `origin`. The `deploy` workflow starts automatically on the tag push.
3. Watch the `deploy` workflow: it builds the image tagged with the commit SHA, pushes it to Artifact Registry, and calls `gcloud run jobs update` to point the Cloud Run Job at the new image.
4. Confirm Cloud Scheduler fires the Job on schedule (or trigger a one-shot execution with `gcloud run jobs execute <name> --region <region>`) and check Cloud Logging for a clean run.

See [docs/release.md](docs/release.md) for versioning, tag creation, and the infra-change ordering rule.

## Secret rotation

1. Update the Discord bot token in Secret Manager.
2. Redeploy or update the Cloud Run job if needed to pick up the latest secret version.
3. Trigger one manual Cloud Run job execution and confirm notifications/posts still work.

## Incident checks

1. Confirm Cloud Scheduler execution status and last response code.
2. Check Cloud Run job execution logs for failures and retry loops.
3. Inspect `guilds.json` / GCS `guilds` object for the expected guild mapping and enabled flag.
4. Inspect `state.json` / GCS `state` object for guild `job.status`, `retry_count`, and stale `posting_started_at`.
5. Inspect `problems.json` / GCS `problems` object for cache freshness and free-problem availability.
6. Check Discord notification channel for final failure messages.

## Recovery

1. If a guild is stuck in `posting` for more than 30 minutes, rerun the job; stale recovery should reset it.
2. If the problem cache is stale or missing, rerun the Cloud Run job after confirming LeetCode access.
3. If Discord permissions changed, restore forum/message permissions and rerun.
