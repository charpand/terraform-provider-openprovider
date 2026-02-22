# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.0.1...HEAD
[1.0.1]: https://github.com/charpand/terraform-provider-openprovider/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/charpand/terraform-provider-openprovider/releases/tag/v1.0.0
[0.1.0]: https://github.com/charpand/terraform-provider-openprovider/releases/tag/v0.1.0
