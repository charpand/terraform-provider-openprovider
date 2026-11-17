data "openprovider_domains" "example" {
  full_name = "example.com"
}

# Renew example.com for two years, once. The account is charged.
resource "openprovider_domain_renewal" "example" {
  domain = "example.com"
  period = 2

  # Change a value to renew again.
  triggers = {
    renewal = "2027"
  }
}

output "can_renew" {
  value = data.openprovider_domains.example.domains[0].can_renew
}
