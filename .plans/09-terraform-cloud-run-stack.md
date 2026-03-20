# Plan: Provision Cloud Run Scheduler and Storage Stack

## Branch

`infra/09-terraform-cloud-run-stack`

## PR Title

`infra: provision Cloud Run scheduler and storage stack`

## Summary

Cloud Run、GCS、Secret Manager access、Cloud Scheduler、IAM を Terraform でまとめて管理し、MVP の本番基盤を作る。

## Motivation

- アプリ実装と同時に再現可能な infra 定義が無いと運用が手作業になる。
- Scheduler、service account、bucket 権限の噛み合わせはコードで固定したい。

## Scope

- Terraform module と variables
- Cloud Run service
- GCS bucket
- Secret Manager access
- Cloud Scheduler と IAM

## Proposed Changes

1. `infra/terraform` に provider、variables、main outputs を追加する。
2. Cloud Run service と service account を作成する。
3. JSON 保存用の GCS bucket と必要 IAM を作成する。
4. Discord token 用 Secret Manager secret の参照権限を付与する。
5. JST 05:00 実行の Cloud Scheduler job を OIDC 認証付き `POST /run` で作成する。
6. deploy に必要な input variables と example tfvars を追加する。

## Risks

- IAM の最小権限を外すと本番運用時の権限漏れが起きやすい。
- Scheduler の retry 設定がアプリ側 retry と競合すると二重実行につながる。

## Validation

- `terraform fmt -check`
- `terraform validate`
- 手動 apply 後に `/healthz` 確認
- 手動 `POST /run` の smoke test

## Done Criteria

- MVP を手作業なしで再現できる Terraform 定義が揃う。
- アプリ側の runtime config が infra から供給できる。
