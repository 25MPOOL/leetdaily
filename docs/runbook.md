# Runbook

## Local setup

1. Install tools with `aqua i`.
2. Prepare `config.json`, `state.json`, and `problems.json` locally or use GCS-backed objects.
3. Set required env vars:
   - `DISCORD_BOT_TOKEN`
   - `LEETDAILY_RUNTIME=http` or `job`
   - `PORT` when using HTTP mode
   - `GCS_BUCKET` plus `CONFIG_OBJECT`, `STATE_OBJECT`, `PROBLEMS_OBJECT` for production-style storage
4. Run `go test ./...` and `go build ./cmd/leetdaily`.
5. Start locally with `go run ./cmd/leetdaily`.

## Deploy

1. Build and push the container image.
2. Update `infra/terraform/terraform.tfvars`.
3. Run:

```bash
cd infra/terraform
terraform init
terraform plan
terraform apply
```

4. Verify `GET /healthz` returns `200 OK`.
5. Send an authenticated `POST /run` smoke test.

## Secret rotation

1. Update the Discord bot token in Secret Manager.
2. Redeploy or restart the Cloud Run service if needed to pick up the latest secret version.
3. Trigger `POST /run` manually and confirm notifications/posts still work.

## Incident checks

1. Confirm Cloud Scheduler execution status and last response code.
2. Check Cloud Run logs for `/run` failures and retry loops.
3. Inspect `state.json` / GCS `state` object for guild `job.status`, `retry_count`, and stale `posting_started_at`.
4. Inspect `problems.json` / GCS `problems` object for cache freshness and free-problem availability.
5. Check Discord notification channel for final failure messages.

## Recovery

1. If a guild is stuck in `posting` for more than 30 minutes, rerun the job; stale recovery should reset it.
2. If the problem cache is stale or missing, rerun `/run` after confirming LeetCode access.
3. If Discord permissions changed, restore forum/message permissions and rerun.
