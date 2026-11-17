# A record is imported by zone_name/name/type, with the name as OpenProvider lists it.
terraform import openprovider_dns_record.www example.com/www/A

# Where the zone has several records of one name and type, add the value.
terraform import openprovider_dns_record.mx example.com/example.com/MX/mail1.example.com
