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

locals {
  services = toset([
    "iam.googleapis.com",
    "iamcredentials.googleapis.com",
    "cloudresourcemanager.googleapis.com",
    "sts.googleapis.com",
    "serviceusage.googleapis.com",
    "secretmanager.googleapis.com"
  ])
}

resource "google_project_service" "service" {
  for_each = local.services
  project  = var.project_id
  service  = each.value
}