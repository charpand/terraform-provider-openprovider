variable "auth_code" {
  description = "Domain authorization code from current registrar"
  type        = string
  sensitive   = true
}

resource "openprovider_customer" "owner" {
  email = "owner@example.com"

  phone {
    country_code = "1"
    area_code    = "555"
    number       = "1234567"
  }

  address {
    street  = "Main Street"
    number  = "100"
    city    = "New York"
    country = "US"
  }

  name {
    first_name = "John"
    last_name  = "Doe"
  }
}

resource "openprovider_nsgroup" "cloudflare" {
  name = "cloudflare-ns"
  nameservers = [
    { name = "ns1.cloudflare.com" },
    { name = "ns2.cloudflare.com" }
  ]
}

resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  ns_group     = openprovider_nsgroup.cloudflare.name
}
