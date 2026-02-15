# OpenProvider Terraform Provider

Terraform provider for managing Openprovider domains and customers.

## Requirements

- Terraform >= 1.3

## Features

- **Customer Management**: Create and manage customer handles (contact information) for domain registrations
- **Domain Management**: Register and manage domains with customizable contact handles
- **Nameserver Groups**: Configure and manage nameserver groups for domains

## Usage

```hcl
terraform {
  required_providers {
    openprovider = {
      source  = "charpand/openprovider"
      version = ">= 1.0.0"
    }
  }
}

provider "openprovider" {
  username = var.openprovider_username
  password = var.openprovider_password
}
```

### Creating Customers and Domains

```hcl
# Create a customer (generates a handle like XX123456-XX)
resource "openprovider_customer" "owner" {
  email = "owner@example.com"

  phone {
    country_code = "1"
    area_code    = "555"
    number       = "1234567"
  }

  address {
    street  = "Main Street"
    number  = "100"
    city    = "New York"
    country = "US"
  }

  name {
    first_name = "John"
    last_name  = "Doe"
  }
}

# Use the customer handle in a domain
resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = openprovider_customer.owner.handle
  period       = 1
}
```

### Using Existing Customer Handles

```hcl
# Reference an existing customer by handle
data "openprovider_customer" "existing" {
  handle = "XX123456-XX"
}

resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = data.openprovider_customer.existing.handle
  period       = 1
}
```

## Documentation

Registry docs are generated from templates and examples:

- Templates live in `templates/docs/`.
- Examples live in `examples/`.
- Generated docs live in `docs/`.

To regenerate docs, run (uses `go tool tfplugindocs` under Go 1.24):

```bash
./scripts/docs
```

## Development

```bash
./scripts/format
./scripts/lint
./scripts/test
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
