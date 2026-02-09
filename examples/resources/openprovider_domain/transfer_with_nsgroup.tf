# Transfer a domain with a specific nameserver group
variable "auth_code" {
  type        = string
  sensitive   = true
  description = "The authorization code from your current registrar"
}

resource "openprovider_customer" "owner" {
  email = "owner@example.com"
  phone = {
    country_code = "1"
    area_code    = "555"
    number       = "1234567"
  }
  address = {
    street  = "Main St"
    number  = "123"
    city    = "New York"
    country = "US"
    zipcode = "10001"
  }
  name = {
    first_name = "John"
    last_name  = "Doe"
  }
}

resource "openprovider_nsgroup" "dns" {
  name = "my-dns-servers"
  nameservers = [
    { name = "ns1.example.com" },
    { name = "ns2.example.com" },
  ]
}

resource "openprovider_domain" "transferred" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  ns_group     = openprovider_nsgroup.dns.name
  autorenew    = true
}
