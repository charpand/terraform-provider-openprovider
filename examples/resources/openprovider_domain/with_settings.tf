resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = "owner123"

  # Hide the owner data in the WHOIS, where the extension allows it
  # (see is_private_whois_allowed).
  is_private_whois_enabled = true

  # Comments for your own use. An empty string removes them.
  comments = "Managed by Terraform"

  # Nameservers of your own, instead of ns_group.
  nameservers = [
    { name = "ns1.example.net" },
    { name = "ns2.example.net" },
    # A nameserver below the domain itself needs an address.
    { name = "ns3.example.com", ip = "192.0.2.3", ip6 = "2001:db8::3" },
  ]
}
