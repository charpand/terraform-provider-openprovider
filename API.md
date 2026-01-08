# API Documentation

## Initialization

```go
import "github.com/charpand/openprovider-go"

client := openprovider.NewClient(openprovider.Config{
BaseURL: "https://api.openprovider.eu",
})
```

## Domains

### List Domains

```go
import "github.com/charpand/openprovider-go/domains"

results, err := domains.List(client)
```