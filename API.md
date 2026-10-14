# API Documentation

## Initialization

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client"

c := client.NewClient(client.Config{
	BaseURL: "https://api.openprovider.eu",
})
```

## Customers

### List Customers

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/customers"

customerList, err := customers.List(c)
```

### Get Customer

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/customers"

customer, err := customers.Get(c, "XX123456-XX")
```

### Create Customer

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/customers"

req := &customers.CreateCustomerRequest{
	Email: "test@example.com",
	Phone: customers.Phone{
		CountryCode: "1",
		AreaCode:    "555",
		Number:      "1234567",
	},
	Address: customers.Address{
		Street:  "Main St",
		Number:  "123",
		City:    "New York",
		Country: "US",
		Zipcode: "10001",
	},
	Name: customers.Name{
		FirstName: "John",
		LastName:  "Doe",
	},
}

handle, err := customers.Create(c, req)
// handle will be something like "XX123456-XX"
```

### Update Customer

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/customers"

req := &customers.UpdateCustomerRequest{
	Email: "updated@example.com",
}

err := customers.Update(c, "XX123456-XX", req)
```

### Delete Customer

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/customers"

err := customers.Delete(c, "XX123456-XX")
```

## Nameserver Groups

### List NS Groups

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"

groups, err := nsgroups.List(c)
```

### Get NS Group

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"

group, err := nsgroups.Get(c, "my-ns-group")
```

### Get NS Group by Name

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"

group, err := nsgroups.GetByName(c, "my-ns-group")
```

### Create NS Group

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"

req := &nsgroups.CreateNSGroupRequest{
	Name: "cloudflare-ns",
	Nameservers: []nsgroups.Nameserver{
		{Name: "ns1.cloudflare.com"},
		{Name: "ns2.cloudflare.com"},
	},
}

group, err := nsgroups.Create(c, req)
```

### Update NS Group

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"

req := &nsgroups.UpdateNSGroupRequest{
	Nameservers: []nsgroups.Nameserver{
		{Name: "ns1.example.com"},
		{Name: "ns2.example.com"},
	},
}

group, err := nsgroups.Update(c, "my-ns-group", req)
```

### Delete NS Group

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"

err := nsgroups.Delete(c, "my-ns-group")
```

## Domains

### List Domains

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

results, err := domains.List(c)
```

### Get Domain

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

domain, err := domains.Get(c, 123)
```

### Create Domain

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.CreateDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.OwnerHandle = "owner123"
req.Period = 1

domain, err := domains.Create(c, req)
```

#### Create Domain with NS Group (Recommended)

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.CreateDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.OwnerHandle = "owner123"
req.Period = 1
req.NSGroup = "my-ns-group"

domain, err := domains.Create(c, req)
```

#### Create Domain with Nameservers (Legacy)

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.CreateDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.OwnerHandle = "owner123"
req.Period = 1
req.Nameservers = []domains.Nameserver{
	{Name: "ns1.example.com"},
	{Name: "ns2.example.com"},
}

domain, err := domains.Create(c, req)
```

#### Create Domain with DS Records (DNSSEC)

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.CreateDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.OwnerHandle = "owner123"
req.Period = 1
req.DnssecKeys = []domains.DnssecKey{
	{
		Alg:      8,
		Flags:    257,
		Protocol: 3,
		PubKey:   "AwEAAaz/tAm8yTn4Mfeh5eyI96WSVexTBAvkMgJzkKTOiW1vkIbzxeF3...",
	},
}

domain, err := domains.Create(c, req)
```

### Update Domain

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.UpdateDomainRequest{
    Autorenew: "on",
}

domain, err := domains.Update(c, 123, req)
```

#### Update Domain Nameservers

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.UpdateDomainRequest{
    Nameservers: []domains.Nameserver{
        {Name: "ns1.cloudflare.com"},
        {Name: "ns2.cloudflare.com"},
    },
}

domain, err := domains.Update(c, 123, req)
```

#### Update Domain DS Records

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.UpdateDomainRequest{
    DnssecKeys: []domains.DnssecKey{
        {
            Alg:      8,
            Flags:    257,
            Protocol: 3,
            PubKey:   "AwEAAaz/tAm8yTn4Mfeh5eyI96WSVexTBAvkMgJzkKTOiW1vkIbzxeF3...",
        },
    },
}

domain, err := domains.Update(c, 123, req)
```

### Delete Domain

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

err := domains.Delete(c, 123)
```

### Transfer Domain

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.TransferDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.AuthCode = "12345678"
req.OwnerHandle = "owner123"
req.Autorenew = "on"

domain, err := domains.Transfer(c, req)
```

#### Transfer Domain with NS Group

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.TransferDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.AuthCode = "12345678"
req.OwnerHandle = "owner123"
req.NSGroup = "my-ns-group"

domain, err := domains.Transfer(c, req)
```

#### Transfer Domain with Import Options

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

req := &domains.TransferDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.AuthCode = "12345678"
req.OwnerHandle = "owner123"
req.ImportContactsFromRegistry = true
req.ImportNameserversFromRegistry = true

domain, err := domains.Transfer(c, req)
```

## DNS Records

### List DNS Records

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/dns"

records, err := dns.ListRecords(c, "example.com")
```

### Get DNS Record

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/dns"

record, err := dns.GetRecord(c, "example.com", "www", "A")
```

### Create DNS Record

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/dns"

req := &dns.CreateRecordRequest{
	Name:     "www",
	Type:     "A",
	Value:    "192.0.2.1",
	TTL:      3600,
	Priority: 0,
}

record, err := dns.CreateRecord(c, "example.com", req)
```

### Update DNS Record

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/dns"

req := &dns.UpdateRecordRequest{
	Name:     "www",
	Type:     "A",
	Value:    "192.0.2.2",
	TTL:      7200,
}

record, err := dns.UpdateRecord(c, "example.com", "www", "A", req)
```

### Delete DNS Record

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/dns"

err := dns.DeleteRecord(c, "example.com", "www", "A", "192.0.2.1")
```

### List DNS Zones

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/dns"

zones, err := dns.ListZones(c)
```

### Get DNS Zone

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/dns"

zone, err := dns.GetZone(c, "example.com")
```

## SSL Certificates

### List SSL Orders

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

orders, err := ssl.ListOrders(c)
```

### Get SSL Order

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

order, err := ssl.GetOrder(c, 123)
```

### Create SSL Order

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

req := &ssl.CreateSSLOrderRequest{
	ProductID:          1,
	CommonName:         "example.com",
	AdditionalDomains:  []string{"www.example.com"},
	DomainValidationMethod: "dns",
	Autorenew:          "on",
}

order, err := ssl.CreateOrder(c, req)
```

### Update SSL Order

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

req := &ssl.UpdateSSLOrderRequest{
	Autorenew: "on",
}

order, err := ssl.UpdateOrder(c, 123, req)
```

### Renew SSL Order

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

req := &ssl.RenewSSLOrderRequest{
	Period: 1,
}

order, err := ssl.RenewOrder(c, 123, req)
```

### Reissue SSL Order

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

req := &ssl.ReissueSSLOrderRequest{
	CommonName: "example.com",
	AdditionalDomains: []string{"www.example.com"},
}

order, err := ssl.ReissueOrder(c, 123, req)
```

### Cancel SSL Order

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

err := ssl.CancelOrder(c, 123)
```

### List SSL Products

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

products, err := ssl.ListProducts(c)
```

### Get SSL Product

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/ssl"

product, err := ssl.GetProduct(c, 1)
```

## Contacts

A contact is a login of the control panel (RCP) and the API. It is a different object from a customer: it has an integer id and lives under `/v1beta/contacts`.

### Get Contact

Returns `(nil, nil)` for a contact that does not exist.

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/contacts"

contact, err := contacts.Get(c, 123456)
if err == nil && contact != nil && contact.RCPClientIPList != nil {
	fmt.Println(contact.RCPClientIPList.Allow)
}
```

### Set RCP IP Whitelist

Replaces the whole control panel whitelist of a contact. An empty (or nil) list switches the whitelist off. Only `rcp_client_ip_list.allow` is sent, so the deny list and other fields stay as they are.

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/contacts"

err := contacts.SetRCPAllowList(c, 123456, []string{"203.0.113.10", "2001:db8::/32"})
```
