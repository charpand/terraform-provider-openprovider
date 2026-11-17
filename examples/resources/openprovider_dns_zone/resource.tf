resource "openprovider_dns_zone" "example" {
  zone_name = "example.com"

  # Set to true to let `terraform destroy` delete the zone and all of its
  # records at OpenProvider. By default the zone is only forgotten.
  allow_deletion = false
}

resource "openprovider_dns_record" "www" {
  zone_name = openprovider_dns_zone.example.zone_name
  name      = "www"
  type      = "A"
  value     = "192.0.2.1"
}
