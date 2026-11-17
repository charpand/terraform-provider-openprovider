data "openprovider_dns_zones" "all" {}

output "zone_names" {
  value = [for zone in data.openprovider_dns_zones.all.zones : zone.zone_name]
}
