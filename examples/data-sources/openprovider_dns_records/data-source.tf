data "openprovider_dns_records" "example" {
  zone_name = "example.com"
}

# Check for a record before it is added.
locals {
  has_www = anytrue([
    for record in data.openprovider_dns_records.example.records :
    record.type == "A" && startswith(record.name, "www")
  ])
}
