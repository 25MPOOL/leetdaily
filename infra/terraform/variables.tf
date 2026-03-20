variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "region" {
  description = "Primary GCP region for Cloud Run and Scheduler"
  type        = string
  default     = "asia-northeast1"
}

variable "bucket_location" {
  description = "GCS bucket location"
  type        = string
  default     = "ASIA-NORTHEAST1"
}

variable "environment" {
  description = "Deployment environment label"
  type        = string
  default     = "prod"
}

variable "service_name" {
  description = "Cloud Run service name"
  type        = string
  default     = "leetdaily"
}

variable "container_image" {
  description = "Container image URL for Cloud Run"
  type        = string
}

variable "bucket_name" {
  description = "Optional override for the data bucket name"
  type        = string
  default     = null
}

variable "discord_token_secret_id" {
  description = "Secret Manager secret ID containing the Discord bot token"
  type        = string
}

variable "config_object" {
  description = "Object path for config.json"
  type        = string
  default     = "config.json"
}

variable "state_object" {
  description = "Object path for state.json"
  type        = string
  default     = "state.json"
}

variable "problems_object" {
  description = "Object path for problems.json"
  type        = string
  default     = "problems.json"
}

variable "container_port" {
  description = "Container port exposed by the app"
  type        = number
  default     = 8080
}

variable "cloud_run_timeout_seconds" {
  description = "Cloud Run request timeout in seconds"
  type        = number
  default     = 900
}

variable "cloud_run_cpu" {
  description = "Cloud Run CPU limit"
  type        = string
  default     = "1"
}

variable "cloud_run_memory" {
  description = "Cloud Run memory limit"
  type        = string
  default     = "512Mi"
}

variable "cloud_run_max_instances" {
  description = "Cloud Run max instances"
  type        = number
  default     = 1
}
