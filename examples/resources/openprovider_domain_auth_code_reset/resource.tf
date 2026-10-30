# Replace the auth code of the domain with a new code, so the old code no
# longer transfers the domain. A change of `triggers` resets the code again.
resource "openprovider_domain_auth_code_reset" "example" {
  domain = "example.com"

  triggers = {
    rotation = "2026-10-30"
  }
}
