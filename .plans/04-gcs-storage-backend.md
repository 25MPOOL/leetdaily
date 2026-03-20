# Plan: Add GCS-Backed Persistence

## Branch

`feat/04-gcs-storage-backend`

## PR Title

`feat: add GCS-backed persistence`

## Summary

本番用の永続化 backend として GCS を追加し、Cloud Run から `config/state/problems` を安全に読み書きできるようにする。

## Motivation

- Cloud Run のローカル filesystem は永続化前提で使えない。
- JSON スキーマは維持しつつ、本番だけ保存先を差し替えたい。

## Scope

- GCS backend
- object key 設定
- generation precondition による競合検知
- 環境変数との接続

## Proposed Changes

1. GCS client を使った storage backend を追加する。
2. `GCS_BUCKET` `CONFIG_OBJECT` `STATE_OBJECT` `PROBLEMS_OBJECT` を runtime config に追加する。
3. `config` は read-only、`state` と `problems` は更新可能 object として扱う。
4. `state` と `problems` の更新時に generation precondition を使い、競合更新を検知する。
5. local backend と共通の interface test を持たせ、backend 差し替えを保証する。

## Risks

- generation conflict の扱いを曖昧にすると重複投稿防止が壊れる。
- GCS client の初期化を domain 層へ漏らすと依存が汚れる。

## Validation

- `go test ./...`
- backend interface test が filesystem backend と GCS backend の両方で通ること
- generation conflict を再現するユニットテスト

## Done Criteria

- production runtime が GCS を唯一の永続化先として使える。
- 後続 PR が storage backend の違いを意識せず job を実装できる。
