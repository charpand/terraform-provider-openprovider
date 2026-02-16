---
page_title: "openprovider_domain Resource - terraform-provider-openprovider"
subcategory: ""
description: |-
  Manages an OpenProvider domain. Supports both domain registration and domain transfer.
---

# openprovider_domain (Resource)

Manages an OpenProvider domain. Supports both domain registration and domain transfer. To transfer a domain, provide an auth_code.

## Example Usage

### Domain Registration

#### Basic

```terraform
resource "openprovider_domain" "example" {
  name         = "example.com"
  owner_handle = "owner123"
  period       = 1
}
```

#### With Customer Handle

```terraform
# Create a customer for the domain owner
resource "openprovider_customer" "domain_owner" {
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
    state   = "NY"
    zipcode = "10001"
    country = "US"
  }

  name {
    first_name = "John"
    last_name  = "Doe"
  }
}

# Register a domain using the customer handle
resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = openprovider_customer.domain_owner.handle
  period       = 1
}
```

#### With NS Group (Recommended)

```terraform
resource "openprovider_nsgroup" "my_nameservers" {
  name = "cloudflare-ns"

  nameservers {
    name = "ns1.cloudflare.com"
  }

  nameservers {
    name = "ns2.cloudflare.com"
  }
}

resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = "owner123"
  period       = 1
  ns_group     = openprovider_nsgroup.my_nameservers.name
}
```

#### With DS Records (DNSSEC)

```terraform
resource "openprovider_domain" "dnssec" {
  domain       = "mydomain.com"
  owner_handle = "owner123"
  period       = 1
  autorenew    = true

  # Optional DS records for DNSSEC
  ds_records = [
    {
      algorithm  = 8
      flags      = 257
      protocol   = 3
      public_key = "AwEAAaz/tAm8yTn4Mfeh5eyI96WSVexTBAvkMgJzkKTOiW1vkIbzxeF3+/4RgWOq7HrxRixHlFlExOLAJr5emLvN7SWXgnLh4+B5xQlNVz8Og8kvArMtNROxVQuCaSnIDdD5LKyWbRd2n9WGe2R8PzgCmr3EgVLrjyBxWezF0jLHwVN8efS3rCj/EWgvIWgb9tarpVUDK/b58Da+sqqls3eNbuv7pr+eoZG+SrDK6nWeL3c6H5Apxz7LjVc1uTIdsIXxuOLYA4/ilBmSVIzuDWfdRUfhHdY6+cn8HFRm+2hM8AnXGXws9555KrUB5qihylGa8subX2Nn6UwNR1AkUTV74bU="
    }
  ]
}
```

#### Full (Legacy Nameservers)

```terraform
resource "openprovider_domain" "prod" {
  name           = "mydomain.com"
  owner_handle   = "owner123"
  admin_handle   = "admin456"
  tech_handle    = "tech789"
  billing_handle = "bill001"
  period         = 2
  autorenew      = true
}
```

### Domain Transfer

To transfer a domain, provide an `auth_code` obtained from your current registrar.

#### Basic Transfer

```terraform
# Transfer a domain to OpenProvider with auth code
variable "auth_code" {
  type        = string
  sensitive   = true
  description = "The authorization code from your current registrar"
}

resource "openprovider_customer" "owner" {
  email = "owner@example.com"
  phone = {
    country_code = "1"
    area_code    = "555"
    number       = "1234567"
  }
  address = {
    street  = "Main St"
    number  = "123"
    city    = "New York"
    country = "US"
    zipcode = "10001"
  }
  name = {
    first_name = "John"
    last_name  = "Doe"
  }
}

resource "openprovider_domain" "transferred" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
}
```

#### Transfer with Import Options

```terraform
# Transfer a domain with contact and autorenew settings
variable "auth_code" {
  type        = string
  sensitive   = true
  description = "The authorization code from your current registrar"
}

resource "openprovider_customer" "owner" {
  email = "owner@example.com"
  phone = {
    country_code = "1"
    area_code    = "555"
    number       = "1234567"
  }
  address = {
    street  = "Main St"
    number  = "123"
    city    = "New York"
    country = "US"
    zipcode = "10001"
  }
  name = {
    first_name = "John"
    last_name  = "Doe"
  }
}

resource "openprovider_domain" "transferred" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
}
```

#### Transfer with NS Group

```terraform
# Transfer a domain with a specific nameserver group
variable "auth_code" {
  type        = string
  sensitive   = true
  description = "The authorization code from your current registrar"
}

resource "openprovider_customer" "owner" {
  email = "owner@example.com"
  phone = {
    country_code = "1"
    area_code    = "555"
    number       = "1234567"
  }
  address = {
    street  = "Main St"
    number  = "123"
    city    = "New York"
    country = "US"
    zipcode = "10001"
  }
  name = {
    first_name = "John"
    last_name  = "Doe"
  }
}

resource "openprovider_nsgroup" "dns" {
  name = "my-dns-servers"
  nameservers = [
    { name = "ns1.example.com" },
    { name = "ns2.example.com" },
  ]
}

resource "openprovider_domain" "transferred" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  ns_group     = openprovider_nsgroup.dns.name
  autorenew    = true
}
```

## Important Notes

- **Transfer vs Registration**: The resource automatically detects whether to register or transfer based on the presence of `auth_code`. If `auth_code` is provided, a transfer is initiated; otherwise, a new domain is registered.
- **Transfer Process**: Domain transfers typically take 5-7 days to complete. The resource is created once the transfer is initiated (status: `REQ`), not when it completes (status: `ACT`).
- **Auth Code**: The authorization code (EPP code) must be obtained from your current registrar before initiating the transfer. This field is sensitive and should be stored securely.
- **Delete Behavior**: For transfers, destroying this resource removes it from Terraform state only; the domain remains at OpenProvider.

<!-- schema generated by tfplugindocs -->
## Schema

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `domain` (String) The domain name (e.g., example.com).
- `owner_handle` (String) The owner contact handle for the domain.

### Optional

- `admin_handle` (String) The admin contact handle for the domain.
- `auth_code` (String, Sensitive) The EPP/Authorization code for domain transfer (also known as transfer code or auth code). This is obtained from the current registrar. When provided, the domain will be transferred instead of registered.
- `autorenew` (Boolean) Whether the domain should auto-renew.
- `billing_handle` (String) The billing contact handle for the domain.
- `dnssec_keys` (Attributes List) DNSSEC keys for the domain. Optional. (see [below for nested schema](#nestedatt--dnssec_keys))
- `is_dnssec_enabled` (Boolean) Enable DNSSEC for the domain.
- `ns_group` (String) The nameserver group to use for this domain. Use this instead of nameserver blocks.
- `period` (Number) Registration period in years. Only applicable for domain registration (not transfers).
- `tech_handle` (String) The tech contact handle for the domain.

### Read-Only

- `expiration_date` (String) The domain expiration date.
- `id` (String) The domain identifier (domain name).
- `status` (String) The current status of the domain. Common values: REQ (transfer requested), ACT (active/completed).

<a id="nestedatt--dnssec_keys"></a>
### Nested Schema for `dnssec_keys`

Required:

- `algorithm` (Number) The algorithm number.
- `flags` (Number) The flags field (typically 257 for KSK or 256 for ZSK).
- `protocol` (Number) The protocol field (typically 3 for DNSSEC).
- `public_key` (String) The public key.




## Import

Import a domain using the domain name.

```shell
$ terraform import openprovider_domain.example example.com
```

**Note for Transferred Domains**: If the domain was transferred to OpenProvider, you must provide the `auth_code` in your Terraform configuration after import, or the resource will show a diff on the next plan.
