# Terraform

This directory provisions the MVP production stack for LeetDaily:

- Cloud Run job
- runtime and scheduler service accounts
- GCS bucket for `config/guilds/state/problems`
- Secret Manager access for the Discord bot token
- Cloud Scheduler job that executes the Cloud Run job at `05:00` JST

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
terraform init \
  -backend-config="bucket=${TF_STATE_BUCKET}" \
  -backend-config="prefix=${TF_STATE_PREFIX:-prod/infra/terraform}"
terraform fmt -recursive
terraform validate
terraform plan
terraform apply
```

After apply:

1. Confirm the Cloud Scheduler target URI matches the Cloud Run job `:run` API.
2. Trigger one manual Cloud Run job execution for a smoke test.

## Bootstrap

Before using this root module in CI or production, apply `infra/bootstrap` once to create:

- the GCS backend bucket
- dedicated GitHub OIDC / Workload Identity Federation providers for `terraform-plan` and `terraform-apply`
- dedicated Terraform service accounts and IAM bindings for `terraform-plan` and `terraform-apply`

`terraform-plan` / `terraform-apply` workflows expect these GitHub repository variables:

- `GCP_PROJECT_ID`
- `GCP_TERRAFORM_PLAN_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_TERRAFORM_APPLY_WORKLOAD_IDENTITY_PROVIDER`
- `LEETDAILY_CONTAINER_IMAGE`
- `LEETDAILY_DISCORD_TOKEN_SECRET_ID`
- `TF_STATE_BUCKET`

If CI reports `terraform-plan-skipped`, the Terraform plan has not run yet. Treat that as bootstrap/configuration incomplete rather than a healthy steady state.
