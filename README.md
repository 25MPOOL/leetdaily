# LeetDaily

LeetDaily は、LeetCode の無料問題を毎日 Discord フォーラムへ投稿する Go サービスです。

このリポジトリは単一の Discord サーバーで運用する前提に寄せており、Discord のチャンネル ID などは手動で設定します。
`/setup` のような対話的な初期設定は使わず、常駐の Discord interaction endpoint も持ちません。

## Requirements

- [aqua](https://aquaproj.github.io/) for reproducible CLI installation
- [mise](https://mise.jdx.dev/) for Go runtime management

## Local Commands

```bash
aqua i
mise trust
mise install
make hooks-install
make verify
make workflow-lint
make terraform-check
make build
go run ./cmd/leetdaily
```

`make verify` は以下を順に実行します。

- `gofmt -l .`
- `go vet ./...`
- `go test ./...`

`make workflow-lint` は `actionlint` と `pinact run --check` を使って GitHub Actions workflow を検証します。

`make terraform-check` は `infra/bootstrap` と `infra/terraform` の両方で `terraform fmt -check -recursive` と `terraform validate` を実行します。

CI では通常変更に `make ci` を実行し、Go の検証に加えて workflow lint と Terraform validate も通します。`renovate.json` / `renovate.json5` だけを変更した pull request では、専用 workflow で Renovate 設定の軽量バリデーションだけを実行します。

## Dependency Updates

`renovate.json` を置いているので、Renovate App をこの repository にインストールすれば GitHub Actions / Go modules / Terraform / `aqua.yaml` の更新提案を自動化できます。

## Runtime Endpoints

HTTP mode is optional and mainly useful for local smoke tests.

- `GET /healthz`
- `POST /run`

## Local Development

1. Install repo-managed CLIs with `aqua i`.
2. Trust the checked-in `mise.toml` with `mise trust`.
3. Install the Go runtime from `mise.toml` with `mise install`.
4. Install the shared Git hooks with `make hooks-install`.
5. Activate `mise` in your shell, or prefix commands with `mise x --`.
6. Run `go test ./...`.
7. Build with `go build ./cmd/leetdaily`.
8. Prepare `config.json` for global settings and `guilds.json` for the single server's Discord settings.
9. Start the daily job with the required env vars:

```bash
eval "$(mise activate zsh)"
```

If you do not want to activate `mise` in your shell, you can run commands through `mise` directly:

```bash
mise x -- go test ./...
mise x -- go build ./cmd/leetdaily
mise x -- make hooks-install
```

Lefthook manages the repository Git hooks. The checked-in defaults are:

- `pre-commit`: `make fmtcheck`
- `pre-push`: `make verify`

Workflow lint and Terraform validation stay as explicit local commands or CI checks because they need extra tooling and are slower than the normal commit and push feedback loop.

With `mise` activated, the default one-shot job mode looks like this:

```bash
DISCORD_BOT_TOKEN=dummy \
LEETDAILY_RUNTIME=job \
go run ./cmd/leetdaily
```

If you want the optional HTTP runtime for local smoke tests, use:

```bash
DISCORD_BOT_TOKEN=dummy \
LEETDAILY_RUNTIME=http \
PORT=8080 \
go run ./cmd/leetdaily
```

`config.json` keeps global runtime behavior such as timezone, retry policy, and cache threshold. Discord channel settings live in `guilds.json` and are edited manually.

Example `guilds.json` for the single-server setup:

```json
{
  "guilds": [
    {
      "guild_id": "123456789012345678",
      "enabled": true,
      "forum_channel_id": "234567890123456789",
      "notification_channel_id": "345678901234567890",
      "start_problem_number": 1
    }
  ]
}
```

The manual configuration keeps the runtime simpler and avoids carrying a Discord interaction endpoint just for first-run setup.
Production uses a Cloud Run job triggered by Cloud Scheduler, so the bot does not need a continuously running HTTP service.

## Container

```bash
docker build -t leetdaily .
docker run --rm \
  -e DISCORD_BOT_TOKEN=dummy \
  -e LEETDAILY_RUNTIME=job \
  leetdaily
```

See [docs/runbook.md](docs/runbook.md) for deploy and operations guidance.

## Terraform CI/CD

- `infra/bootstrap` は Terraform backend 用 GCS bucket と GitHub OIDC / Workload Identity Federation の初期資材を作成します
- `infra/terraform` は GCS backend を使ってアプリ本体の infra を管理します
- PR では `terraform-plan` workflow が `infra/terraform` の plan を実行します
- apply は `terraform-apply` workflow を手動実行し、`production` environment の承認で gate する前提です
- `terraform-plan-skipped` が pass の場合は、CI 自体は成功でも Terraform plan はまだ実行されていません

必要な GitHub repository variables は以下です。

- `GCP_PROJECT_ID`
- `GCP_TERRAFORM_PLAN_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_TERRAFORM_PLAN_SERVICE_ACCOUNT`
- `GCP_TERRAFORM_APPLY_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_TERRAFORM_APPLY_SERVICE_ACCOUNT`
- `LEETDAILY_CONTAINER_IMAGE`
- `LEETDAILY_DISCORD_TOKEN_SECRET_ID`
- `TF_STATE_BUCKET`
- `TF_STATE_PREFIX`

`infra/bootstrap` 適用後は、上記 variables を設定して `terraform-plan` が skip ではなく実行されることを確認してください。
