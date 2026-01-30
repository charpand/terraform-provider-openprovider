resource "openprovider_domain_transfer" "example" {
  domain                            = "example.com"
  auth_code                         = var.auth_code
  owner_handle                      = openprovider_customer.owner.handle
  import_contacts_from_registry     = true
  import_nameservers_from_registry  = true
}
