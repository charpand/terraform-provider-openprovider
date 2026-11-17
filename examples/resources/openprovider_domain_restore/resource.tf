# What the restore costs.
data "openprovider_domain_price" "restore" {
  domain    = "example.com"
  operation = "restore"
}

# Restore example.com, once. The account is charged.
resource "openprovider_domain_restore" "example" {
  domain = "example.com"

  # Change a value to restore again.
  triggers = {
    restore = "2026-11"
  }
}
