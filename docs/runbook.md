# Runbook

## Local setup

1. Install tools with `aqua i`.
2. Prepare `config.json`, `guilds.json`, `state.json`, and `problems.json` locally or use GCS-backed objects.
3. Set required env vars:
   - `DISCORD_BOT_TOKEN`
   - `DISCORD_APPLICATION_PUBLIC_KEY` when enabling Discord `/setup`
   - `LEETDAILY_RUNTIME=http` or `job`
   - `PORT` when using HTTP mode
   - `GCS_BUCKET` plus `CONFIG_OBJECT`, `GUILDS_OBJECT`, `STATE_OBJECT`, `PROBLEMS_OBJECT` for production-style storage
4. Run `go test ./...` and `go build ./cmd/leetdaily`.
5. Start locally with `go run ./cmd/leetdaily`.

### Discord setup command

1. Register the Discord slash command `/setup`.
2. Point the Discord interaction endpoint at `POST /discord/interactions`.
3. Run `/setup forum:<forum-channel> notify:<text-channel> start:<number>` as a server admin.
4. Confirm `guilds.json` or the configured GCS guild object now contains the guild entry.

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
2. Update the Discord application public key as well if the application was recreated.
3. Redeploy or restart the Cloud Run service if needed to pick up the latest secret version.
4. Trigger `POST /run` manually and confirm notifications/posts still work.

## Incident checks

1. Confirm Cloud Scheduler execution status and last response code.
2. Check Cloud Run logs for `/run` failures and retry loops.
3. Inspect `guilds.json` / GCS `guilds` object for the expected guild mapping and enabled flag.
4. Inspect `state.json` / GCS `state` object for guild `job.status`, `retry_count`, and stale `posting_started_at`.
5. Inspect `problems.json` / GCS `problems` object for cache freshness and free-problem availability.
6. Check Discord notification channel for final failure messages.

## Recovery

1. If a guild is stuck in `posting` for more than 30 minutes, rerun the job; stale recovery should reset it.
2. If the problem cache is stale or missing, rerun `/run` after confirming LeetCode access.
3. If Discord permissions changed, restore forum/message permissions and rerun.
