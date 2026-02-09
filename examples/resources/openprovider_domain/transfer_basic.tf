# Transfer a domain to OpenProvider with auth code
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

resource "openprovider_domain" "transferred" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
}
