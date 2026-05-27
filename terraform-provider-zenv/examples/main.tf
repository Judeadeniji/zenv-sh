terraform {
  required_providers {
    zenv = {
      source  = "registry.terraform.io/judeadeniji/zenv"
      version = "~> 1.0.0"
    }
  }
}

provider "zenv" {
  api_url   = "http://localhost:8080" # Pointing to local dev server
  token     = "zenv_dev_token_123"    # Your service token
  project   = "prj_0190123"           # Your project ID
  env       = "production"
  vault_key = "my_secure_passphrase"
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
