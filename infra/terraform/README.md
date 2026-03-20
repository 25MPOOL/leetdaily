# Terraform

This directory provisions the MVP production stack for LeetDaily:

- Cloud Run service
- runtime and scheduler service accounts
- GCS bucket for `config/state/problems`
- Secret Manager access for the Discord bot token
- Cloud Scheduler job that calls `POST /run` at `05:00` JST

## Usage

```bash
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform fmt
terraform validate
terraform plan
terraform apply
```

After apply:

1. Check `GET /healthz` on the Cloud Run service URI.
2. Trigger `POST /run` manually with an authenticated request for a smoke test.
