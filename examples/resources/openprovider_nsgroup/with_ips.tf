resource "openprovider_nsgroup" "with_ips" {
  name = "my-ns-group-with-ips"

  nameservers {
    name = "ns1.example.com"
    ip   = "192.0.2.1"
    ip6  = "2001:db8::1"
  }

  nameservers {
    name = "ns2.example.com"
    ip   = "192.0.2.2"
    ip6  = "2001:db8::2"
  }
}
