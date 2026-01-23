# API Documentation

## Initialization

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client"

c := client.NewClient(client.Config{
	BaseURL: "https://api.openprovider.eu",
})
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

group, err := nsgroups.Get(c, 123)
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

group, err := nsgroups.Update(c, 123, req)
```

### Delete NS Group

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/nsgroups"

err := nsgroups.Delete(c, 123)
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
	{Hostname: "ns1.example.com"},
	{Hostname: "ns2.example.com"},
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
        {Hostname: "ns1.cloudflare.com"},
        {Hostname: "ns2.cloudflare.com"},
    },
}

domain, err := domains.Update(c, 123, req)
```

### Delete Domain

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client/domains"

err := domains.Delete(c, 123)
```
