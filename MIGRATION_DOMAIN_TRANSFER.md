# Migration Guide: domain_transfer to domain

The `openprovider_domain_transfer` resource has been consolidated into the `openprovider_domain` resource.

## What Changed?

Previously, domain transfers required using a separate `openprovider_domain_transfer` resource. Now, you can simply add an `auth_code` to the `openprovider_domain` resource to transfer a domain.

## Migration Steps

### Before (using domain_transfer)

```hcl
resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
}
```

### After (using domain)

```hcl
resource "openprovider_domain" "example" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  autorenew    = true
}
```

## State Migration

If you have existing `openprovider_domain_transfer` resources in your state, you'll need to update your Terraform configuration and state:

1. **Update your Terraform configuration** to use `openprovider_domain` instead of `openprovider_domain_transfer`

2. **Update your state** using one of these methods:

   **Option A: Remove and Re-import (Recommended)**
   ```bash
   # Remove old resource from state
   terraform state rm openprovider_domain_transfer.example
   
   # Import as domain resource
   terraform import openprovider_domain.example example.com
   
   # Update configuration to include auth_code (even though import won't retrieve it)
   # Then run terraform plan - there may be a diff for auth_code which is expected
   ```

   **Option B: State Move**
   ```bash
   # Move state from domain_transfer to domain
   terraform state mv openprovider_domain_transfer.example openprovider_domain.example
   ```

3. **Run terraform plan** to verify the migration was successful

## Key Differences

- **Resource Type**: Changed from `openprovider_domain_transfer` to `openprovider_domain`
- **Behavior**: The domain resource now automatically detects whether to register or transfer based on the presence of `auth_code`
- **All Fields Supported**: Transfer-specific fields like `import_contacts_from_registry`, `import_nameservers_from_registry`, and `is_private_whois_enabled` are now part of the domain resource

## Benefits

- **Unified API**: One resource for both domain registration and transfer
- **Simpler Configuration**: No need to decide upfront which resource type to use
- **Consistent Interface**: All domain operations use the same resource type

## Support

If you encounter any issues during migration, please open an issue on the GitHub repository.
