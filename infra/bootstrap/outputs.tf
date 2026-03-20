output "terraform_plan_service_account_email" {
  description = "Service account email used by the terraform-plan workflow"
  value       = google_service_account.terraform_ci["plan"].email
}

output "terraform_apply_service_account_email" {
  description = "Service account email used by the terraform-apply workflow"
  value       = google_service_account.terraform_ci["apply"].email
}

output "terraform_state_bucket_name" {
  description = "GCS bucket name used as the Terraform backend"
  value       = google_storage_bucket.terraform_state.name
}

output "terraform_plan_workload_identity_provider_name" {
  description = "Full resource name for the terraform-plan Workload Identity Provider"
  value       = google_iam_workload_identity_pool_provider.github["plan"].name
}

output "terraform_apply_workload_identity_provider_name" {
  description = "Full resource name for the terraform-apply Workload Identity Provider"
  value       = google_iam_workload_identity_pool_provider.github["apply"].name
}
