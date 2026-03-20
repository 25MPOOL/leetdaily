# Plan: Add Integration Coverage and Operational Docs

## Branch

`chore/10-integration-tests-and-runbook`

## PR Title

`chore: add integration coverage and operational docs`

## Summary

MVP 全体の挙動を fake 依存で結合検証し、運用に必要な runbook とデプロイ手順を仕上げる。

## Motivation

- 小さめ PR で進めると subsystem 単位のテストは揃っても全体結合の保証が弱くなりやすい。
- 運用者向け手順が無いと障害時の復旧速度が落ちる。

## Scope

- fake LeetCode / fake Discord を使う結合テスト
- ローカル実行手順
- デプロイと障害対応の runbook

## Proposed Changes

1. fake LeetCode server と fake Discord server を使う end-to-end 寄りの結合テストを追加する。
2. 初回成功、retry 後成功、最終失敗通知、同日 skip を主要シナリオとして固定する。
3. ローカル開発手順、必要 env、sample JSON、実行コマンドを README か `docs/` に整理する。
4. 本番デプロイ、secret 更新、state/problem cache の確認方法、障害時確認項目を runbook にまとめる。

## Risks

- 結合テストが遅く不安定だと PR 体験が悪化する。
- runbook が実装とずれると保守価値が落ちる。

## Validation

- `go test ./...`
- 結合テストの主要シナリオがローカルで再現可能であること
- runbook に沿って手動 smoke test ができること

## Done Criteria

- MVP の主要フローが結合テストで守られている。
- 新しい運用者が docs だけでデプロイと基本調査を実施できる。
