variable "ci_contact_password" {
  type      = string
  sensitive = true
}

resource "openprovider_contact" "ci" {
  username = "terraform-ci"

  # Write-only: sent when the contact is created and never stored in the
  # plan or state. Needs Terraform 1.11 or later.
  password = var.ci_contact_password

  email  = "ci@example.com"
  locale = "en_GB"

  name = {
    first_name = "Terraform"
    last_name  = "CI"
  }

  phone = {
    country_code = "+31"
    area_code    = "20"
    number       = "1234567"
  }

  address = {
    street  = "Main Street"
    number  = "1"
    zipcode = "1000AA"
    city    = "Amsterdam"
    country = "NL"
  }
}
