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

locals {
  service_name          = var.service_name
  bucket_name           = coalesce(var.bucket_name, "${var.project_id}-${var.service_name}-data")
  scheduler_service_url = "${google_cloud_run_v2_service.leetdaily.uri}/run"
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

resource "google_cloud_run_v2_service" "leetdaily" {
  name     = local.service_name
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"
  labels   = local.common_labels

  template {
    service_account = google_service_account.leetdaily_runtime.email
    timeout         = "${var.cloud_run_timeout_seconds}s"

    containers {
      image = var.container_image

      env {
        name  = "LEETDAILY_RUNTIME"
        value = "http"
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

      ports {
        container_port = var.container_port
      }

      resources {
        cpu_idle = true
        limits = {
          cpu    = var.cloud_run_cpu
          memory = var.cloud_run_memory
        }
      }
    }

    scaling {
      min_instance_count = 0
      max_instance_count = var.cloud_run_max_instances
    }
  }
}

resource "google_cloud_run_service_iam_member" "invoker_public" {
  service  = google_cloud_run_v2_service.leetdaily.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_service_account" "scheduler_invoker" {
  account_id   = "${var.service_name}-scheduler"
  display_name = "LeetDaily scheduler invoker"
}

resource "google_cloud_run_service_iam_member" "scheduler_invoker" {
  service  = google_cloud_run_v2_service.leetdaily.name
  location = var.region
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.scheduler_invoker.email}"
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
    uri         = local.scheduler_service_url

    oidc_token {
      service_account_email = google_service_account.scheduler_invoker.email
      audience              = google_cloud_run_v2_service.leetdaily.uri
    }
  }
}
