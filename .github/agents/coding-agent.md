# Coding Agent Instructions for openprovider-go

This document provides guidelines for AI coding agents working on the openprovider-go repository.

## Project Overview

This is a Go client library for the Openprovider.org API. The library follows RESTful API patterns and uses the Openprovider API documented at https://docs.openprovider.com/swagger.json.

## Development Workflow

### 1. Initial Setup

Before making any changes:
1. Run `./scripts/bootstrap` to install dependencies
2. Run `./scripts/lint` to verify the code lints correctly
3. Run `./scripts/test` to verify tests pass

### 2. Making Changes

When implementing new endpoints or features:
- Follow the existing code patterns in the repository
- Use Go formatting with `gofmt`
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Keep changes minimal and focused

### 3. Testing

- All tests use the Prism mock server (started via `./scripts/mock`)
- Tests should be in `*_test.go` files alongside the implementation
- Use `testutils.MockTransport` for HTTP client testing
- Environment variable `TEST_API_BASE_URL` can override the mock server URL (defaults to `http://localhost:4010`)
- Run `./scripts/test` to execute all tests

### 4. Validation

Before completing your work:
1. Run `./scripts/lint` - must pass
2. Run `./scripts/test` - all tests must pass
3. Ensure code is formatted with `gofmt`
4. Update API.md with usage examples if adding new public APIs

## Code Structure and Patterns

### Project Structure

```
openprovider-go/
├── .github/
│   ├── workflows/        # CI/CD workflows
│   └── agents/          # Coding agent instructions
├── domains/             # Domain-related API endpoints
├── authentication/      # Authentication functionality
├── internal/
│   └── testutils/       # Test utilities and mock transport
├── scripts/             # Build, test, and lint scripts
├── client.go            # Main client implementation
└── API.md              # API documentation with examples
```

### Implementing New Endpoints

When adding new API endpoints (e.g., based on Openprovider Swagger docs):

#### 1. Create the implementation file

Example: `domains/get.go`

```go
// Package domains provides functionality for working with domains.
package domains

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/charpand/openprovider-go"
)

// GetDomainResponse represents a response for a single domain.
type GetDomainResponse struct {
	Code int    `json:"code"`
	Data Domain `json:"data"`
}

// Get retrieves a single domain by ID from the Openprovider API.
//
// Endpoint: GET https://api.openprovider.eu/v1beta/domains/{id}
func Get(c *openprovider.Client, id int) (*Domain, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1beta/domains/%d", c.BaseURL, id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	var result GetDomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
```

#### 2. Create the test file

Example: `domains/get_test.go`

```go
// Package domains_test contains tests for the domains package.
package domains_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/charpand/openprovider-go"
	"github.com/charpand/openprovider-go/domains"
	"github.com/charpand/openprovider-go/internal/testutils"
)

func TestGetDomain(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}

	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}

	config := openprovider.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	client := openprovider.NewClient(config)

	// Use an example ID that exists in your OpenAPI examples/mock
	domain, err := domains.Get(client, 123)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if domain == nil {
		t.Log("Note: No domain returned by mock server (check your swagger examples)")
	}
}
```

#### 3. Update API Documentation

Add usage examples to `API.md`:

```markdown
### Get Domain

\`\`\`go
import "github.com/charpand/openprovider-go/domains"

domain, err := domains.Get(client, 123)
\`\`\`
```

## Code Style Guidelines

### From CONTRIBUTING.md

- Use `gofmt` to format all Go code
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use the present tense in commit messages ("Add feature" not "Added feature")
- Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
- Limit the first line of commit messages to 72 characters or less

### API Client Patterns

- All API functions take `*openprovider.Client` as the first parameter
- Use proper error handling and return errors up the stack
- Close response bodies in defer statements with proper error checking
- Use `json.NewDecoder` for parsing JSON responses
- Create response structs that match the API schema from Swagger docs

### Test Patterns

- Test files are in `*_test.go` files
- Use `package <name>_test` for package-level tests
- Use `testutils.MockTransport` for HTTP mocking
- Respect `TEST_API_BASE_URL` environment variable
- Tests should work with the Prism mock server (http://localhost:4010)

## CI/CD

The project uses GitHub Actions for CI/CD (`.github/workflows/ci.yml`):

- **Lint job**: Runs `./scripts/lint` using golangci-lint or go vet
- **Test job**: Runs `./scripts/bootstrap` then `./scripts/test`
- Both jobs run on push and pull requests
- Timeout: 10 minutes for lint, 30 minutes for tests

## Key Scripts

- `./scripts/bootstrap` - Install dependencies and setup environment
- `./scripts/lint` - Run golangci-lint or go vet, check test compilation
- `./scripts/test` - Start Prism mock server and run Go tests
- `./scripts/mock` - Start Prism mock server manually
- `./scripts/format` - Format Go code

## Important References

- **API Documentation**: https://docs.openprovider.com/swagger.json
- **Issue #44**: Contains detailed example implementation pattern
- **CONTRIBUTING.md**: Full contributing guidelines
- **API.md**: User-facing API documentation with examples

## Common Tasks

### Adding a new endpoint

1. Check Openprovider Swagger docs for the endpoint specification
2. Create implementation file (e.g., `domains/get.go`)
3. Create test file (e.g., `domains/get_test.go`)
4. Update `API.md` with usage example
5. Run `./scripts/lint` to verify code quality
6. Run `./scripts/test` to verify tests pass
7. Ensure all code is formatted with `gofmt`

### Fixing a bug

1. Write a failing test that reproduces the bug
2. Fix the bug with minimal changes
3. Verify the test now passes
4. Run full test suite with `./scripts/test`
5. Run linter with `./scripts/lint`

### Updating dependencies

1. Modify `go.mod` as needed
2. Run `go mod tidy`
3. Run `./scripts/test` to ensure nothing breaks
4. Run `./scripts/lint` to check for issues

## Notes

- The project uses Go 1.23 (see `go.mod`)
- Prism mock server runs on port 4010 by default
- The test script automatically starts/stops the mock server
- All public functions should have documentation comments
- Struct fields must match the Openprovider API response schema
