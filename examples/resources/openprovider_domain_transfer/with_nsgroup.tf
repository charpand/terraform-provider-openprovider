resource "openprovider_nsgroup" "cloudflare" {
  name = "cloudflare-ns"
  nameservers = [
    { name = "ns1.cloudflare.com" },
    { name = "ns2.cloudflare.com" }
  ]
}

resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  ns_group     = openprovider_nsgroup.cloudflare.name
}
