# Plan: Add Discord Setup Command

## Branch

`feat/11-discord-setup-command`

## PR Title

`feat: add Discord setup command`

## Summary

Discord 上の `/setup` で guild ごとの投稿先設定を登録できるようにし、`config.json` の guild 固定設定を手編集しなくても運用開始できるようにする。

## Motivation

- 複数 guild 運用で `guild_id` や channel ID の手入力は負担が大きい。
- guild 固定設定を Bot 自身が登録できないと導入導線が重い。
- 初回導入の成否を Discord 上の操作で閉じたい。

## Scope

- guild 設定の永続化モデル追加
- Discord interaction の受け口追加
- `/setup` コマンド追加
- guild / channel / 権限の基本検証
- job runner が保存済み guild 設定を読む対応
- 最低限のテストと運用ドキュメント更新

## Proposed Changes

1. `config.json` から guild 配列を切り離し、guild 固定設定を別ストアで保存できるようにする。
2. Discord interaction request を受ける HTTP endpoint と署名検証を追加する。
3. `/setup` で `forum channel`、`notification channel`、`start problem number` を受け付け、実行 guild の設定として保存する。
4. `/setup` 実行者が必要権限を持つこと、指定 channel が同一 guild に属することを確認する。
5. job runner と notifier wiring を保存済み guild 設定ベースに切り替える。
6. セットアップ手順、必要 env、Discord 側の設定を README か runbook に追記する。

## Risks

- interaction 署名検証や command 応答の実装を誤ると Discord 連携自体が不安定になる。
- 保存モデル変更で既存の job 実行経路を壊すと回帰範囲が広い。
- `config.json` と新ストアの責務分離が曖昧だと以後の拡張が難しくなる。

## Validation

- `go test ./...`
- `/setup` request handler のユニットテストが通ること
- filesystem / GCS repository で guild 設定の保存と読み出しが検証されること
- ローカルで Discord interaction を模した request から設定登録まで再現できること

## Done Criteria

- 管理者が Discord 上の `/setup` だけで guild 設定を登録できる。
- 日次 job が登録済み guild 設定を使って従来どおり投稿できる。
- セットアップ手順が docs にまとまっている。
