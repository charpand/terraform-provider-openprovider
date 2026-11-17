# Every order of the account.
data "openprovider_ssl_orders" "all" {}

# The active orders for one common name.
data "openprovider_ssl_orders" "www" {
  common_name = "www.example.com"
  status      = ["ACT"]
}

output "expiration_dates" {
  value = { for order in data.openprovider_ssl_orders.all.orders : order.id => order.expiration_date }
}
