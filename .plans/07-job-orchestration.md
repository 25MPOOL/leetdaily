# Plan: Orchestrate Per-Guild Daily Posting Jobs

## Branch

`feat/07-job-orchestration`

## PR Title

`feat: orchestrate per-guild daily posting jobs`

## Summary

config、state、problem cache、Discord client を束ね、guild ごとの日次投稿フローと retry/idempotency を完成させる。

## Motivation

- 各 subsystem が揃っても、job の状態遷移が曖昧だと二重投稿や投稿漏れが起きる。
- 失敗時の retry、stale recovery、notification をここで一貫して扱う必要がある。

## Scope

- 起動時の読み込みフロー
- guild 初期化
- 同日重複防止
- retry と state 遷移

## Proposed Changes

1. `job.Run(ctx, targetDate)` を実装し、config/state/cache を読み込んで guild を順次処理する。
2. state に guild が無い場合の初期化と、`enabled` guild だけを対象にする処理を追加する。
3. `job.target_date == 今日` かつ `posting/posted` の skip 条件を実装する。
4. `posting_started_at` が 30 分以上古い stale state の回復ロジックを追加する。
5. 投稿前に `posting` へ遷移して保存し、成功時に `posted` と next problem を更新する。
6. 初回を含む最大 3 回試行、5 分待機、最終失敗時通知を実装する。
7. guild 単位の失敗が他 guild の処理を止めないようにする。

## Risks

- state 保存の順序を誤ると重複投稿防止が成立しない。
- retry と notifier の責務が混ざると障害時挙動が追いにくい。

## Validation

- `go test ./...`
- 同日二重起動 skip のテスト
- stale `posting` 回復のテスト
- retry 成功、retry 枯渇、notification 送信のテスト

## Done Criteria

- guild ごとの日次投稿フローが state 遷移込みで固定されている。
- 再実行時に重複投稿せず、失敗時は通知まで到達する。
