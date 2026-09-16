# A nameserver named inside the domain it serves needs glue: the registry must
# publish its addresses, because nothing else can resolve the name.
resource "openprovider_glue_record" "ns1" {
  domain    = "example.com"
  subdomain = "ns1"

  ips = ["192.0.2.1", "2001:db8::1"]
}
