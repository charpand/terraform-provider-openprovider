resource "openprovider_dns_template" "web" {
  name = "web"

  records = [
    {
      name  = "www"
      type  = "A"
      value = "192.0.2.1"
      ttl   = 3600
    },
    {
      name     = ""
      type     = "MX"
      value    = "mail.example.com"
      ttl      = 3600
      priority = 10
    },
  ]
}

# A new zone starts with the records of the template.
resource "openprovider_dns_zone" "example" {
  zone_name     = "example.com"
  template_name = openprovider_dns_template.web.name
}
