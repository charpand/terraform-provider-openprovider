resource "openprovider_rcp_ip_whitelist" "login" {
  contact_id = 123456
  allow      = ["203.0.113.10"]

  # Destroying this resource empties the list, which switches the whitelist
  # off and opens the control panel to every address.
  on_destroy = "clear"
}
