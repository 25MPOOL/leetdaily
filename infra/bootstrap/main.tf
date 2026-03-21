terraform {
  required_version = "1.14.7"

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
  repository             = "${var.github_owner}/${var.github_repository}"
  terraform_state_bucket = coalesce(var.terraform_state_bucket_name, "${var.project_id}-${var.service_name}-tfstate")
  naming = {
    terraform_plan = {
      service_account_account_id         = "${var.service_name}-terraform-plan"
      service_account_display_name       = "LeetDaily Terraform Plan"
      workload_identity_provider_id      = "${var.service_name}-terraform-plan"
      workload_identity_provider_name    = "terraform-plan"
      workload_identity_provider_desc    = "OIDC provider for terraform-plan in ${local.repository}"
      workload_identity_attribute_filter = "assertion.repository_id == '${var.github_repository_id}' && assertion.event_name == 'pull_request'"
    }
    terraform_apply = {
      service_account_account_id         = "${var.service_name}-terraform-apply"
      service_account_display_name       = "LeetDaily Terraform Apply"
      workload_identity_provider_id      = "${var.service_name}-terraform-apply"
      workload_identity_provider_name    = "terraform-apply"
      workload_identity_provider_desc    = "OIDC provider for terraform-apply in ${local.repository}"
      workload_identity_attribute_filter = "assertion.repository_id == '${var.github_repository_id}' && ( assertion.ref == 'refs/heads/main' || starts_with(assertion.ref, 'refs/tags/') ) && (assertion.event_name == 'workflow_dispatch' || assertion.event_name == 'push')"
    }
  }

  terraform_service_accounts = {
    plan = {
      account_id   = local.naming.terraform_plan.service_account_account_id
      display_name = local.naming.terraform_plan.service_account_display_name
      roles        = var.terraform_plan_roles
    }
    apply = {
      account_id   = local.naming.terraform_apply.service_account_account_id
      display_name = local.naming.terraform_apply.service_account_display_name
      roles        = var.terraform_apply_roles
    }
  }

  workload_identity_providers = {
    plan = {
      provider_id         = local.naming.terraform_plan.workload_identity_provider_id
      display_name        = "GitHub Actions ${local.repository} ${local.naming.terraform_plan.workload_identity_provider_name}"
      description         = local.naming.terraform_plan.workload_identity_provider_desc
      attribute_condition = local.naming.terraform_plan.workload_identity_attribute_filter
    }
    apply = {
      provider_id         = local.naming.terraform_apply.workload_identity_provider_id
      display_name        = "GitHub Actions ${local.repository} ${local.naming.terraform_apply.workload_identity_provider_name}"
      description         = local.naming.terraform_apply.workload_identity_provider_desc
      attribute_condition = local.naming.terraform_apply.workload_identity_attribute_filter
    }
  }
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
  for_each = local.terraform_service_accounts

  account_id   = each.value.account_id
  display_name = each.value.display_name
}

resource "google_project_iam_member" "terraform_ci_roles" {
  for_each = merge([
    for workflow_name, config in local.terraform_service_accounts : {
      for role in config.roles : "${workflow_name}:${role}" => {
        project = var.project_id
        role    = role
        member  = "serviceAccount:${google_service_account.terraform_ci[workflow_name].email}"
      }
    }
  ]...)

  project = each.value.project
  role    = each.value.role
  member  = each.value.member
}

resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = var.workload_identity_pool_id
  display_name              = "GitHub Actions"
  description               = "Federated identity for GitHub Actions in ${local.repository}"
}

resource "google_iam_workload_identity_pool_provider" "github" {
  for_each = local.workload_identity_providers

  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = each.value.provider_id
  display_name                       = each.value.display_name
  description                        = each.value.description
  attribute_condition                = each.value.attribute_condition

  attribute_mapping = {
    "google.subject"          = "assertion.sub"
    "attribute.actor"         = "assertion.actor"
    "attribute.event_name"    = "assertion.event_name"
    "attribute.ref"           = "assertion.ref"
    "attribute.repository"    = "assertion.repository"
    "attribute.repository_id" = "assertion.repository_id"
    "attribute.workflow_ref"  = "assertion.workflow_ref"
  }

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "terraform_ci_workload_identity_user" {
  for_each = local.workload_identity_providers

  service_account_id = google_service_account.terraform_ci[each.key].name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool_provider.github[each.key].name}/attribute.repository_id/${var.github_repository_id}"
}

resource "google_storage_bucket_iam_member" "terraform_state_backend_access" {
  for_each = local.terraform_service_accounts

  bucket = google_storage_bucket.terraform_state.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.terraform_ci[each.key].email}"
}
