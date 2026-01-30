# Domain Transfer Implementation Investigation

## Executive Summary

This document outlines the investigation findings for implementing domain transfer functionality in the Terraform Openprovider provider. Based on analysis of the Openprovider API and existing provider implementation, domain transfers can be implemented through a new `openprovider_domain_transfer` resource.

## Current State

### Existing Resources
The provider currently supports:
- `openprovider_domain` - Domain registration and management
- `openprovider_customer` - Customer/contact handle management
- `openprovider_nsgroup` - Nameserver group management

### Domain Resource Capabilities
The existing `openprovider_domain` resource:
- Registers new domains via `POST /v1beta/domains`
- Updates domain settings (contacts, nameservers, autorenew)
- Supports import of existing domains
- Uses domain name as the primary identifier

## Openprovider API Analysis

### Transfer Endpoints

#### 1. Transfer Domain (POST /v1beta/domains/transfer)
Primary endpoint for initiating domain transfers.

**Key Request Parameters:**
- `domain` (required): Object with `name` and `extension`
- `auth_code` (required): Authorization/EPP code from current registrar
- `owner_handle` (required): Owner contact handle
- `admin_handle`, `tech_handle`, `billing_handle` (optional): Contact handles
- `ns_group` or `name_servers` (optional): Nameserver configuration
- `autorenew` (optional): "on", "off", or "default"
- `period` (optional): Transfer period (if applicable)
- `import_contacts_from_registry` (optional): Import existing contacts
- `import_nameservers_from_registry` (optional): Import existing nameservers
- `is_private_whois_enabled` (optional): Enable WHOIS privacy
- `additional_data` (optional): TLD-specific requirements

**Response Data:**
- `id`: Domain ID in Openprovider system
- `status`: Transfer status (e.g., "REQ" for requested)
- `expiration_date`: Domain expiration after transfer
- `auth_code`: Authorization code (echoed back)

#### 2. Get Auth Code (GET /v1beta/domains/{id}/authcode)
For domains already managed in Openprovider, retrieves the auth code needed to transfer away.

**Use case:** When transferring domains OUT of Openprovider (not relevant for incoming transfers).

#### 3. Approve Transfer (POST /v1beta/domains/{id}/transfer/approve)
For approving incoming transfer requests when the domain is currently at Openprovider.

**Use case:** When Openprovider is the losing registrar and needs to approve the transfer.

## Implementation Approach

### Option 1: Separate Transfer Resource (RECOMMENDED)

Create a new `openprovider_domain_transfer` resource specifically for domain transfers.

**Advantages:**
- Clear separation of concerns (registration vs. transfer)
- Explicit intent in Terraform configuration
- Easier to manage transfer-specific parameters
- Better error handling for transfer-specific issues
- Follows Terraform best practices for distinct operations

**Schema:**
```hcl
resource "openprovider_domain_transfer" "example" {
  domain              = "example.com"
  auth_code           = "EPP-AUTH-CODE-12345"
  owner_handle        = openprovider_customer.owner.handle
  admin_handle        = openprovider_customer.admin.handle   # optional
  tech_handle         = openprovider_customer.tech.handle    # optional
  billing_handle      = openprovider_customer.billing.handle # optional
  
  autorenew           = true  # optional, default: false
  
  # Nameserver options (mutually exclusive)
  ns_group = "my-ns-group"  # preferred
  # OR
  # nameservers {
  #   name = "ns1.example.com"
  # }
  
  # Import options (optional)
  import_contacts_from_registry    = false  # optional
  import_nameservers_from_registry = false  # optional
  
  # Privacy options
  is_private_whois_enabled = false  # optional
  
  # Additional data for specific TLDs (optional)
  additional_data = {
    # TLD-specific fields as needed
  }
}
```

**Computed Attributes:**
- `id` - Domain ID assigned by Openprovider
- `status` - Current transfer status
- `expiration_date` - Domain expiration after transfer

**Lifecycle:**
- `create`: Initiates the transfer via POST /v1beta/domains/transfer
- `read`: Retrieves domain details via GET /v1beta/domains/{id}
- `update`: Updates domain settings after transfer completes (limited to mutable fields)
- `delete`: Does NOT delete the domain, removes from state only (transfer is permanent)
- `import`: Import an existing transferred domain by domain name

### Option 2: Extend Existing Domain Resource (NOT RECOMMENDED)

Add a `transfer_auth_code` parameter to the existing `openprovider_domain` resource.

**Disadvantages:**
- Mixing registration and transfer logic complicates the resource
- Unclear semantics (is this creating or transferring?)
- Risk of breaking existing domain registrations
- Harder to handle transfer-specific parameters
- Terraform plan would be ambiguous

## Transfer Workflow

### User Workflow
1. **Preparation Phase:**
   - User obtains auth code from current registrar
   - User creates/identifies customer handles in Openprovider
   - User optionally creates nameserver groups

2. **Transfer Initiation:**
   ```hcl
   resource "openprovider_domain_transfer" "example" {
     domain       = "example.com"
     auth_code    = var.auth_code
     owner_handle = openprovider_customer.owner.handle
     ns_group     = openprovider_nsgroup.dns.name
   }
   ```

3. **Monitoring:**
   - Terraform tracks the transfer status
   - User can check `status` attribute
   - Transfer typically takes 5-7 days for most TLDs

4. **Post-Transfer:**
   - Domain is managed like any other domain
   - Can be updated using standard update operations
   - Original `openprovider_domain_transfer` resource remains in state

### API Workflow
1. Provider calls `POST /v1beta/domains/transfer` with:
   - Domain name and extension
   - Auth code
   - Contact handles
   - Nameserver configuration

2. Openprovider initiates transfer with registry:
   - Validates auth code
   - Contacts current registrar
   - Waits for approval/auto-approval

3. Transfer completes (asynchronously):
   - Domain appears in domain list
   - Status updates from "REQ" to "ACT"
   - Domain becomes fully manageable

## Technical Requirements

### Client API Implementation
Create new functions in `internal/client/domains/`:

```go
// transfer.go
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
    // Additional fields as needed
}
```

### Provider Resource Implementation
Create `internal/provider/resource_domain_transfer.go`:
- Implement `DomainTransferResource` with Terraform Framework
- Handle transfer initiation in `Create()`
- Map transfer response to Terraform state
- Implement `Read()` to check domain status
- Implement `Update()` for post-transfer modifications
- Implement `Delete()` to remove from state (with clear documentation that domain is not deleted)
- Implement `ImportState()` for existing transferred domains

### Testing Strategy
1. **Unit Tests:** Test API client functions with mock responses
2. **Integration Tests:** Use Prism mock server for provider resource tests
3. **Manual Testing:** Document manual test procedure with real API

### Documentation Requirements
1. **Resource Documentation** (`templates/docs/resources/domain_transfer.md.tmpl`):
   - Clear explanation of transfer process
   - Required vs. optional parameters
   - Auth code acquisition instructions
   - Status values and meanings
   - Import instructions

2. **Examples** (`examples/resources/openprovider_domain_transfer/`):
   - Basic transfer
   - Transfer with nameserver group
   - Transfer with contact handles
   - Import example

3. **Guide** (add to README or create separate guide):
   - Step-by-step transfer process
   - Common pitfalls and solutions
   - Timeline expectations
   - Troubleshooting

## Configuration Examples

### Basic Transfer
```hcl
resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.domain_auth_code
  owner_handle = openprovider_customer.owner.handle
}
```

### Complete Transfer with All Options
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

resource "openprovider_nsgroup" "dns" {
  name = "my-dns"
  
  nameserver {
    name = "ns1.example.com"
  }
  
  nameserver {
    name = "ns2.example.com"
  }
}

resource "openprovider_domain_transfer" "example" {
  domain       = "example.com"
  auth_code    = var.domain_auth_code
  owner_handle = openprovider_customer.owner.handle
  ns_group     = openprovider_nsgroup.dns.name
  autorenew    = true
  
  is_private_whois_enabled = true
}

output "transfer_status" {
  value = openprovider_domain_transfer.example.status
}

output "domain_id" {
  value = openprovider_domain_transfer.example.id
}
```

### Import Existing Transfer
```bash
terraform import openprovider_domain_transfer.example example.com
```

## Scope Definition

### In Scope
1. **Domain Transfer Initiation:**
   - Support for standard gTLDs and ccTLDs
   - Auth code validation
   - Contact handle assignment
   - Nameserver configuration
   - Basic TLD-specific requirements

2. **Transfer Monitoring:**
   - Status tracking via Terraform state
   - Domain details after transfer

3. **Post-Transfer Management:**
   - Update contact handles
   - Update nameservers
   - Configure autorenew

4. **Import Support:**
   - Import existing transferred domains

### Out of Scope (for initial implementation)
1. **Transfer Approval (Losing Registrar):**
   - Approving outbound transfers when Openprovider is current registrar
   - Can be added later as separate resource/data source

2. **Auth Code Retrieval:**
   - Getting auth codes for domains at Openprovider
   - Can be added as data source if needed

3. **Bulk Transfers:**
   - Multi-domain transfer operations
   - Use multiple resource blocks instead

4. **Transfer Cancellation:**
   - No explicit cancel operation
   - User can destroy resource from state

5. **Advanced TLD-Specific Fields:**
   - Initial implementation focuses on common parameters
   - Additional TLD fields can be added incrementally

6. **Transfer Status Polling:**
   - No active polling for completion
   - User runs `terraform refresh` to update status

## Technical Considerations

### State Management
- **Transfer as Create Operation:** Initiating a transfer is the resource creation
- **Idempotency:** Cannot re-transfer an already transferred domain; provider should detect and handle gracefully
- **Status Tracking:** Domain status changes asynchronously; resource should reflect current state

### Error Handling
- **Invalid Auth Code:** Clear error message directing user to verify code
- **Domain Locked:** Inform user domain must be unlocked at current registrar
- **Missing Requirements:** Validate required contact handles before API call
- **Registry Rejection:** Surface registry-specific error messages

### Security
- **Auth Code Sensitivity:** Document that auth codes should be in variables, not hardcoded
- **State File Security:** Warn users that auth codes appear in state file
- **Backend Encryption:** Recommend encrypted backend for production

### Async Nature
- Transfers are NOT immediate (5-7 days typical)
- Resource creation succeeds when transfer is initiated, not completed
- Status reflects current state: "REQ" (requested) → "ACT" (active)
- No built-in waiting mechanism; user can check status attribute

## Implementation Phases

### Phase 1: Core Transfer Resource (MVP)
- [ ] Create transfer API client functions
- [ ] Implement basic `openprovider_domain_transfer` resource
- [ ] Support required fields only (domain, auth_code, owner_handle)
- [ ] Basic unit and integration tests
- [ ] Minimal documentation

**Deliverable:** Working transfer resource for common use cases

### Phase 2: Full Feature Support
- [ ] Add optional parameters (admin/tech/billing handles)
- [ ] Implement nameserver configuration (ns_group, nameservers)
- [ ] Add import functionality
- [ ] Comprehensive tests with Prism mock server
- [ ] Complete documentation and examples

**Deliverable:** Production-ready transfer resource

### Phase 3: Advanced Features (Future)
- [ ] TLD-specific additional_data fields
- [ ] Transfer status data source
- [ ] Auth code data source (for outbound transfers)
- [ ] Transfer approval resource (for outbound transfers)

**Deliverable:** Full transfer lifecycle management

## Risks and Mitigations

### Risk: Transfer Failures
**Mitigation:** Comprehensive error handling and clear error messages. Document common failure scenarios.

### Risk: State Drift During Long Transfer
**Mitigation:** Implement robust Read() function to sync state with API. Document that user should run `terraform refresh`.

### Risk: Auth Code Exposure
**Mitigation:** Document security best practices. Consider marking auth_code as sensitive in schema.

### Risk: Duplicate Transfers
**Mitigation:** Check if domain already exists before attempting transfer. Return clear error if transfer in progress.

### Risk: Breaking Existing Domain Resource
**Mitigation:** Create separate resource; no changes to existing `openprovider_domain` resource.

## Success Criteria

1. ✅ User can initiate domain transfer with auth code
2. ✅ User can specify contact handles for transferred domain
3. ✅ User can configure nameservers during transfer
4. ✅ Transfer status is tracked in Terraform state
5. ✅ Clear documentation guides users through process
6. ✅ All tests pass with Prism mock server
7. ✅ No breaking changes to existing resources

## Next Steps

1. **Review and Approval:**
   - Review this investigation document with stakeholders
   - Confirm approach (separate resource vs. extension)
   - Prioritize features for MVP

2. **Create Implementation Issue:**
   - Break down into specific tasks
   - Estimate effort for each phase
   - Define acceptance criteria

3. **Implementation:**
   - Start with Phase 1 (MVP)
   - Iterate based on feedback
   - Add Phase 2 features

4. **Testing and Documentation:**
   - Write comprehensive tests
   - Create user-facing documentation
   - Add examples

## References

- [Openprovider API Documentation](https://docs.openprovider.com/swagger.json)
- [Transfer Domain Endpoint](https://api.openprovider.eu/v1beta/domains/transfer)
- [Existing Domain Resource](internal/provider/resource_domain.go)
- [Terraform Plugin Framework](https://developer.hashicorp.com/terraform/plugin/framework)
- [ICANN Transfer Policy](https://www.icann.org/resources/pages/transfer-policy-2016-06-01-en)

## Appendix A: API Request/Response Examples

### Transfer Request
```json
{
  "domain": {
    "name": "example",
    "extension": "com"
  },
  "auth_code": "EPP-AUTH-CODE-12345",
  "owner_handle": "XX123456-XX",
  "admin_handle": "XX123456-XX",
  "tech_handle": "XX123456-XX",
  "billing_handle": "XX123456-XX",
  "autorenew": "on",
  "ns_group": "my-ns-group"
}
```

### Transfer Response
```json
{
  "code": 0,
  "data": {
    "id": 123456789,
    "status": "REQ",
    "auth_code": "EPP-AUTH-CODE-12345",
    "expiration_date": "2025-03-31 23:59:59"
  }
}
```

## Appendix B: Alternative Approaches Considered

### Approach: Magic Parameter on Domain Resource
Add `transfer_auth_code` to `openprovider_domain`. If present, use transfer API instead of create API.

**Rejected because:**
- Confusing semantics (create vs. transfer)
- Hard to understand from Terraform plan
- Risk of accidental transfers
- Complicates existing resource logic

### Approach: Separate Provider for Transfers
Create `terraform-provider-openprovider-transfer` as separate provider.

**Rejected because:**
- Unnecessary complexity
- Requires users to configure multiple providers
- Doesn't follow Terraform conventions
- Split functionality across providers

### Approach: Data Source for Transfer Status
Use data source to initiate transfers (anti-pattern).

**Rejected because:**
- Data sources should be read-only
- Violates Terraform conventions
- No state management
- Cannot track transfer lifecycle
