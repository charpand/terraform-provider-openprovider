data "openprovider_contact" "login" {
  id = "123456"
}

output "current_whitelist" {
  value = data.openprovider_contact.login.rcp_allow
}
