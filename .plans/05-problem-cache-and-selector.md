# Plan: Add Problem Cache Refill and Free-Problem Selector

## Branch

`feat/05-problem-cache-and-selector`

## PR Title

`feat: add problem cache refill and free-problem selector`

## Summary

LeetCode GraphQL から問題メタデータを取得して `problems.json` を補充し、無料問題だけを problem number 順で選べるようにする。

## Motivation

- Bot のコア価値は「毎日 1 問、無料問題だけを順番に出す」ことにある。
- selector と cache refill を先に切り出すと Discord 連携と独立に検証できる。

## Scope

- LeetCode GraphQL client
- problem cache の補充判定
- 問題データの正規化
- 無料問題選択ロジック

## Proposed Changes

1. LeetCode GraphQL client を追加し、問題一覧をページング取得できるようにする。
2. レスポンスを `problem_number/title/slug/difficulty/is_paid_only` に正規化する。
3. `refill_threshold` に基づく cache 補充判定を実装する。
4. メモリ上では `map[int]Problem` を使い、`next_problem_number` から最初の無料問題を選ぶ selector を追加する。
5. cache 不足時の補充成功、補充失敗、既存 cache 継続利用の振る舞いを明示する。

## Risks

- GraphQL schema 依存を強くしすぎると API 変更に弱い。
- paid 問題の skip 条件を間違えると投稿対象がずれる。

## Validation

- `go test ./...`
- fake LeetCode server を使った client test
- paid 問題を挟んだ selector test

## Done Criteria

- cache 補充の成否に応じた振る舞いがテストで固定されている。
- 無料問題選択ロジックが Discord 実装なしで独立検証できる。
