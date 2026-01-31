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

resource "openprovider_domain_transfer" "example" {
  domain         = "example.com"
  auth_code      = var.auth_code
  owner_handle   = openprovider_customer.owner.handle
  admin_handle   = openprovider_customer.owner.handle
  tech_handle    = openprovider_customer.owner.handle
  billing_handle = openprovider_customer.owner.handle
  autorenew      = true
  ns_group       = "my-ns-group"

  is_private_whois_enabled = true
}
