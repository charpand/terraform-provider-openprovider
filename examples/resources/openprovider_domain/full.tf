resource "openprovider_domain" "prod" {
  domain         = "mydomain.com"
  owner_handle   = "owner123"
  admin_handle   = "admin456"
  tech_handle    = "tech789"
  billing_handle = "bill001"
  period         = 2
  autorenew      = true

  nameserver {
    hostname = "ns1.example.com"
  }

  nameserver {
    hostname = "ns2.example.com"
  }
}
