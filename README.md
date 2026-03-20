# LeetDaily

LeetDaily は、LeetCode の無料問題を毎日 Discord フォーラムへ投稿する Go サービスです。

現時点では、job orchestration と HTTP runtime の土台まで実装済みで、Cloud Run 配備前提の構成を順次追加しています。

## Requirements

- Go 1.26+
- [aqua](https://aquaproj.github.io/) for Terraform and workflow lint tooling

## Local Commands

```bash
aqua i
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

CI では `make ci` を実行し、Go の検証に加えて workflow lint と Terraform validate も通します。

## Runtime Endpoints

- `GET /healthz`
- `POST /run`

## Local Development

1. Install tools with `aqua i`.
2. Run `go test ./...`.
3. Build with `go build ./cmd/leetdaily`.
4. Start the service with the required env vars:

```bash
DISCORD_BOT_TOKEN=dummy \
LEETDAILY_RUNTIME=http \
PORT=8080 \
go run ./cmd/leetdaily
```

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

必要な GitHub repository variables は以下です。

- `GCP_PROJECT_ID`
- `GCP_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_TERRAFORM_SERVICE_ACCOUNT`
- `LEETDAILY_CONTAINER_IMAGE`
- `LEETDAILY_DISCORD_TOKEN_SECRET_ID`
- `TF_STATE_BUCKET`
- `TF_STATE_PREFIX`
