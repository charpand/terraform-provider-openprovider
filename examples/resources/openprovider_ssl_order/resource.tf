resource "openprovider_ssl_order" "www" {
  product_id  = 7
  common_name = "www.example.com"

  # Your own certificate signing request. Without one, OpenProvider generates
  # a key pair and you do not hold the private key.
  csr = file("${path.module}/www.example.com.csr")

  domain_validation_method = "dns"
}

# Empty until the order is active.
output "certificate" {
  value = openprovider_ssl_order.www.certificate
}

output "certificate_chain" {
  value = "${openprovider_ssl_order.www.intermediate_certificate}${openprovider_ssl_order.www.root_certificate}"
}
