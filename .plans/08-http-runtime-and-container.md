# Plan: Add HTTP Runtime and Cloud Run Containerization

## Branch

`feat/08-http-runtime-and-container`

## PR Title

`feat: add HTTP runtime and Cloud Run containerization`

## Summary

Cloud Run から起動できる HTTP runtime、Dockerfile、環境変数 wiring を追加し、アプリをデプロイ可能な形にする。

## Motivation

- orchestration ができても、Cloud Scheduler から叩ける runtime が無ければ本番投入できない。
- `/run` と `/healthz` を分けておくと運用確認がしやすい。

## Scope

- HTTP server
- `/run` と `/healthz`
- env wiring
- Dockerfile

## Proposed Changes

1. `POST /run` で job 実行、`GET /healthz` でヘルス応答する HTTP server を追加する。
2. runtime config と各 backend/client の依存注入を `main` に集約する。
3. Cloud Run 向けの `PORT` 対応と graceful shutdown を追加する。
4. Dockerfile と必要な `.dockerignore` を追加する。
5. ローカル起動手順を README か docs に追記する。

## Risks

- `/run` のエラーハンドリングが曖昧だと Scheduler 側の再試行方針と衝突する。
- container image を複雑にしすぎると build 時間が増える。

## Validation

- `go test ./...`
- `go build ./cmd/leetdaily`
- ローカルで `/healthz` が 200 を返す確認
- ローカルで `POST /run` が job runner を呼ぶ確認
- Docker build の確認

## Done Criteria

- ローカルと Cloud Run の両方で同じ binary を起動できる。
- Scheduler 連携に必要な HTTP contract が固定されている。
