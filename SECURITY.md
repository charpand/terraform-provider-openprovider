# Security Policy

## Reporting a Vulnerability

We take the security of our software seriously. If you find a security vulnerability, please report it to us responsibly.

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them privately, in either of two ways:

- on GitHub, through [Report a vulnerability](https://github.com/charpand/terraform-provider-openprovider/security/advisories/new) (a private security advisory that only the maintainers can see), or
- by emailing us at [security@charpand.nl](mailto:security@charpand.nl).

We will acknowledge your report within 48 hours and provide a timeline for a fix, if applicable. We request that you do not disclose the vulnerability publicly until we have had a chance to address it.

## Verifying a release

Each release publishes, next to the provider archives, a `SHA256SUMS` file, a detached GPG signature of it (`SHA256SUMS.sig`), and a CycloneDX SBOM per archive. Releases made after build provenance was introduced (see the changelog) also carry signed SLSA build provenance for every archive.

```sh
# 1. Checksums: verify the GPG signature of SHA256SUMS, then the archive against it
gpg --verify terraform-provider-openprovider_<version>_SHA256SUMS.sig terraform-provider-openprovider_<version>_SHA256SUMS
sha256sum --check --ignore-missing terraform-provider-openprovider_<version>_SHA256SUMS

# 2. Provenance: the archive was built by this repository's release workflow
gh attestation verify terraform-provider-openprovider_<version>_<os>_<arch>.zip \
  --repo charpand/terraform-provider-openprovider
```

Terraform itself verifies the GPG signature when it installs the provider from the registry.
