# Domain Transfer Investigation - Quick Reference

## TL;DR

✅ **Domain transfer support is feasible and recommended**

**Approach:** Create new `openprovider_domain_transfer` resource

**API Available:** Openprovider provides `POST /v1beta/domains/transfer` endpoint

**Effort Estimate:** 15-20 hours for complete implementation

## Key Decision: Separate Resource

Create `openprovider_domain_transfer` resource (not extend existing `openprovider_domain`)

**Why separate?**
- Clear intent (transfer vs. register)
- Easier to handle transfer-specific parameters
- No risk to existing domain resource
- Follows Terraform best practices

## Resource Schema (Quick View)

```hcl
resource "openprovider_domain_transfer" "example" {
  # Required
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
  
  # Optional
  admin_handle   = openprovider_customer.admin.handle
  tech_handle    = openprovider_customer.tech.handle
  billing_handle = openprovider_customer.billing.handle
  autorenew      = true
  ns_group       = "my-ns-group"
  
  # Computed
  id              = "123456"     # read-only
  status          = "REQ"        # read-only
  expiration_date = "2025-12-31" # read-only
}
```

## API Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v1beta/domains/transfer` | POST | Initiate transfer |
| `/v1beta/domains/{id}` | GET | Get domain status |
| `/v1beta/domains` | GET | List domains (for lookup) |

## Implementation Checklist

- [ ] **Phase 1:** API client (`internal/client/domains/transfer.go`)
- [ ] **Phase 2:** Provider resource (`internal/provider/resource_domain_transfer.go`)
- [ ] **Phase 3:** Full features (update, import, validation)
- [ ] **Phase 4:** Documentation and examples
- [ ] **Phase 5:** Testing and QA

## Key Features

### In Scope ✅
- Initiate domain transfer with auth code
- Configure contact handles
- Set nameservers (via ns_group or nameserver blocks)
- Track transfer status
- Import existing transfers
- WHOIS privacy configuration

### Out of Scope ❌
- Transfer approval (outbound from Openprovider)
- Auth code retrieval
- Bulk transfers
- Active status polling
- Advanced TLD-specific fields (initial release)

## Transfer Workflow

```
User obtains auth code from current registrar
         ↓
User creates Terraform configuration
         ↓
Terraform calls POST /v1beta/domains/transfer
         ↓
Openprovider initiates transfer (5-7 days)
         ↓
User runs terraform refresh to check status
         ↓
Domain active at Openprovider
```

## Important Notes

1. **Async Operation:** Transfer takes 5-7 days; resource creation succeeds when initiated, not completed
2. **Status Tracking:** Check `status` attribute (`REQ` → `ACT`)
3. **Security:** Auth code is sensitive; use variables and encrypted state backend
4. **Delete Behavior:** Destroy removes from state only, does not delete domain
5. **No Breaking Changes:** Existing resources unaffected

## Example Usage

### Basic Transfer
```hcl
resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.auth_code
  owner_handle = openprovider_customer.owner.handle
}
```

### Import Existing Transfer
```bash
terraform import openprovider_domain_transfer.example example.com
```

## Testing Strategy

- **Unit Tests:** Mock API responses with Prism server
- **Integration Tests:** Full resource lifecycle with mock transport
- **Manual Tests:** (Optional) Real API validation

## Documentation Required

1. Resource template (`templates/docs/resources/domain_transfer.md.tmpl`)
2. Examples (`examples/resources/openprovider_domain_transfer/*.tf`)
3. API documentation update (`API.md`)
4. Import instructions

## Success Criteria

- ✅ User can transfer domain with auth code
- ✅ Contact handles configurable
- ✅ Nameservers configurable
- ✅ Transfer status tracked
- ✅ Import works
- ✅ All tests pass
- ✅ Clear documentation
- ✅ No breaking changes

## Files to Create

```
internal/client/domains/
  ├── transfer.go          # API client
  └── transfer_test.go     # Unit tests

internal/provider/
  ├── resource_domain_transfer.go  # Resource implementation
  └── models_domain_transfer.go    # Schema models

templates/docs/resources/
  └── domain_transfer.md.tmpl      # Documentation template

examples/resources/openprovider_domain_transfer/
  ├── basic.tf             # Basic example
  ├── complete.tf          # Full example
  ├── with_nsgroup.tf      # NS group example
  └── import.sh            # Import example
```

## Risk Level: LOW

- ✅ API well documented
- ✅ No changes to existing code
- ✅ Clear separation of concerns
- ✅ Standard Terraform patterns
- ✅ Comprehensive test coverage planned

## Next Action

Review and approve this investigation, then create GitHub issue from `DRAFT_ISSUE_DOMAIN_TRANSFER.md` for implementation.

---

**Full Details:**
- Investigation: [DOMAIN_TRANSFER_INVESTIGATION.md](DOMAIN_TRANSFER_INVESTIGATION.md)
- Draft Issue: [DRAFT_ISSUE_DOMAIN_TRANSFER.md](DRAFT_ISSUE_DOMAIN_TRANSFER.md)
