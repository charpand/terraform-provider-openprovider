resource "openprovider_nsgroup" "my_nameservers" {
  name = "cloudflare-ns"

  nameservers {
    name = "ns1.cloudflare.com"
  }

  nameservers {
    name = "ns2.cloudflare.com"
  }
}

resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = "owner123"
  period       = 1
  ns_group     = openprovider_nsgroup.my_nameservers.name
}
