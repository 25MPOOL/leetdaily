output "terraform_service_account_email" {
  description = "Service account email used by GitHub Actions Terraform workflows"
  value       = google_service_account.terraform_ci.email
}

output "terraform_state_bucket_name" {
  description = "GCS bucket name used as the Terraform backend"
  value       = google_storage_bucket.terraform_state.name
}

output "workload_identity_provider_name" {
  description = "Full resource name for the GitHub Workload Identity Provider"
  value       = google_iam_workload_identity_pool_provider.github.name
}
