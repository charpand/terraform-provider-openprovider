data "openprovider_contact" "login" {
  id = "123456"
}

resource "openprovider_rcp_ip_whitelist" "login" {
  contact_id = data.openprovider_contact.login.id

  # Include every address you apply from, CI runners too: an address that is
  # not on the list is refused with error 10005 (Access Denied).
  allow = concat(
    ["203.0.113.10", "2001:db8:1::/48"],
    var.ci_runner_ips,
  )
}

variable "ci_runner_ips" {
  type    = list(string)
  default = []
}
