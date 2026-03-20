terraform {
  required_version = ">= 1.6.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.46"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

locals {
  repository               = "${var.github_owner}/${var.github_repository}"
  terraform_state_bucket   = coalesce(var.terraform_state_bucket_name, "${var.project_id}-${var.service_name}-tfstate")
  terraform_ci_account_id  = "${var.service_name}-terraform-ci"
  terraform_ci_member      = "serviceAccount:${google_service_account.terraform_ci.email}"
  workload_identity_member = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${local.repository}"
}

resource "google_storage_bucket" "terraform_state" {
  name                        = local.terraform_state_bucket
  location                    = var.bucket_location
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }
}

resource "google_service_account" "terraform_ci" {
  account_id   = local.terraform_ci_account_id
  display_name = "LeetDaily Terraform CI"
}

resource "google_project_iam_member" "terraform_ci_roles" {
  for_each = toset(var.terraform_admin_roles)

  project = var.project_id
  role    = each.value
  member  = local.terraform_ci_member
}

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = var.workload_identity_pool_id
  display_name              = "GitHub Actions"
  description               = "Federated identity for GitHub Actions in ${local.repository}"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = var.workload_identity_provider_id
  display_name                       = "GitHub Actions ${local.repository}"
  description                        = "OIDC provider for ${local.repository}"
  attribute_condition                = "assertion.repository == '${local.repository}'"

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.actor"      = "assertion.actor"
    "attribute.ref"        = "assertion.ref"
    "attribute.repository" = "assertion.repository"
  }

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "terraform_ci_workload_identity_user" {
  service_account_id = google_service_account.terraform_ci.name
  role               = "roles/iam.workloadIdentityUser"
  member             = local.workload_identity_member
}
