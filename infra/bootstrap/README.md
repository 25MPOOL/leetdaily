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
- `GCP_TERRAFORM_PLAN_SERVICE_ACCOUNT`
- `GCP_TERRAFORM_APPLY_WORKLOAD_IDENTITY_PROVIDER`
- `GCP_TERRAFORM_APPLY_SERVICE_ACCOUNT`
- `TF_STATE_BUCKET`
- `TF_STATE_PREFIX`

You also need application-specific variables for the main stack:

- `LEETDAILY_CONTAINER_IMAGE`
- `LEETDAILY_DISCORD_TOKEN_SECRET_ID`

Use the outputs from this module to populate the provider and service-account variables. `terraform-plan` should not remain in the skipped path once these are configured.
