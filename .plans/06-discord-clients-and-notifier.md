# Plan: Add Discord Forum Posting and Failure Notification

## Branch

`feat/06-discord-clients-and-notifier`

## PR Title

`feat: add Discord forum posting and failure notification`

## Summary

Discord API 層を追加し、forum tag の確保、thread 投稿、通知チャンネルへのエラー送信を実装する。

## Motivation

- 投稿と通知は外部副作用であり、job orchestration から独立してテスト可能な層にしておきたい。
- forum tag の自動作成をここで閉じ込めると orchestration が単純になる。

## Scope

- Discord API client
- difficulty tag の確認と作成
- forum thread 作成
- notification channel への送信

## Proposed Changes

1. Bot token を使う Discord client を追加する。
2. forum channel から既存 tag を取得し、`Easy` `Medium` `Hard` が無ければ作成する helper を実装する。
3. title/body/tag を受けて新規 thread を作成する API を追加する。
4. 失敗通知メッセージを組み立てて notification channel に送る notifier を追加する。
5. Discord API の request/response をテスト可能にするため transport 抽象を入れる。

## Risks

- rate limit や permissions error の扱いが薄いと本番障害時の調査が難しい。
- API schema を広くラップしすぎると実装量だけ増える。

## Validation

- `go test ./...`
- fake Discord server を使った tag ensure test
- thread 作成と notification 送信のテスト

## Done Criteria

- job 層が Discord の HTTP 詳細を知らずに投稿と通知を呼び出せる。
- difficulty tag の自動復旧が実装されている。
