# LeetDaily 設計書

## 概要

LeetDaily は、単一の Discord サーバー向けに、毎朝 5:00 JST に LeetCode の問題を 1 問ずつ自動投稿する Discord Bot である。

### 目的
- LeetCode を毎日解く習慣を作る
- 問題を自動でフォーラムに投稿する
- 難易度ごとのタグ付けを自動化する
- 低コストで安定運用する

---

## 前提条件

### 基本仕様
- サーバーは 1 つ固定で運用する
- フォーラムチャンネルと通知チャンネルは手動設定する
- 毎朝 5:00 JST ごろに 1 件投稿する
- 問題は LeetCode の problem number 順
- 投稿対象は無料問題のみ
- Premium 問題はスキップする
- 土日含め毎日投稿する
- 開始問題番号は手動設定する

### 投稿フォーマット
- タイトル: `<問題番号>. <問題名>`
- 本文: `[<問題名>](<URL>)`
- タグ: `Easy` / `Medium` / `Hard`

### 失敗時の挙動
- 5 分おきに最大 3 回リトライ
- それでも失敗した場合は通知チャンネルへ通知
- 重複投稿は防止する

---

## 技術選定

### 実装言語
- Go

### インフラ
- Terraform
- Cloud Scheduler
- Cloud Run

### 命名
- Bot 名: `LeetDaily`

---

## システム構成

```text
Cloud Scheduler
    ↓
Cloud Run Job
    ↓
LeetDaily (Go)
    ├─ config.json
    ├─ guilds.json
    ├─ state.json
    ├─ problems.json
    ├─ LeetCode GraphQL
    └─ Discord API
```

### 実行モデル
毎朝 5:00 JST に Cloud Scheduler が Cloud Run Job を 1 回実行し、その中で単一サーバー向けの投稿処理を完了させる。
`/setup` のような Discord interaction は使わず、guild と channel の設定は `guilds.json` もしくは GCS の設定オブジェクトを直接編集して管理する。

---

## LeetCode 問題取得戦略

## 方針
LeetCode の問題データは、毎朝都度フル取得するのではなく、`problems.json` にキャッシュして使う。

### 理由
- 毎朝の投稿処理を軽くしたい
- 外部依存を減らしたい
- 5:00 の投稿成功率を上げたい
- Cloud Run のコストを抑えたい

### 取得元
LeetCode の GraphQL エンドポイントを利用して問題メタデータを取得する想定。

### 保存する項目
- `problem_number`
- `title`
- `slug`
- `difficulty`
- `is_paid_only`

### URL 生成
URL は保存せず、投稿時に以下の形式で組み立てる。

```text
https://leetcode.com/problems/{slug}
```

### キャッシュ補充方針
- 初回セットアップ時に可能な限り多く取得する
- 毎朝の投稿では `problems.json` を参照する
- キャッシュ残量が少なくなったら補充更新を試みる
- 補充失敗時も既存キャッシュが残っていれば継続利用する

### 補充しきい値
- `refill_threshold = 30`

---

## データ設計

## 1. config.json

人が編集する設定ファイル。

### 役割
- timezone
- retry 設定
- problem cache 補充しきい値

### 例

```json
{
  "timezone": "Asia/Tokyo",
  "retry": {
    "interval_minutes": 5,
    "max_attempts": 3
  },
  "problem_cache": {
    "refill_threshold": 30
  }
}
```

---

## 2. guilds.json

人が編集する Discord 固定設定ファイル。

### 役割
- 単一サーバーの guild 設定
- forum / notification channel 設定
- 開始 problem number の管理

### 例

```json
{
  "guilds": [
    {
      "guild_id": "123456789012345678",
      "enabled": true,
      "forum_channel_id": "234567890123456789",
      "notification_channel_id": "345678901234567890",
      "start_problem_number": 1
    }
  ]
}
```

---

## 3. state.json

Bot が更新する状態ファイル。

### 役割
- guild ごとの進捗
- 今日の投稿状態
- 直近エラー
- 重複防止用ジョブ状態

### 例

```json
{
  "guild_states": {
    "123456789012345678": {
      "next_problem_number": 1,
      "last_posted_problem_number": null,
      "last_posted_at": null,
      "last_posted_thread_id": null,
      "job": {
        "target_date": null,
        "status": "idle",
        "problem_number": null,
        "retry_count": 0,
        "last_error": null,
        "posting_started_at": null
      }
    }
  }
}
```

---

## 4. problems.json

LeetCode 問題キャッシュ。

### 役割
- 毎朝の投稿で外部依存を減らす
- 難易度タグ付けに使う
- 無料問題判定に使う
- URL 組み立てに使う

### 例

```json
{
  "updated_at": "2026-03-20T05:00:00+09:00",
  "problems": [
    {
      "problem_number": 1,
      "title": "Two Sum",
      "slug": "two-sum",
      "difficulty": "Easy",
      "is_paid_only": false
    },
    {
      "problem_number": 2,
      "title": "Add Two Numbers",
      "slug": "add-two-numbers",
      "difficulty": "Medium",
      "is_paid_only": false
    }
  ]
}
```

---

## 状態管理設計

## job.status
- `idle`
- `posting`
- `posted`
- `failed`

### 意味
- `idle`: 今日の処理前
- `posting`: 投稿処理中
- `posted`: 今日の投稿成功済み
- `failed`: 今日の最終失敗

### job に持つ項目
- `target_date`
- `status`
- `problem_number`
- `retry_count`
- `last_error`
- `posting_started_at`

### 例

```json
{
  "target_date": "2026-03-20",
  "status": "posting",
  "problem_number": 137,
  "retry_count": 1,
  "last_error": null,
  "posting_started_at": "2026-03-20T05:00:03+09:00"
}
```

---

## 投稿フロー

## 1. 起動時
1. `config.json` を読む
2. `state.json` を読む
3. state に guild がなければ初期化する
4. `problems.json` を読む
5. 必要なら問題キャッシュ補充を試みる

## 2. guild ごとに処理
- `enabled: true` の guild を順番に処理する

## 3. 今日すでに投稿済みか確認
以下を満たす場合はスキップする。

- `job.target_date == 今日`
- `job.status == posted`

## 4. 投稿対象問題を決定
`next_problem_number` から順に見て、最初の無料問題を選ぶ。

### ルール
- `is_paid_only == true` の問題はスキップ
- `is_paid_only == false` の問題を採用
- データが不足していれば補充を試みる

## 5. 投稿前に state を `posting` に更新
Discord 投稿より先に state を `posting` にして保存する。

## 6. Discord タグ確認
フォーラムのタグ一覧から以下を探す。

- Easy
- Medium
- Hard

存在しない場合は作成する。

## 7. 投稿内容生成
### タイトル
```text
137. Single Number II
```

### 本文
```md
[Single Number II](https://leetcode.com/problems/single-number-ii)
```

## 8. フォーラム投稿
- タイトル
- 本文
- 難易度タグ

を使って新規スレッドを作成する。

## 9. 成功時の state 更新
- `job.status = posted`
- `last_posted_problem_number = current`
- `last_posted_at = now`
- `last_posted_thread_id = created_thread_id`
- `next_problem_number = current + 1`

---

## 無料問題選択ロジック

### 方針
`next_problem_number` から順に探索し、最初に見つかった無料問題を投稿対象にする。

### 推奨実装
`problems.json` 読み込み後、メモリ上では以下の形で扱う。

```text
Map<number, Problem>
```

### 理由
- 配列添字依存を避けられる
- 問題番号ベースで自然に探索できる
- 将来欠番があっても扱いやすい

---

## リトライ戦略

### 方針
- 初回投稿に失敗したら 5 分待機
- 最大 3 回まで再試行
- 3 回とも失敗したら通知

### 実行モデル
1 回の Cloud Run 実行の中で完結させる。

### 理由
- 実装がシンプル
- 状態遷移が追いやすい
- Terraform / Scheduler 構成も単純になる

---

## 通知設計

### 通知先
- `notification_channel_id`

### 通知タイミング
- 全リトライ失敗後
- 設定不備で処理不能な時
- `problems.json` がなく、問題取得にも失敗した時

### 通知内容例

```text
LeetCode daily post failed.
Guild: 25MPOOL
Problem: 137
Attempt: 3/3
Error: Missing permissions to create forum post
```

---

## Discord タグ設計

### 使用タグ
- Easy
- Medium
- Hard

### 方針
毎回実行時に確認する。

### 理由
- 手動削除時に自動復旧できる
- tag ID の変化に対応しやすい
- 初回セットアップを簡単にできる

---

## 初期化仕様

### 初回起動時
state に対象 guild の状態が存在しない場合は、以下で初期化する。

- `next_problem_number = start_problem_number`
- `last_posted_problem_number = null`
- `last_posted_at = null`
- `last_posted_thread_id = null`
- `job.status = idle`

### 注意
`start_problem_number` は初回のみ利用し、それ以降は `state.json` の内容を優先する。

---

## 多重起動対策

### 想定事故
- Cloud Scheduler の重複実行
- 手動再実行
- 一時的な障害による再起動

### 対策
以下を満たす場合、同日の重複実行をスキップする。

- `job.target_date == 今日`
- `job.status in [posting, posted]`

### stale posting 対策
`posting` 状態が長時間残るケースに備えて、`posting_started_at` を持つ。

### 推奨ルール
- `posting` が 30 分以上古ければ異常状態とみなす
- その場合のみ再実行対象にする

---

## JSON 保存の安全性

### 方針
`state.json` と `problems.json` は直接上書きせず、一時ファイル経由で更新する。

### 流れ
1. `*.tmp.json` に書き込む
2. 書き込み成功を確認する
3. rename で本番ファイルへ置き換える

### 理由
- 途中クラッシュ時の破損を防ぎやすい
- 壊れた JSON の生成リスクを減らせる

---

## モジュール構成案

### 推奨モジュール
- `config`
- `state`
- `problemcache`
- `leetcode`
- `selector`
- `discordforum`
- `notifier`
- `job`

### 各責務

#### config
- config 読み込み
- バリデーション

#### state
- state 読み込み
- state 初期化
- state 保存

#### problemcache
- problems 読み込み
- 補充判定
- 更新保存

#### leetcode
- LeetCode GraphQL 取得

#### selector
- 次の無料問題を選ぶ

#### discordforum
- タグ確認
- フォーラム投稿

#### notifier
- 通知チャンネルへのエラー通知

#### job
- 全体のオーケストレーション

---

## MVP 範囲

### MVP に含める
- 毎朝 5:00 JST 投稿
- guild ごとの独立進捗
- 無料問題のみ投稿
- 難易度タグ自動確認・自動作成
- 5 分おき最大 3 回リトライ
- 失敗通知
- JSON 永続化
- 重複防止用 job state

### MVP では含めない
- Slash Command
- 管理画面
- DB
- 手動再投稿コマンド
- 投稿済みスレッドへの追記
- Web UI

---

## 今後の拡張候補
- `/status`
- `/retry`
- `/set-number`
- `/post-now`
- Firestore / GCS への移行
- 管理用 HTTP エンドポイント
- 手動キャッシュ更新エンドポイント

---

## 命名案

### リポジトリ名
- `leetdaily-bot`

### Cloud Run Service
- `leetdaily-bot`

### Terraform Module / Resource Prefix
- `leetdaily`

---

## 最終まとめ

LeetDaily は、LeetCode の無料問題を problem number 順に毎日 Discord フォーラムへ投稿し、学習習慣を支える Bot である。

### 設計上の重要ポイント
- 毎朝の本番処理はできるだけ軽くする
- 問題データは `problems.json` にキャッシュする
- guild ごとに進捗を独立管理する
- state に `posting` / `posted` / `failed` を持たせて重複投稿を防ぐ
- 低コスト運用のため Cloud Scheduler + Cloud Run を採用する
- Go で堅く実装する

この設計をベースにすれば、まずは十分安定した MVP を作れる。
