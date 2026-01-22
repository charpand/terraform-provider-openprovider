# API Documentation

## Initialization

```go
import "github.com/charpand/terraform-provider-openprovider"

client := openprovider.NewClient(openprovider.Config{
BaseURL: "https://api.openprovider.eu",
})
```

## Domains

### List Domains

```go
import "github.com/charpand/terraform-provider-openprovider/domains"

results, err := domains.List(client)
```

### Get Domain

```go
import "github.com/charpand/terraform-provider-openprovider/domains"

domain, err := domains.Get(client, 123)
```

### Create Domain

```go
import "github.com/charpand/openprovider-go/domains"

req := &domains.CreateDomainRequest{}
req.Domain.Name = "example"
req.Domain.Extension = "com"
req.OwnerHandle = "owner123"
req.Period = 1

domain, err := domains.Create(client, req)
```

### Update Domain

```go
import "github.com/charpand/openprovider-go/domains"

req := &domains.UpdateDomainRequest{
    Autorenew: "on",
}

domain, err := domains.Update(client, 123, req)
```

### Delete Domain

```go
import "github.com/charpand/openprovider-go/domains"

err := domains.Delete(client, 123)
```