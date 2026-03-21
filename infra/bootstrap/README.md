# Terraform Bootstrap

This root module creates the minimal shared resources required before `infra/terraform` can be managed from GitHub Actions:

- the GCS bucket used as the Terraform remote backend
- a GitHub Workload Identity Pool plus dedicated providers for `terraform-plan` and `terraform-apply`
- dedicated Terraform service accounts for `terraform-plan` and `terraform-apply`
- project-level IAM bindings required by the current Cloud Run / GCS / Secret Manager / Cloud Scheduler stack

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform fmt -recursive
terraform validate
terraform apply
```

After apply, configure these GitHub repository variables:

- `GCP_PROJECT_ID`
- `GCP_TERRAFORM_PLAN_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_TERRAFORM_APPLY_WORKLOAD_IDENTITY_PROVIDER`
- `TF_STATE_BUCKET`

You also need application-specific variables for the main stack:

- `LEETDAILY_CONTAINER_IMAGE`
- `LEETDAILY_DISCORD_TOKEN_SECRET_ID`

This module now fixes the auth naming contract in Terraform itself:

- plan service account account_id: `leetdaily-terraform-plan`
- apply service account account_id: `leetdaily-terraform-apply`
- plan provider ID: `leetdaily-terraform-plan`
- apply provider ID: `leetdaily-terraform-apply`

The outputs expose both the fixed IDs and the resulting full resource names so downstream automation can validate or derive workflow inputs from a stable convention. Terraform workflow service account emails are derived from the fixed naming convention plus `GCP_PROJECT_ID`. `terraform-plan` should not remain in the skipped path once the remaining repository variables are configured.
