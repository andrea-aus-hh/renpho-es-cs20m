resource "google_service_account" "google_sheets_account" {
  account_id   = "access-google-sheets"
  display_name = "Access to the Google Sheets"
}

resource "google_cloud_run_service_iam_member" "member" {
  location = google_cloudfunctions2_function.weight_updater_function.location
  service  = google_cloudfunctions2_function.weight_updater_function.name
  role     = "roles/run.invoker"
  member   = "allAuthenticatedUsers"
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

resource "google_cloudfunctions2_function" "weight_updater_function" {
  name        = "weight-updater-function"
  location    = "europe-west8"
  description = "This is the function that will write the weight to the Diary Google Sheet"
  project     = var.project_id
  build_config {
    runtime     = "go123"
    entry_point = "WeightUpdater"
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
  value = google_cloudfunctions2_function.weight_updater_function.service_config[0].uri
}
