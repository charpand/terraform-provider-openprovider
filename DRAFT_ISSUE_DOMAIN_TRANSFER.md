# [DRAFT ISSUE] Implement Domain Transfer Resource

## Summary
Implement a new `openprovider_domain_transfer` resource to enable declarative domain transfers through Terraform using the Openprovider API.

## Background
Users need the ability to transfer domains to Openprovider using infrastructure-as-code workflows. This feature would allow seamless domain migrations as part of Terraform-managed infrastructure.

**Investigation Document:** See [DOMAIN_TRANSFER_INVESTIGATION.md](DOMAIN_TRANSFER_INVESTIGATION.md) for complete analysis and design decisions.

## Scope

### In Scope
- Create new `openprovider_domain_transfer` resource
- Support transfer initiation with auth code
- Allow configuration of contact handles
- Support nameserver configuration (ns_group or nameservers)
- Implement autorenew configuration
- Track transfer status in state
- Enable import of existing transferred domains
- Support WHOIS privacy configuration

### Out of Scope
- Transfer approval for outbound transfers (when Openprovider is losing registrar)
- Auth code retrieval data source
- Bulk transfer operations
- Advanced TLD-specific fields (can be added later)
- Active polling for transfer completion

## Requirements

### API Client Implementation
**Location:** `internal/client/domains/`

Create the following new files:
- `transfer.go` - Transfer API client function
- `transfer_test.go` - Unit tests using mock server

**Functions to implement:**
```go
// Transfer initiates a domain transfer
func Transfer(c *client.Client, req *TransferDomainRequest) (*Domain, error)

type TransferDomainRequest struct {
    Domain struct {
        Name      string `json:"name"`
        Extension string `json:"extension"`
    } `json:"domain"`
    AuthCode                       string       `json:"auth_code"`
    OwnerHandle                    string       `json:"owner_handle"`
    AdminHandle                    string       `json:"admin_handle,omitempty"`
    TechHandle                     string       `json:"tech_handle,omitempty"`
    BillingHandle                  string       `json:"billing_handle,omitempty"`
    Autorenew                      string       `json:"autorenew,omitempty"`
    NSGroup                        string       `json:"ns_group,omitempty"`
    Nameservers                    []Nameserver `json:"name_servers,omitempty"`
    ImportContactsFromRegistry     bool         `json:"import_contacts_from_registry,omitempty"`
    ImportNameserversFromRegistry  bool         `json:"import_nameservers_from_registry,omitempty"`
    IsPrivateWhoisEnabled          bool         `json:"is_private_whois_enabled,omitempty"`
}

type TransferDomainResponse struct {
    Code int `json:"code"`
    Data Domain `json:"data"`
}
```

**Endpoint:** `POST /v1beta/domains/transfer`

### Provider Resource Implementation
**Location:** `internal/provider/`

Create the following new files:
- `resource_domain_transfer.go` - Main resource implementation
- `models_domain_transfer.go` - Terraform models/schema

**Resource Schema:**
```hcl
resource "openprovider_domain_transfer" "example" {
  # Required
  domain       = string  # Full domain name (e.g., "example.com")
  auth_code    = string  # EPP/Authorization code (sensitive)
  owner_handle = string  # Owner contact handle

  # Optional Contact Handles
  admin_handle   = string
  tech_handle    = string
  billing_handle = string

  # Optional Settings
  autorenew = bool  # Default: false
  
  # Nameserver Configuration (mutually exclusive)
  ns_group = string
  # OR nameserver blocks (if ns_group not specified)
  
  # Import Options
  import_contacts_from_registry    = bool  # Default: false
  import_nameservers_from_registry = bool  # Default: false
  
  # Privacy
  is_private_whois_enabled = bool  # Default: false

  # Computed (read-only)
  id              = string  # Domain ID in Openprovider
  status          = string  # Transfer status
  expiration_date = string  # Domain expiration after transfer
}
```

**Resource Methods:**
- `Create()` - Initiate transfer via API
- `Read()` - Retrieve domain status via existing `domains.Get()` or `domains.List()`
- `Update()` - Update domain settings post-transfer (limited fields)
- `Delete()` - Remove from state only (document that domain is not deleted)
- `ImportState()` - Import by domain name, look up via `domains.List()`

**Key Implementation Details:**
1. Mark `auth_code` as sensitive in schema
2. Parse domain name into name + extension for API
3. Convert autorenew bool to "on"/"off" string for API
4. Reuse existing `getDomainByName()` helper for Read/Import
5. Validate mutual exclusivity of `ns_group` and nameserver blocks
6. Clear error messages for common failures (invalid auth code, locked domain, etc.)

### Testing

**Unit Tests:** `internal/client/domains/transfer_test.go`
- Test successful transfer request
- Test error handling
- Use Prism mock server (default: http://localhost:4010)

**Integration Tests:** `internal/provider/domain_transfer_test.go`
- Test resource creation with required fields only
- Test resource creation with all optional fields
- Test resource read/refresh
- Test resource import
- Test delete (removes from state)
- Use Prism mock server with `testutils.MockTransport`

**Test Coverage:**
- All success paths
- Common error scenarios
- Edge cases (domain already exists, invalid auth code, etc.)

### Documentation

**Template:** `templates/docs/resources/domain_transfer.md.tmpl`
```markdown
---
page_title: "openprovider_domain_transfer Resource - terraform-provider-openprovider"
subcategory: ""
description: |-
  Manages domain transfers to Openprovider.
---

# openprovider_domain_transfer (Resource)

Transfers a domain to Openprovider from another registrar.

## Example Usage

{{tffile "examples/resources/openprovider_domain_transfer/basic.tf"}}

## Schema

### Required
- `domain` (String) - Full domain name including extension (e.g., "example.com")
- `auth_code` (String, Sensitive) - Authorization/EPP code from current registrar
- `owner_handle` (String) - Owner contact handle

### Optional
- `admin_handle` (String) - Admin contact handle
- `tech_handle` (String) - Tech contact handle
- `billing_handle` (String) - Billing contact handle
- `autorenew` (Boolean) - Enable automatic renewal (default: false)
- `ns_group` (String) - Nameserver group to use
- `import_contacts_from_registry` (Boolean) - Import existing contact info (default: false)
- `import_nameservers_from_registry` (Boolean) - Import existing nameservers (default: false)
- `is_private_whois_enabled` (Boolean) - Enable WHOIS privacy (default: false)

### Read-Only
- `id` (String) - Domain ID in Openprovider system
- `status` (String) - Current transfer status
- `expiration_date` (String) - Domain expiration date after transfer

## Import

Import an existing transferred domain by its domain name:

```shell
terraform import openprovider_domain_transfer.example example.com
```

## Notes

### Transfer Process
Domain transfers typically take 5-7 days to complete. The resource creation succeeds when the transfer is initiated, not when it completes. Check the `status` attribute to monitor progress.

### Auth Code Security
The auth code is marked as sensitive and should be provided via variables, not hardcoded. It will appear in the Terraform state file, so use an encrypted backend for production.

### Status Values
- `REQ` - Transfer requested, pending approval
- `ACT` - Transfer complete, domain active
- Other status codes may appear based on registry

### Deletion Behavior
Destroying this resource removes it from Terraform state but does NOT delete or transfer the domain away. The domain remains registered at Openprovider.
```

**Examples:** Create `examples/resources/openprovider_domain_transfer/`
1. `basic.tf` - Minimal transfer with required fields
2. `complete.tf` - Full example with all options
3. `with_nsgroup.tf` - Transfer with nameserver group
4. `import.sh` - Import command example

### Update API.md
Add section for domain transfers:
```markdown
## Domain Transfers

### Transfer Domain

\`\`\`go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.TransferDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.AuthCode = "EPP-AUTH-CODE-12345"
req.OwnerHandle = "owner123"
req.NSGroup = "my-ns-group"
req.Autorenew = "on"

domain, err := domains.Transfer(c, req)
\`\`\`
```

## Implementation Plan

### Phase 1: API Client (MVP)
- [ ] Implement `internal/client/domains/transfer.go`
- [ ] Add request/response structs
- [ ] Implement `Transfer()` function
- [ ] Write unit tests in `transfer_test.go`
- [ ] Verify tests pass with mock server

**Acceptance Criteria:**
- `domains.Transfer()` successfully calls API
- All tests pass with Prism mock server
- Error handling works correctly

### Phase 2: Provider Resource (Core)
- [ ] Implement `internal/provider/resource_domain_transfer.go`
- [ ] Create `models_domain_transfer.go` with schema
- [ ] Implement `Create()` method
- [ ] Implement `Read()` method
- [ ] Implement `Delete()` method
- [ ] Mark auth_code as sensitive
- [ ] Add basic integration tests

**Acceptance Criteria:**
- Resource can be created in Terraform
- Resource state tracks transfer
- Delete removes from state only
- Basic tests pass

### Phase 3: Full Features
- [ ] Implement `Update()` method for post-transfer changes
- [ ] Implement `ImportState()` method
- [ ] Add validation (mutual exclusivity, required fields)
- [ ] Add all optional parameters
- [ ] Comprehensive error handling
- [ ] Complete integration test suite

**Acceptance Criteria:**
- All optional parameters work
- Import functionality works
- Update handles post-transfer changes
- All edge cases covered
- Full test coverage

### Phase 4: Documentation
- [ ] Create documentation template
- [ ] Write example configurations
- [ ] Update API.md
- [ ] Update README.md if needed
- [ ] Generate docs with `./scripts/docs`
- [ ] Review generated documentation

**Acceptance Criteria:**
- Clear, comprehensive documentation
- All examples work
- Import instructions clear
- Security notes prominent

### Phase 5: Testing & QA
- [ ] Run full test suite: `./scripts/test`
- [ ] Run linter: `./scripts/lint`
- [ ] Format code: `./scripts/format`
- [ ] Manual testing with mock server
- [ ] Test all examples
- [ ] Security review (sensitive fields)

**Acceptance Criteria:**
- All tests pass
- Linter passes with no warnings
- Code properly formatted
- Examples verified
- No security issues

## Code Style Guidelines

Follow existing patterns in the codebase:
- Use `gofmt` for formatting
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- All API functions take `*client.Client` as first parameter
- Close response bodies in defer statements
- Create response structs matching Openprovider API
- Use `testutils.MockTransport` for tests
- Run Prism mock server for integration tests

## Security Considerations

1. **Sensitive Data:**
   - Mark `auth_code` as sensitive in schema
   - Document state file security requirements
   - Recommend variable usage for auth codes

2. **State Management:**
   - Auth codes stored in state file
   - Recommend encrypted backend
   - Document best practices

3. **Error Messages:**
   - Don't expose sensitive data in errors
   - Sanitize API error messages if needed

## Testing Strategy

### Unit Testing
- Test API client with mock responses
- Verify request serialization
- Test error conditions
- Mock server: http://localhost:4010 (Prism)

### Integration Testing  
- Test full resource lifecycle
- Verify state management
- Test import functionality
- Test update scenarios
- Use `testutils.MockTransport`

### Manual Testing
- Test with real API (optional, documented)
- Verify examples work
- Test error scenarios
- Validate documentation accuracy

## Documentation Checklist

- [ ] Resource documentation template created
- [ ] Basic example provided
- [ ] Complete example provided
- [ ] Import instructions clear
- [ ] Schema fully documented
- [ ] Status values explained
- [ ] Security notes included
- [ ] Transfer timeline documented
- [ ] Error scenarios explained
- [ ] API.md updated
- [ ] README.md updated (if needed)

## Success Metrics

1. **Functionality:**
   - ✅ Transfer can be initiated via Terraform
   - ✅ Required parameters work correctly
   - ✅ Optional parameters work correctly
   - ✅ Status is tracked in state
   - ✅ Import works correctly

2. **Code Quality:**
   - ✅ All tests pass
   - ✅ Linter passes with no warnings
   - ✅ Code follows style guidelines
   - ✅ Error handling is comprehensive

3. **Documentation:**
   - ✅ Documentation is clear and complete
   - ✅ Examples are working and helpful
   - ✅ Security considerations documented
   - ✅ Transfer process explained

4. **User Experience:**
   - ✅ Clear error messages
   - ✅ Intuitive resource schema
   - ✅ Predictable behavior
   - ✅ Well-documented edge cases

## Questions & Considerations

### Open Questions
1. Should we support `nameserver` blocks or only `ns_group`?
   - **Decision:** Support both, with validation for mutual exclusivity
   
2. Should auth_code remain in state after transfer?
   - **Decision:** Yes, for state consistency, but mark as sensitive

3. Should we actively poll transfer status?
   - **Decision:** No, user runs `terraform refresh` to update

4. What happens if user tries to transfer a domain already at Openprovider?
   - **Decision:** API returns error, surface clear message to user

5. Should Update() support changing contact handles post-transfer?
   - **Decision:** Yes, limited to fields updatable via domain update API

### Design Decisions Made
- ✅ Separate resource (not extension of openprovider_domain)
- ✅ Delete removes from state only (does not delete domain)
- ✅ Create initiates transfer, does not wait for completion
- ✅ Import uses domain name as identifier
- ✅ Reuse existing domain list/get APIs for Read operation

## Dependencies

- No new external dependencies required
- Uses existing Terraform Plugin Framework
- Uses existing client infrastructure
- Uses existing test utilities

## Timeline Estimate

- Phase 1 (API Client): 2-3 hours
- Phase 2 (Core Resource): 4-6 hours
- Phase 3 (Full Features): 3-4 hours
- Phase 4 (Documentation): 2-3 hours
- Phase 5 (Testing & QA): 2-3 hours

**Total: ~15-20 hours** for complete implementation

## References

- Investigation Document: [DOMAIN_TRANSFER_INVESTIGATION.md](DOMAIN_TRANSFER_INVESTIGATION.md)
- Openprovider API: https://docs.openprovider.com/swagger.json
- Transfer Endpoint: `POST /v1beta/domains/transfer`
- Existing Domain Resource: `internal/provider/resource_domain.go`
- Terraform Plugin Framework: https://developer.hashicorp.com/terraform/plugin/framework

## Additional Notes

### Future Enhancements (Post-MVP)
- Transfer status data source
- Auth code retrieval data source (for outbound transfers)
- Transfer approval resource (when Openprovider is losing registrar)
- Bulk transfer support
- Advanced TLD-specific fields via `additional_data`
- Active status polling option

### Migration Path
If user has domains manually transferred (outside Terraform):
1. Create `openprovider_domain_transfer` resource configuration
2. Run `terraform import openprovider_domain_transfer.example example.com`
3. Run `terraform plan` to verify state matches config
4. Continue managing via Terraform

### Relationship to Existing Resources
- **openprovider_domain**: For new domain registration
- **openprovider_domain_transfer**: For incoming transfers from other registrars
- Both resources manage domains, but through different lifecycle paths
- After transfer, domain appears in domain list and can be managed
- No migration needed between resources (they serve different purposes)
