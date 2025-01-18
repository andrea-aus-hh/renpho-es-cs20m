terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 4.34.0"
    }
  }
  backend "gcs" {
    bucket = "apt-octagon-254417-terraform-state"
    prefix = "terraform/state-files"
  }
}

variable "project_id" {
  description = "The GCP project ID"
  type        = string
  default     = "apt-octagon-254417"
}

provider "google" {
  project = var.project_id
  region  = "europe-west8"
}

resource "google_storage_bucket" "terraform_state" {
  name                        = "${var.project_id}-terraform-state"
  location                    = "europe-west8"
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  versioning {
    enabled = true
  }
  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 90
    }
  }
}

resource "google_service_account" "google_sheets_account" {
  account_id   = "access-google-sheets"
  display_name = "Access to the Google Sheets"
}

resource "google_cloud_run_service_iam_member" "member" {
  location = google_cloudfunctions2_function.google_sheet_function.location
  service  = google_cloudfunctions2_function.google_sheet_function.name
  role     = "roles/run.invoker"
  member   = "allAuthenticatedUsers"
}

resource "google_secret_manager_secret" "user_email" {
  secret_id = "user_email_secret"
  replication {
    auto {}
  }
}

data "google_secret_manager_secret_version" "user_email" {
  secret = google_secret_manager_secret.user_email.id
}

resource "google_cloud_run_service_iam_member" "me" {
  location = google_cloudfunctions2_function.google_sheet_function.location
  service  = google_cloudfunctions2_function.google_sheet_function.name
  role     = "roles/run.invoker"
  member   = "user:${data.google_secret_manager_secret_version.user_email.secret_data}"
}

resource "random_id" "default" {
  byte_length = 8
}

resource "google_storage_bucket" "function_code_storage_bucket" {
  name                        = "${random_id.default.hex}-gcf-source"
  location                    = "EU"
  uniform_bucket_level_access = true
}

data "archive_file" "local_archive_function" {
  type        = "zip"
  output_path = "/tmp/function-source.zip"
  source_dir  = "./weightupdater"
}
resource "google_storage_bucket_object" "function_code" {
  name   = "function_code.${data.archive_file.local_archive_function.output_md5}.zip"
  bucket = google_storage_bucket.function_code_storage_bucket.name
  source = data.archive_file.local_archive_function.output_path
}

resource "google_cloudfunctions2_function" "google_sheet_function" {
  name        = "weight-write-function"
  location    = "europe-west8"
  description = "This is the function that will write to the Diary Google Sheet"
  project     = var.project_id
  build_config {
    runtime     = "go123"
    entry_point = "WeightWriter"
    source {
      storage_source {
        bucket = google_storage_bucket.function_code_storage_bucket.name
        object = google_storage_bucket_object.function_code.name
      }
    }
  }

  service_config {
    service_account_email = google_service_account.google_sheets_account.email
  }
}

output "function_uri" {
  value = google_cloudfunctions2_function.google_sheet_function.service_config[0].uri
}

