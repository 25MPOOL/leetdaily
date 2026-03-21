# LeetDaily

LeetDaily は、LeetCode の無料問題を毎日 Discord フォーラムへ投稿する Go サービスです。

現時点では、job orchestration と HTTP runtime の土台まで実装済みで、Cloud Run 配備前提の構成を順次追加しています。

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

- `GET /healthz`
- `POST /run`
- `POST /discord/interactions`

## Local Development

1. Install repo-managed CLIs with `aqua i`.
2. Trust the checked-in `mise.toml` with `mise trust`.
3. Install the Go runtime from `mise.toml` with `mise install`.
<<<<<<< HEAD
4. Install the shared Git hooks with `make hooks-install`.
5. Activate `mise` in your shell, or prefix commands with `mise x --`.
6. Run `go test ./...`.
7. Build with `go build ./cmd/leetdaily`.
8. Prepare `config.json` for global settings and `guilds.json` for per-server settings.
9. Start the service with the required env vars:

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

With `mise` activated, you can start the service like this:

```bash
DISCORD_BOT_TOKEN=dummy \
LEETDAILY_RUNTIME=http \
PORT=8080 \
go run ./cmd/leetdaily
```

`config.json` now keeps only global runtime behavior such as timezone, retry policy, and cache threshold. Guild-specific Discord settings are stored in `guilds.json` and can be managed through the Discord `/setup` command.

Example `guilds.json`:

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

To enable `/setup`, configure your Discord application to send interactions to `/discord/interactions` and provide `DISCORD_APPLICATION_PUBLIC_KEY`.

For local testing, set `DISCORD_APPLICATION_PUBLIC_KEY` to the real Discord application public key, or omit it when you do not need `/setup`.

Register the slash command once against your application, for example:

```bash
curl -X POST "https://discord.com/api/v10/applications/$DISCORD_APPLICATION_ID/commands" \
  -H "Authorization: Bot $DISCORD_BOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "setup",
    "description": "Configure LeetDaily for this server",
    "options": [
      {
        "name": "forum",
        "description": "Forum channel used for daily posts",
        "type": 7,
        "required": true
      },
      {
        "name": "notify",
        "description": "Text channel used for failure notifications",
        "type": 7,
        "required": true
      },
      {
        "name": "start",
        "description": "First LeetCode problem number to post",
        "type": 4,
        "required": true,
        "min_value": 1
      }
    ]
  }'
```

After the command is registered, a server admin can run `/setup forum:<forum-channel> notify:<text-channel> start:<number>`.

## Container

```bash
docker build -t leetdaily .
docker run --rm -p 8080:8080 \
  -e PORT=8080 \
  -e DISCORD_BOT_TOKEN=dummy \
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
