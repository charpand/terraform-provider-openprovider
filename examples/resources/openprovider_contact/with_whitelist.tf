resource "openprovider_contact" "ci" {
  username   = "terraform-ci"
  password   = var.ci_contact_password
  email      = "ci@example.com"
  on_destroy = "delete"
}

# The whitelist takes the numeric contact id.
resource "openprovider_rcp_ip_whitelist" "ci" {
  contact_id = tonumber(openprovider_contact.ci.id)
  allow      = ["203.0.113.10"]
}
