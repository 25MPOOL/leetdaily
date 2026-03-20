# Plan: Add Domain Models and Local JSON Storage

## Branch

`feat/03-domain-models-and-storage`

## PR Title

`feat: add domain models and local JSON storage`

## Summary

`config.json` `state.json` `problems.json` の型と JSON I/O を確定し、開発とテストで使う filesystem backend を作る。

## Motivation

- 後続 PR のほぼ全てが config/state/problem cache の型に依存する。
- まずローカル backend を固めると、本番用 GCS backend も同じ interface で差し替えられる。

## Scope

- domain types と validation
- JSON load/save
- filesystem backend
- atomic write

## Proposed Changes

1. `Config` `State` `GuildState` `JobState` `ProblemCache` `Problem` の型を定義する。
2. 設計書に沿った validation を実装し、不正設定を起動時に落とせるようにする。
3. `config.json` `state.json` `problems.json` を読む repository interface を定義する。
4. filesystem backend を追加し、ローカル path から JSON を load/save できるようにする。
5. `state.json` と `problems.json` の保存は temp file + rename の atomic write に統一する。
6. state に guild が無い場合の初期化 helper を追加する。

## Risks

- field 名や null 許容を曖昧にすると後続 PR の互換性が崩れる。
- filesystem backend の責務が広がりすぎると GCS backend に差し替えにくい。

## Validation

- `go test ./...`
- filesystem backend の load/save テスト
- atomic write 後に JSON が壊れないことのテスト

## Done Criteria

- 3 種類の JSON を型安全に扱える。
- 後続 PR が local backend で state 遷移や cache 更新を検証できる。
