# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.4.0] - 2026-11-18

### Added
- `openprovider_domain_price` data source: reads what a registration, transfer, renewal or restore of a domain costs, without ordering it
- `prices.Get` client function: prices any operation, where `prices.Create` and `prices.Transfer` price two
- `openprovider_ssl_orders` data source: lists the SSL orders of the account with their status and expiration date, optionally for one `common_name` or some `status` values
- `openprovider_dns_zones` data source: lists every DNS zone of the account
- `openprovider_dns_records` data source: lists every record of a DNS zone
- `openprovider_domain_renewal` resource: renews a domain for 1 to 10 years when it is created or replaced, and only when the domain is active and OpenProvider allows a renewal now; `triggers` renews again
- `openprovider_domains` data source: `can_renew` shows whether OpenProvider allows a renewal of the domain now
- `domains.Renew` client function
- `openprovider_domain_restore` resource: restores a domain that has expired, where the registry and the API allow it, when it is created or replaced; also finds a domain that is deleted since it expired; `triggers` restores again
- `domains.Restore` client function
- `openprovider_dns_zone` resource: creates a DNS zone, optionally from a DNS template, and imports an existing zone by its name; a destroy only removes the zone from the state unless `allow_deletion` is true, as for `openprovider_dns_record`
- `dns.CreateZone` and `dns.DeleteZone` client functions
- `openprovider_dns_record`: import an existing record by `zone_name/name/type`, or `zone_name/name/type/value` where the zone has several records of one name and type; an imported record starts with `allow_deletion` false
- `dns.GetRecordWithValue` client function
- `openprovider_dns_template` resource: defines a set of DNS records once, for a new zone to start with; the API has no call to change a template, so a change replaces it; imports by id
- `dns.CreateTemplate`, `dns.GetTemplate` and `dns.DeleteTemplate` client functions
- `openprovider_ssl_order`: `csr` orders with your own certificate signing request, `certificate`, `intermediate_certificate` and `root_certificate` hold the issued certificate and its chain once the order is active, and an existing order is imported by its id; a change of the CSR replaces the order, except a change of whitespace or the CSR that an import read
- `openprovider_domain`: `is_private_whois_enabled` (WHOIS privacy) with `is_private_whois_allowed`, `comments`, and `nameservers`, a list of nameservers of your own (`name`, `ip`, `ip6`) instead of `ns_group`; the registration, the transfer and an update send them, and a domain that does not set them keeps the values that OpenProvider has. Tags are not included: the API's tags are account-wide key-value pairs with no link to a domain
- `openprovider_domain`: `auth_code_wo` and `auth_code_wo_version`, a write-only auth code for a transfer that Terraform keeps out of the plan and the state; a change of the version replaces the domain and sends the transfer again
- `openprovider_domain_transfer_restart`: sends a transfer that failed to the registry again, without a new order
- `data.openprovider_tld`: reads which contact roles a domain of an extension takes
- `data.openprovider_customer_exists`: reads whether the account holds a customer handle, and keeps the data of the customer out of the state
- `data.openprovider_nsgroup`: `allow_missing` makes a group that does not exist a result instead of an error, and `exists` tells whether the account holds the group, so a configuration can find out if it must adopt the group or create it
- `openprovider_domain`: `additional_data`, a map of strings that the registration and the transfer send as `additional_data`, for example the acceptance flag of the terms of a registry (`dk_acceptance = "1"`); a change after the order gives a warning
- CI: supply-chain checks: `zizmor` (workflow analysis), `govulncheck` (reachable Go vulnerabilities, also daily), dependency review on pull requests, CodeQL (Go and workflows), OpenSSF Scorecard, `gitleaks` secret scanning (also in the pre-commit hook), `go mod verify`, and `harden-runner` in audit mode on the `ci.yml` jobs
- Releases attest signed SLSA build provenance for every archive (`gh attestation verify`); `SECURITY.md` documents how to verify a release
- Fuzz tests (`go test -fuzz`) for the `Retry-After` handling, the domain name split and the IP whitelist validator

### Changed
- `openprovider_domain`: a change of `owner_handle` changes the owner of the domain with an update; `domains.UpdateDomainRequest` has `OwnerHandle`. The provider does not send `accept_update_fee`, so the registrar refuses a change that requires a fee, and the apply fails
- The client calls the `/v1` endpoints of the Openprovider API instead of `/v1beta`, following the new API reference at `developer.openprovider.com`; the mock server and `.stats.yml` use its spec (`/data/swagger.json`)
- Renovate requests a review from the `maintainers` team instead of the single user `dsdeboer`
- CI: `jdx/mise-action` no longer caches, so a cache written by an earlier run cannot influence a release build
- CI: the release job runs in the `release` environment
- `domains.UpdateDomainRequest.DnssecKeys` and `domains.TransferDomainRequest.DnssecKeys` are now `*[]DnssecKey`
- Resources, data sources and the ephemeral resource take the API client from the provider through one helper, so a wrong provider data type gives one error message, "Unexpected Provider Data Type", instead of three wordings
- Client functions send their request through the new `Client.JSON`, which replaces the copied build-request, close-body and decode code of most of them; `client.APIError` is the error for a status outside 2xx, with the same message as before, and `client.IsNotFound` tells a 404 apart
- `internal/provider/resource_domain.go` (1,593 lines) is split by concern into `resource_domain_{schema,create,read,update,delete}.go`, `domain_validators.go`, `domain_dnssec.go` and `domain_lookup.go`; the declarations are moved unchanged
- Lint: `errorlint`, `usestdlibvars` and `gocritic` are on; errors are compared with `errors.Is`, and requests name their method with `http.MethodGet` and the like

### Fixed
- `ssl.ListOrders` pages through the listing; before, it returned only the API's first page of 100 orders
- `dns.ListZones` and `dns.ListRecords` page through the listing; before, they returned only the API's first page of 100, so `openprovider_dns_record` could not find a record beyond it
- `openprovider_dns_record`: a refresh reads the record with the value in the state, so a zone with several records of one name and type (for example two MX records) no longer reads the first of them
- Client: a `Retry-After` header with a huge number of seconds overflowed to a negative pause, so the request was repeated at once instead of after the 30 s cap
- `openprovider_domain`: `dnssec_keys = []` removes the keys of the domain on update and sends an empty list on transfer; before, the empty list was dropped from the request

## [1.3.1] - 2026-11-17

### Fixed
- Release: `SHA256SUMS` lists only the archives and the registry manifest, not the SBOMs. In v1.3.0 it listed the `.cdx.json` files too, and the Terraform Registry rejected the release with `missing files in request body`; the SBOMs are still attached to the release

## [1.3.0] - 2026-11-01

### Added
- CI: the `ci.yml` workflow builds the provider, generates a CycloneDX SBOM from the compiled binary, and scans it with Trivy and Grype; releases attach a CycloneDX SBOM per archive
- `openprovider_domain`: `is_locked` sets the lock of the domain against a transfer to a different registrar, so a configuration can unlock a domain before it is transferred out; if it is not set, the provider keeps the lock that the domain has
- `openprovider_domain`: a transfer sends `dnssec_keys` and `is_dnssec_enabled` with the transfer request; while the transfer is in progress (status `REQ` or `PEN`), the state keeps the configured DNSSEC values, and when it completes a refresh reads them from OpenProvider again; a deleted record of the domain in the listing does not replace the order in state; when the listings show only a deleted record of a transfer in progress, for example after a transfer back of a domain that the account had before, a refresh keeps the state as last read and a destroy fails
- `domains` client package: `ListOptions.IsDeleted` sends the `is_deleted` filter of `GET /v1beta/domains`
- `openprovider_domain`: `on_destroy = "expire"` keeps the domain in the account, as `retain` does, and turns its autorenew off, so a domain that Terraform no longer manages expires at its expiration date at OpenProvider
- `openprovider_domain_auth_code` ephemeral resource: reads the auth code of a domain that the account holds, so a different registrar or reseller account can transfer the domain, without keeping the code in the plan or the state
- `openprovider_domain_auth_code_reset` resource: replaces the auth code of a domain with a new code on create or replace (for example on a change of `triggers`), so the old code no longer transfers the domain
- `domains.GetAuthCode` and `domains.ResetAuthCode` client functions

### Changed
- CI installs Go, golangci-lint, Terraform and Node through `jdx/mise-action` at the versions in `mise.toml` and `mise.lock`, so CI uses the same tools as a local checkout; before, the lint job had no golangci-lint and `./scripts/lint` fell back to `go vet`
- CI checks that `go.mod` and `go.sum` are tidy (`go mod tidy -diff`) and that `./scripts/docs` leaves no new or changed files in `docs/` or `templates/`; the test job no longer runs `go mod tidy`
- CI uses one concurrency group per ref: a newer push cancels a pull request's older run, while runs on `main` and on tags always complete; checkouts do not keep the token, and GoReleaser stays on v2
- `./scripts/test` runs `go test -count=1` instead of first running `go clean -cache -testcache -modcache`, so a test run no longer deletes the local build and module caches and CI keeps its cache; test results are still never taken from the cache

### Removed
- The `snyk` CI job; Snyk scans pull requests and `main` through the Snyk GitHub App (`security/snyk` and `code/snyk` statuses), so `SNYK_TOKEN` is no longer used
- The self-hosted Renovate workflow (`.github/workflows/renovate.yml`); `renovate.json` and `.github/renovate-global.json5` stay in place

### Fixed
- `openprovider_domain_auth_code` and `openprovider_domain_auth_code_reset`: a domain that the account shows only as deleted (status `DEL`) is not held, so a read or a reset fails with "Domain Not Found" instead of asking OpenProvider for the auth code of a domain that left the account
- `openprovider_domain`: when the order response of a registration or a transfer leaves out `autorenew` or `ns_group`, and the listing does not give them, the state keeps the planned values; before, it set `autorenew` to false and `ns_group` to null
- `openprovider_domain`: a domain that the account shows as deleted (status `DEL`), for example after a transfer to a different registrar, no longer fails a destroy; the destroy makes no change at OpenProvider and gives a warning. A destroy also looks up a domain that the default listing does not show in the listing of deleted domains, and with `on_destroy = "expire"` fails when neither listing shows a domain that the registry calls taken. A refresh removes a deleted domain from state
- `openprovider_domain`: a registration that leaves `period` out no longer fails the apply with "unknown value after apply" after the domain is ordered; it now orders one year, the period that `max_cost` was checked against, and keeps that period in state
- `openprovider_domain`: a `period` below 1 on a registration is now rejected at plan time; before, a registration ordered one year but kept the configured value in state
- `openprovider_domain`: a transfer is quoted at the transfer price for one year, not at the registration price for its `period`, so `max_cost` holds against what the transfer costs

## [1.2.0] - 2026-10-14

### Added
- `openprovider_rcp_ip_whitelist` resource: manages the control panel (RCP) IP whitelist of a contact through `PUT /v1beta/contacts/{id}`, with IPv4/IPv6/CIDR validation, drift detection, import by contact id, and `on_destroy` (`retain` by default, or `clear`, which switches the whitelist off)
- `openprovider_contact` data source: a contact by numeric id, with its control panel `rcp_allow` and `rcp_deny` lists
- `openprovider_contact` resource: creates, imports and manages a contact (name, email, phone, address, company, locale, comments, VAT, role, `is_active`, `api_access_enabled`) through `/v1beta/contacts`; the `password` is write-only and never kept in state, attributes left out of the configuration are not managed, and `on_destroy` is `retain` (the default) or `delete`
- `company_name`, `locale`, `comments`, `vat`, `is_active` and `api_access_enabled` on the `openprovider_contact` data source
- `contacts` client package with `Get`, `Create`, `Update`, `Delete` and `SetRCPAllowList`

## [1.1.2] - 2026-09-26

### Changed
- Renovate now runs self-hosted (own `renovate.yml` workflow and image) instead of the hosted GitHub App, and its config moved from `config:recommended` to `config:best-practices`
- Renovate now automerges minor/patch updates across every package manager in the repo (GitHub Actions, Go modules, Terraform, mise) instead of just GitHub Actions and `golang.org/x/`/`google.golang.org/` modules
- GitHub Actions are now pinned to commit digests (via `helpers:pinGitHubActionDigests`, pulled in by `config:best-practices`) instead of semver tags only, so a compromised tag can't silently repoint a workflow step
- Renovate no longer waits for a weekly Monday schedule or shows a dependency dashboard issue; it rebases a PR only when it conflicts instead of whenever it falls behind base
- CI workflows consolidated: `docs.yml` and `release.yml` were folded into `ci.yml`, and runners switched to `ubuntu-26.04-arm`
- Go directive bumped to 1.27.1
- Removed the `.devcontainer` configuration
- Reformatted `LICENSE` to the standard MIT license text (no change in terms)

## [1.1.1] - 2026-09-23

### Fixed
- `openprovider_domain`: `Update` no longer produces "Provider produced inconsistent result after apply ... on_destroy: was cty.StringVal(\"retain\"), but now null" for a domain whose state predates `on_destroy` — the same class of bug as `openprovider_nsgroup`'s `allow_deletion` fix in 1.1.0, where the applied state was built from a `Read` seeded off the prior state rather than the plan

## [1.1.0] - 2026-09-22

### Added
- `max_cost` and `currency` on `openprovider_domain`: a registration or transfer is quoted before it is ordered, and the apply fails without spending where the quote exceeds the bound
- `openprovider_domain_check` data source: whether a domain is available to register, asked of the registry rather than the account
- `openprovider_domains` data source: the domains the account holds, optionally filtered to one `full_name`, an empty list where the account does not hold it
- `full_name` filtering in the domains client, which the domain lookup by name now uses instead of paging through the account
- `on_destroy` on `openprovider_domain`: a destroy retains the domain at OpenProvider (the default) or deletes it, where it used to fail
- `mise.toml` for local tool version management
- `CLAUDE.md` with project-specific development guidelines

### Changed
- Migrated dependency management from Dependabot to Renovate
- Updated Go version to 1.26
- Replaced `mergo` module with `dario.cat/mergo`
- Updated various Go dependencies and GitHub Actions to their latest versions
- Consolidated AI agent documentation into `AGENTS.md` and removed duplication across `CLAUDE.md` and GitHub Copilot instructions
- Improved repository maintenance by removing obsolete agent configurations
- Made `CLAUDE.md` the canonical AI agent instructions doc; `AGENTS.md`, `.github/copilot-instructions.md`, and `.github/agents/coding-agent.md` now reference it instead of duplicating (or symlinking) content

### Fixed
- `openprovider_nsgroup`: `Create` reads the group back to expose its computed attributes instead of leaving them unknown, `Read` handles a group deleted outside Terraform, `Delete` deletes the group at OpenProvider only when `allow_deletion` is set (state-only removal by default, matching `openprovider_dns_record`), and a group deleted outside Terraform is dropped from state instead of surfacing as an error (the same `client.Client.Do` retry/error-reporting change from #110 that broke `openprovider_glue_record`)
- `openprovider_nsgroup`: `Update` no longer produces "Provider produced inconsistent result after apply ... allow_deletion: was cty.False, but now null" — it built the result state from a `Read` seeded off the framework's blank `UpdateResponse.State` instead of from the plan, which silently dropped `allow_deletion`
- `openprovider_glue_record`: a record deleted outside Terraform is now dropped from state instead of erroring on `Read`, and destroying an already-deleted record is a no-op again instead of failing (a 404 from `client.Client.Do` was no longer told apart from any other error after its retry/error-reporting change)
- `openprovider_domain`: the request timeout is now long enough for a registration to complete, a failed request reports the API's reason instead of a bare status, an update no longer drops the order fields (`period`, `max_cost`, `currency`), and a plan with `dnssec_keys` left unstated no longer reports a change on every run
- `openprovider_domain`: `period` now carries its prior state into the plan when left unstated, instead of going unknown; `Update` also no longer writes an unknown `period`, `max_cost` or `currency` into the applied state when the prior state held null for them (a domain whose state predates the field, or was imported before it was ever set) -- both cases used to fail with "Provider returned invalid result object after apply"
- `openprovider_domain` no longer leaves the state when the account's listing lags behind a registration: an absent listing is confirmed against the availability check, and a listing whose envelope reports a non-zero `code` is an error rather than an empty account
- The import of an `openprovider_domain` warns about the transfer authorization code only where the domain's status says a transfer is not complete, and no longer on a domain the account already holds
- Resolved `go get -u all` failure by fixing `mergo` module path conflict
- Resolved `openpgp: key expired` error in documentation workflow by explicitly setting up Terraform

## [1.0.1] - 2026-02-22

### Fixed
- Minor bug fixes and improvements for DNSSEC key mapping and conversion

## [1.0.0] - 2026-02-15

### Added
- DNS record management resource (openprovider_dns_record)
  - Full CRUD operations for DNS records (A, AAAA, CNAME, MX, TXT, NS, SRV, etc.)
  - Support for TTL and priority fields
  - DNS zone data source (openprovider_dns_zone)
- SSL/TLS certificate management
  - SSL order resource (openprovider_ssl_order)
  - Full CRUD operations for SSL orders
  - Renewal and reissue workflows
  - Autorenew configuration
  - Additional domains (SANs) support
  - SSL product data source (openprovider_ssl_product)
- Comprehensive client library for DNS and SSL operations
- Unit tests for all new DNS and SSL functionality
- API documentation with usage examples for DNS and SSL
- Health check documentation and improvements
- Contributor Covenant code of conduct
- .editorconfig for editor defaults
- Makefile shortcuts for common scripts
- Dependabot updates for Go modules

### Removed
- Deprecated transfer-only domain attributes (import_contacts_from_registry, import_nameservers_from_registry, is_private_whois_enabled)

## [0.1.0] - Initial Release

### Added
- Customer management resources and data sources
- Domain management resources and data sources
- Nameserver group management
- OpenProvider API client library
- Comprehensive testing infrastructure with Prism mock server
- CI/CD pipeline with GitHub Actions
- Documentation generation support

### Features
- Create and manage customer handles (contact information)
- Register and manage domains with customizable contact handles
- Configure and manage nameserver groups
- Automatic token refresh and authentication handling
- Support for Terraform >= 1.3

[Unreleased]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.4.0...HEAD
[1.4.0]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.3.1...v1.4.0
[1.3.1]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.3.0...v1.3.1
[1.3.0]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.1.2...v1.2.0
[1.1.2]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.1.1...v1.1.2
[1.1.1]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.0.1...v1.1.0
[1.0.1]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/charpand/terraform-provider-openprovider/releases/tag/v1.0.0
[0.1.0]: https://github.com/charpand/terraform-provider-openprovider/releases/tag/v0.1.0
