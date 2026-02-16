# Transfer a domain with multiple contact handles
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

resource "openprovider_customer" "admin" {
  email = "admin@example.com"
  phone = {
    country_code = "1"
    area_code    = "555"
    number       = "7654321"
  }
  address = {
    street  = "Main St"
    number  = "123"
    city    = "New York"
    country = "US"
    zipcode = "10001"
  }
  name = {
    first_name = "Jane"
    last_name  = "Smith"
  }
}

resource "openprovider_domain" "transferred" {
  domain         = "example.com"
  auth_code      = var.auth_code
  owner_handle   = openprovider_customer.owner.handle
  admin_handle   = openprovider_customer.admin.handle
  tech_handle    = openprovider_customer.admin.handle
  billing_handle = openprovider_customer.admin.handle
  autorenew      = true
}
