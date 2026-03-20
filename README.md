# LeetDaily

LeetDaily は、LeetCode の無料問題を毎日 Discord フォーラムへ投稿する Go サービスです。

現時点ではアプリ骨格のみが実装されており、runtime 本体や永続化 backend は後続の plan で追加していきます。

## Requirements

- Go 1.26+

## Local Commands

```bash
make verify
make build
```

`make verify` は以下を順に実行します。

- `gofmt -l .`
- `go vet ./...`
- `go test ./...`

CI でも同じ検証を `make ci` 経由で実行します。
