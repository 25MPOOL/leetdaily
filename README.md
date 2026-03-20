# LeetDaily

LeetDaily は、LeetCode の無料問題を毎日 Discord フォーラムへ投稿する Go サービスです。

現時点では、job orchestration と HTTP runtime の土台まで実装済みで、Cloud Run 配備前提の構成を順次追加しています。

## Requirements

- Go 1.26+

## Local Commands

```bash
make verify
make build
go run ./cmd/leetdaily
```

`make verify` は以下を順に実行します。

- `gofmt -l .`
- `go vet ./...`
- `go test ./...`

CI でも同じ検証を `make ci` 経由で実行します。

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
