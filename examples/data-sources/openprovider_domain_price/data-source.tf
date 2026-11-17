# What a registration of example.com for two years costs.
data "openprovider_domain_price" "registration" {
  domain = "example.com"
  period = 2
}

# What a transfer costs.
data "openprovider_domain_price" "transfer" {
  domain    = "example.com"
  operation = "transfer"
}

output "registration_price" {
  value = "${data.openprovider_domain_price.registration.price} ${data.openprovider_domain_price.registration.currency}"
}
