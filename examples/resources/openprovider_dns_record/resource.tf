resource "openprovider_dns_record" "www" {
  zone_name = "example.com"
  name      = "www"
  type      = "A"
  value     = "192.0.2.1"
  ttl       = 3600
}
