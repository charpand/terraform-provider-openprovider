data "openprovider_tld" "cfd" {
  name = "cfd"
}

output "cfd_takes_an_admin_handle" {
  value = data.openprovider_tld.cfd.admin_handle_supported
}
