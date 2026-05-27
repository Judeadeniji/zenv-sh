terraform {
  required_providers {
    zenv = {
      source  = "registry.terraform.io/judeadeniji/zenv"
      version = "~> 1.0.0"
    }
  }
}

provider "zenv" {
  api_url   = "http://localhost:8080"              # Or use ZENV_API_URL env var
  token     = var.zenv_token                       # Pass via TF_VAR_zenv_token
  project   = var.zenv_project                     # Pass via TF_VAR_zenv_project
  env       = "production"
  vault_key = var.zenv_vault_key                   # Pass via TF_VAR_zenv_vault_key
}

variable "zenv_token" {
  type      = string
  sensitive = true
}

variable "zenv_project" {
  type = string
}

variable "zenv_vault_key" {
  type      = string
  sensitive = true
}

# Fetch a single secret
data "zenv_secret" "database_url" {
  name = "DATABASE_URL"
}

# Fetch all secrets
data "zenv_secrets" "all" {}

output "single_secret" {
  value     = data.zenv_secret.database_url.value
  sensitive = true
}

output "all_secrets" {
  value     = data.zenv_secrets.all.secrets
  sensitive = true
}
