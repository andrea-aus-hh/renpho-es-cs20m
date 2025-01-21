resource "google_service_account" "pi_invoker" {
  account_id   = "raspberrypiinvoker"
  display_name = "Raspberry Pi Cloud Function Invoker"
}

resource "google_cloudfunctions2_function_iam_member" "invoker" {
  project        = google_cloudfunctions2_function.weight_updater_function.project
  location       = google_cloudfunctions2_function.weight_updater_function.location
  cloud_function = google_cloudfunctions2_function.weight_updater_function.name
  role           = "roles/cloudfunctions.invoker"
  member         = "serviceAccount:${google_service_account.pi_invoker.email}"
}
