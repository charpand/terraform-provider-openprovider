# Send a transfer that failed to the registry again, for example after the
# holder gives a corrected auth code. A change of `triggers` restarts it again.
resource "openprovider_domain_transfer_restart" "example" {
  domain    = "example.com"
  auth_code = var.auth_code

  triggers = {
    attempt = "2"
  }
}
