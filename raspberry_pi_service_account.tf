resource "google_service_account" "pi_invoker" {
  account_id   = "raspberrypiinvoker"
  display_name = "Raspberry Pi Cloud Function Invoker"
}

resource "google_project_iam_binding" "pi_invoker_binding" {
  project = var.project_id
  role    = "roles/cloudfunctions.invoker"
  members = ["serviceAccount:${google_service_account.pi_invoker.email}"]
}
