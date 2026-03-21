mock_provider "google" {}

variables {
  project_id           = "leetdaily-test"
  github_owner         = "25MPOOL"
  github_repository    = "leetdaily"
  github_repository_id = "123456789"
}

run "naming_contract" {
  command = plan

  assert {
    condition     = output.terraform_plan_service_account_account_id == "leetdaily-terraform-plan"
    error_message = "terraform-plan service account account_id must remain stable"
  }

  assert {
    condition     = output.terraform_apply_service_account_account_id == "leetdaily-terraform-apply"
    error_message = "terraform-apply service account account_id must remain stable"
  }

  assert {
    condition     = output.terraform_plan_workload_identity_provider_id == "leetdaily-terraform-plan"
    error_message = "terraform-plan provider ID must remain stable"
  }

  assert {
    condition     = output.terraform_apply_workload_identity_provider_id == "leetdaily-terraform-apply"
    error_message = "terraform-apply provider ID must remain stable"
  }
}
