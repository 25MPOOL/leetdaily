output "cloud_run_service_name" {
  value       = google_cloud_run_v2_service.leetdaily.name
  description = "Provisioned Cloud Run service name"
}

output "cloud_run_service_uri" {
  value       = google_cloud_run_v2_service.leetdaily.uri
  description = "Cloud Run service base URI"
}

output "runtime_service_account_email" {
  value       = google_service_account.leetdaily_runtime.email
  description = "Service account used by Cloud Run"
}

output "scheduler_service_account_email" {
  value       = google_service_account.scheduler_invoker.email
  description = "Service account used by Cloud Scheduler OIDC"
}

output "bucket_name" {
  value       = google_storage_bucket.leetdaily_data.name
  description = "GCS bucket name used for JSON persistence"
}
