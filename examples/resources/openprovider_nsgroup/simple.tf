resource "openprovider_nsgroup" "example" {
  name = "my-ns-group"

  nameservers {
    name = "ns1.example.com"
  }

  nameservers {
    name = "ns2.example.com"
  }
}
