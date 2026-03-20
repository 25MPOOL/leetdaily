variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "Primary GCP region for IAM-related resources"
  type        = string
  default     = "asia-northeast1"
}

variable "bucket_location" {
  description = "Location of the Terraform state GCS bucket"
  type        = string
  default     = "ASIA-NORTHEAST1"
}

variable "service_name" {
  description = "Service name prefix used for shared bootstrap resources"
  type        = string
  default     = "leetdaily"
}

variable "github_owner" {
  description = "GitHub organization or user name"
  type        = string
}

variable "github_repository" {
  description = "GitHub repository name"
  type        = string
}

variable "terraform_state_bucket_name" {
  description = "Optional override for the Terraform backend bucket name"
  type        = string
  default     = null
}

variable "workload_identity_pool_id" {
  description = "Workload Identity Pool ID"
  type        = string
  default     = "github-actions"
}

variable "workload_identity_provider_id" {
  description = "Workload Identity Provider ID"
  type        = string
  default     = "leetdaily"
}

variable "terraform_admin_roles" {
  description = "Project-level roles granted to the Terraform CI service account"
  type        = list(string)
  default = [
    "roles/cloudscheduler.admin",
    "roles/iam.serviceAccountAdmin",
    "roles/iam.serviceAccountUser",
    "roles/resourcemanager.projectIamAdmin",
    "roles/run.admin",
    "roles/secretmanager.admin",
    "roles/storage.admin",
  ]
}
