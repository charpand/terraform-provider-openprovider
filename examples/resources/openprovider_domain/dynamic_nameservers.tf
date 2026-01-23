variable "domain_name" {
  description = "The domain name to manage"
  type        = string
  default     = "example.com"
}

variable "nameservers" {
  description = "List of nameserver hostnames"
  type        = list(string)
  default = [
    "chad.ns.cloudflare.com",
    "sandy.ns.cloudflare.com"
  ]

  validation {
    condition     = length(var.nameservers) >= 2
    error_message = "At least two nameservers are required."
  }
}

resource "openprovider_domain" "domain" {
  domain       = var.domain_name
  owner_handle = "owner123"

  dynamic "nameserver" {
    for_each = var.nameservers
    content {
      hostname = nameserver.value
    }
  }
}
