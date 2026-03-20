# Plan: Add CI Quality Gates for Go Project

## Branch

`chore/02-ci-quality-gates`

## PR Title

`chore: add CI quality gates for Go project`

## Summary

ローカルだけに依存しない品質ゲートを整備し、以降の PR で lint/test の破壊を早期検知できる状態にする。

## Motivation

- 小さめ PR を前提にするなら、各 PR で自動検証が回ることが前提になる。
- 空 repo の段階で CI を入れておくと後続 PR の運用が安定する。

## Scope

- GitHub Actions の CI workflow
- `go test` `go vet` `gofmt -l` などの品質ゲート
- ローカル実行コマンドの整理

## Proposed Changes

1. `.github/workflows/ci.yml` を追加する。
2. CI で Go を setup し、依存解決、`go test ./...`、`go vet ./...` を実行する。
3. `gofmt -l` による formatting check を追加する。
4. ローカルでも同じ検証がしやすいように `Makefile` か `mage` の最小コマンドを追加する。
5. README か docs に CI の実行内容を短く追記する。

## Risks

- 初期段階でチェックを盛り込みすぎると後続 PR の変更コストが増える。
- formatter/linter の選定を早く決めすぎると見直しにくい。

## Validation

- `go test ./...`
- `go vet ./...`
- `gofmt -l .`
- GitHub Actions の branch 実行確認

## Done Criteria

- PR ごとに最低限の test/vet/format check が自動で走る。
- ローカルと CI の検証内容が大きく乖離していない。
