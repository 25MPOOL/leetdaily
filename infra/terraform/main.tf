terraform {
  required_version = "1.14.7"

  backend "gcs" {}

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "6.50.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

import {
  to = google_service_account.leetdaily_runtime
  id = "projects/${var.project_id}/serviceAccounts/${var.service_name}-runtime@${var.project_id}.iam.gserviceaccount.com"
}

import {
  to = google_storage_bucket.leetdaily_data
  id = "${var.project_id}/${coalesce(var.bucket_name, "${var.project_id}-${var.service_name}-data")}"
}

import {
  to = google_service_account.scheduler_invoker
  id = "projects/${var.project_id}/serviceAccounts/${var.service_name}-scheduler@${var.project_id}.iam.gserviceaccount.com"
}

locals {
  service_name          = var.service_name
  bucket_name           = coalesce(var.bucket_name, "${var.project_id}-${var.service_name}-data")
  scheduler_job_run_url = "https://run.googleapis.com/v2/projects/${var.project_id}/locations/${var.region}/jobs/${google_cloud_run_v2_job.leetdaily.name}:run"
  common_labels = {
    app         = "leetdaily"
    environment = var.environment
  }
}

resource "google_service_account" "leetdaily_runtime" {
  account_id   = "${var.service_name}-runtime"
  display_name = "LeetDaily runtime"
}

resource "google_storage_bucket" "leetdaily_data" {
  name                        = local.bucket_name
  location                    = var.bucket_location
  uniform_bucket_level_access = true
  force_destroy               = false
  labels                      = local.common_labels
}

resource "google_project_iam_member" "runtime_secret_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.leetdaily_runtime.email}"
}

resource "google_storage_bucket_iam_member" "runtime_bucket_admin" {
  bucket = google_storage_bucket.leetdaily_data.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.leetdaily_runtime.email}"
}

resource "google_cloud_run_v2_job" "leetdaily" {
  name                = local.service_name
  location            = var.region
  labels              = local.common_labels
  deletion_protection = false

  template {
    template {
      service_account = google_service_account.leetdaily_runtime.email
      timeout         = "${var.cloud_run_timeout_seconds}s"
      max_retries     = 0

      containers {
        image = var.container_image

        env {
          name  = "LEETDAILY_RUNTIME"
          value = "job"
        }

        env {
          name  = "GCS_BUCKET"
          value = google_storage_bucket.leetdaily_data.name
        }

        env {
          name  = "CONFIG_OBJECT"
          value = var.config_object
        }

        env {
          name  = "GUILDS_OBJECT"
          value = var.guilds_object
        }

        env {
          name  = "STATE_OBJECT"
          value = var.state_object
        }

        env {
          name  = "PROBLEMS_OBJECT"
          value = var.problems_object
        }

        env {
          name = "DISCORD_BOT_TOKEN"
          value_source {
            secret_key_ref {
              secret  = var.discord_token_secret_id
              version = "latest"
            }
          }
        }

        resources {
          limits = {
            cpu    = var.cloud_run_cpu
            memory = var.cloud_run_memory
          }
        }
      }
    }
  }
}

resource "google_cloud_run_v2_job_iam_member" "scheduler_invoker" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_job.leetdaily.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler_invoker.email}"
}

resource "google_service_account" "scheduler_invoker" {
  account_id   = "${var.service_name}-scheduler"
  display_name = "LeetDaily scheduler invoker"
}

resource "google_cloud_scheduler_job" "daily_run" {
  name        = "${var.service_name}-daily-run"
  description = "Trigger LeetDaily daily posting job"
  region      = var.region
  schedule    = "0 5 * * *"
  time_zone   = "Asia/Tokyo"

  retry_config {
    retry_count = 0
  }

  http_target {
    http_method = "POST"
    uri         = local.scheduler_job_run_url

    oauth_token {
      service_account_email = google_service_account.scheduler_invoker.email
    }
  }
}
