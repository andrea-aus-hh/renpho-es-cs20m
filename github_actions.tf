locals {
  roles = [
    "roles/resourcemanager.projectIamAdmin",
    "roles/editor",
  ]
  github_repository_name = "andrea-aus-hh/renpho-es-cs20m"
}

resource "google_service_account" "github_actions" {
  project      = var.project_id
  account_id   = "github-actions"
  display_name = "GitHub Actions"
  description  = "link to Workload Identity Pool used by GitHub Actions"
}

resource "google_project_iam_member" "roles" {
  project = var.project_id
  for_each = {
    for role in local.roles : role => role
  }
  role   = each.value
  member = "serviceAccount:${google_service_account.github_actions.email}"
}

resource "google_iam_workload_identity_pool" "github_identity_pool" {
  provider                  = google-beta
  project                   = var.project_id
  workload_identity_pool_id = "github-actions-pool"
  display_name              = "github"
  description               = "Identity pool for GitHub Actions"
}

resource "google_iam_workload_identity_pool_provider" "github_pool_provider" {
  provider                           = google-beta
  project                            = var.project_id
  workload_identity_pool_id          = google_iam_workload_identity_pool.github_identity_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  display_name                       = "github actions provider"
  description                        = "OIDC identity pool provider for execute GitHub Actions"
  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
    "attribute.ref"        = "assertion.ref"
    "attribute.workflow"   = "assertion.workflow"
  }
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  attribute_condition = "attribute.repository == \"${local.github_repository_name}\" && attribute.ref == \"refs/heads/master\" && attribute.workflow == \"Terraform\""
}

resource "google_service_account_iam_member" "github_actions" {
  service_account_id = google_service_account.github_actions.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github_identity_pool.name}/attribute.repository/${local.github_repository_name}"
}

resource "google_storage_bucket_iam_member" "terraform_state_access" {
  bucket = google_storage_bucket.terraform_state.name
  role   = "roles/storage.objectAdmin"

  member = "serviceAccount:${google_service_account.github_actions.email}"  # Replace with your service account email
}


output "service_account_github_actions_email" {
  description = "Service Account used by GitHub Actions"
  value       = google_service_account.github_actions.email
}

output "google_iam_workload_identity_pool_provider_github_name" {
  description = "Workload Identity Pood Provider ID"
  value       = google_iam_workload_identity_pool_provider.github_pool_provider.name
}