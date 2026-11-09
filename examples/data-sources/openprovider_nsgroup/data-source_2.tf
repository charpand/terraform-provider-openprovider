# Read a group that may not exist. A missing group is a result, not an error.
data "openprovider_nsgroup" "shared" {
  name          = "my-ns-group"
  allow_missing = true
}

output "shared_group_exists" {
  value = data.openprovider_nsgroup.shared.exists
}
