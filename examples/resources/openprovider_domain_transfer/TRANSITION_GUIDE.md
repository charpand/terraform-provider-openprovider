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

```bash
terraform import openprovider_domain.example example.com
```

### 3d. Apply Configuration

```bash
terraform apply
```

Terraform should show no changes, confirming the transition is complete.

## Complete Example Script

Here's a shell script that automates the transition:

```bash
#!/bin/bash

DOMAIN="example.com"

echo "Checking transfer status..."
terraform refresh

# Check if status is ACT (transfer complete)
STATUS=$(terraform show -json | jq -r '.values.root_module.resources[] | select(.address=="openprovider_domain_transfer.example") | .values.status')

if [ "$STATUS" != "ACT" ]; then
    echo "Transfer not complete yet. Status: $STATUS"
    echo "Please wait for the transfer to complete (status: ACT) before transitioning."
    exit 1
fi

echo "Transfer complete! Starting transition..."

# Remove transfer resource from state
echo "Removing transfer resource from state..."
terraform state rm openprovider_domain_transfer.example

# Import as domain resource
echo "Importing as domain resource..."
terraform import openprovider_domain.example "$DOMAIN"

# Verify no changes needed
echo "Verifying configuration..."
terraform plan

echo "Transition complete!"
```

## Notes

- The domain remains in your Openprovider account throughout the transition
- No actual changes are made to the domain in the API during transition
- Both resources manage the same underlying domain, just with different Terraform resources
- After transition, you can manage the domain like any other `openprovider_domain` resource
- The auth_code is no longer needed after the transfer completes
