# Transitioning from Domain Transfer to Domain Resource

This example demonstrates how to transition a transferred domain from the `openprovider_domain_transfer` 
resource to the `openprovider_domain` resource for ongoing management.

## Step 1: Initial Transfer

First, transfer the domain using the transfer resource:

```hcl
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

resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
  ns_group     = "my-ns-group"
}
```

Apply this configuration and wait for the transfer to complete (5-7 days).

## Step 2: Check Transfer Status

Run `terraform refresh` to update the status:

```bash
terraform refresh
```

Check the status output - when it shows `ACT`, the transfer is complete.

## Step 3: Transition to Domain Resource

Once the transfer is complete (status is `ACT`), you can transition to the regular domain resource:

### 3a. Remove the Transfer Resource from Configuration

Update your Terraform configuration to use `openprovider_domain` instead:

```hcl
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

resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
  ns_group     = "my-ns-group"
}
```

### 3b. Remove Transfer Resource from State

```bash
terraform state rm openprovider_domain_transfer.example
```

### 3c. Import as Domain Resource

Use Terraform's declarative `import` block to bring the domain under management as a regular domain resource:

```hcl
import {
  to = openprovider_domain.example
  id = "example.com"
}
```

Add this import block to your Terraform configuration file alongside the `openprovider_domain` resource definition. When you run `terraform plan`, Terraform will show that it will import the domain into state.

### 3d. Apply Configuration

```bash
terraform apply
```

Terraform will import the domain and should show no changes, confirming the transition is complete. After the import is successful, you can remove the `import` block from your configuration.

## Complete Example

Here's a complete example showing the configuration changes needed for the transition:

**Before (Transfer Configuration):**

```hcl
resource "openprovider_customer" "owner" {
  email = "owner@example.com"
  # ... other customer details
}

resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
  ns_group     = "my-ns-group"
}
```

**After (Domain Resource Configuration with Import):**

```hcl
resource "openprovider_customer" "owner" {
  email = "owner@example.com"
  # ... other customer details (unchanged)
}

# First, remove the old transfer resource and add:
resource "openprovider_domain" "example" {
  domain       = "example.com"
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
  ns_group     = "my-ns-group"
}

# Add this import block temporarily
import {
  to = openprovider_domain.example
  id = "example.com"
}
```

**Steps:**
1. Verify transfer is complete: `terraform refresh` and check that `status = "ACT"`
2. Remove the transfer resource from state: `terraform state rm openprovider_domain_transfer.example`
3. Update your `.tf` files with the new configuration above
4. Run `terraform plan` to see the import operation
5. Run `terraform apply` to complete the import
6. Remove the `import` block from your configuration after successful import

## Notes

- The domain remains in your Openprovider account throughout the transition
- No actual changes are made to the domain in the API during transition
- Both resources manage the same underlying domain, just with different Terraform resources
- After transition, you can manage the domain like any other `openprovider_domain` resource
- The auth_code is no longer needed after the transfer completes
