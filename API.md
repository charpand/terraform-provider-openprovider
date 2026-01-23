# API Documentation

## Initialization

```go
import "github.com/charpand/terraform-provider-openprovider/internal/client"

c := client.NewClient(client.Config{
	BaseURL: "https://api.openprovider.eu",
})
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

#### Create Domain with Nameservers

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
