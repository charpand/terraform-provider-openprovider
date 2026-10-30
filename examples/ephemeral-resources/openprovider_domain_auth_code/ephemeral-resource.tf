# The auth code lets a different registrar transfer the domain. Terraform does
# not keep it in the plan or the state, so give it only to an ephemeral or
# write-only argument.
ephemeral "openprovider_domain_auth_code" "example" {
  domain = "example.com"
}
