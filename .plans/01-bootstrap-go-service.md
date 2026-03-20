# Plan: Bootstrap Go Service Skeleton

## Branch

`feat/01-bootstrap-go-service`

## PR Title

`feat: bootstrap Go service skeleton`

## Summary

空のリポジトリを Go サービスとして初期化し、以降の PR が乗る最小のアプリ骨格を作る。

## Motivation

- 現在は設計書しかなく、ビルド可能なコードが存在しない。
- 後続 PR で package 構成や logger の置き場を毎回決め直したくない。

## Scope

- `go.mod` と基本ディレクトリ構成
- `cmd/leetdaily` の最小エントリポイント
- 共通 logger と runtime 設定の読み込み骨格
- `go test` を回せる最小テスト基盤

## Proposed Changes

1. `github.com/nkoji21/leetdaily` で Go module を初期化する。
2. `cmd/leetdaily` と `internal/` 配下の基本 package を追加する。
3. 構造化ログを出せる共通 logger package を追加する。
4. 環境変数から runtime 設定を読む最小 config loader を追加する。
5. `main` は HTTP server または job runner の骨格だけを持ち、実ロジックは後続 PR に委譲する。
6. package の雛形に対する最小ユニットテストを追加する。

## Risks

- ここで domain model や storage まで入れると後続 PR の責務が崩れる。
- 初期 package 分割が細かすぎると空 package が増える。

## Validation

- `go test ./...`
- `go build ./cmd/leetdaily`

## Done Criteria

- リポジトリが Go project として build/test 可能である。
- 後続 PR が package 配置の再整理なしで実装を積める。
