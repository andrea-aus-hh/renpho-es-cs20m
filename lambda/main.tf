terraform {
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 4.34.0"
    }
  }
}

provider "google" {
  project = "apt-octagon-254417"
  region  = "europe-west8"
}

resource "google_service_account" "service_account_default" {
  account_id   = "example-service-account"
  display_name = "Access to the Google Sheets"
}

resource "google_cloud_run_service_iam_member" "member" {
  location = google_cloudfunctions2_function.my_function.location
  service  = google_cloudfunctions2_function.my_function.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "random_id" "default" {
  byte_length = 8
}

resource "google_storage_bucket" "default" {
  name                        = "${random_id.default.hex}-gcf-source"
  location                    = "EU"
  uniform_bucket_level_access = true
}

data "archive_file" "default" {
  type        = "zip"
  output_path = "/tmp/function-source.zip"
  source_dir  = "."
}
resource "google_storage_bucket_object" "object" {
  name   = "h.zip"
  bucket = google_storage_bucket.default.name
  source = data.archive_file.default.output_path
}

resource "google_cloudfunctions2_function" "my_function" {
  name        = "google-sheets-function"
  location    = "europe-west8"
  description = "a new function"
  build_config {
    runtime = "go123"
    entry_point = "HelloHTTP"
    source {
      storage_source {
        bucket = google_storage_bucket.default.name
        object = google_storage_bucket_object.object.name
      }
    }
  }

  service_config {

    service_account_email = google_service_account.service_account_default.email
  }
}

output "function_uri" {
  value = google_cloudfunctions2_function.my_function.service_config[0].uri
}

