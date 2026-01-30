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
