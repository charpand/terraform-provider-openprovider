resource "openprovider_domain_transfer" "example" {
  domain                            = "example.com"
  auth_code                         = var.auth_code
  # When import options are enabled, contact handles will be imported from the registry
  # Provide a placeholder or actual handle as required by your registrar
  owner_handle                      = "placeholder-handle"
  import_contacts_from_registry     = true
  import_nameservers_from_registry  = true
}
