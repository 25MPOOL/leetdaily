output "cloud_run_job_name" {
  value       = google_cloud_run_v2_job.leetdaily.name
  description = "Provisioned Cloud Run job name"
}

output "cloud_run_job_run_uri" {
  value       = "https://run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${google_cloud_run_v2_job.leetdaily.name}:run"
  description = "Cloud Run job run API URI"
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
