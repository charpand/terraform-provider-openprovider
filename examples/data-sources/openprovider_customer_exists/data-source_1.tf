data "openprovider_customer_exists" "owner" {
  handle = "XX123456-XX"
}

output "owner_handle_exists" {
  value = data.openprovider_customer_exists.owner.exists
}
