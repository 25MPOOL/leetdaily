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

variable "guilds_object" {
  description = "Object path for guilds.json"
  type        = string
  default     = "guilds.json"
}

variable "problems_object" {
  description = "Object path for problems.json"
  type        = string
  default     = "problems.json"
}

variable "discord_application_id" {
  description = "Discord application ID for slash command registration"
  type        = string
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
