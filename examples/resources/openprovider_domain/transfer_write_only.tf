# Transfer a domain with an auth code that stays out of the plan and the state.
# Write-only arguments need Terraform 1.11 or OpenTofu 1.11, or later.
variable "auth_code" {
  type        = string
  sensitive   = true
  ephemeral   = true
  description = "The authorization code from your current registrar"
}

resource "openprovider_domain" "transferred" {
  domain       = "example.com"
  auth_code_wo = var.auth_code
  # Increase this to send the transfer again with a new code.
  auth_code_wo_version = 1
  owner_handle         = "XX123456-XX"
  autorenew            = true
}
