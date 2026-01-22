# GitHub Copilot Instructions

This is a Terraform provider for Openprovider with Go client library code for the Openprovider API.

## Project Structure

- `internal/client/` - API client implementation
- `internal/testutils/` - Test utilities and mock transport
- `main.go` - Provider entry point
- `scripts/` - Build, test, lint, and format scripts
- API documentation: [API.md](../API.md)
- Contributing guide: [CONTRIBUTING.md](../CONTRIBUTING.md)

## Development Workflow

### Before Making Changes
1. Run `./scripts/bootstrap` to install dependencies
2. Run `./scripts/lint` to verify code lints correctly
3. Run `./scripts/test` to verify tests pass

### Code Standards
- Use `gofmt` for formatting (or run `./scripts/format`)
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- All API functions take `*client.Client` as first parameter (from `internal/client`)
- Use proper error handling and close response bodies in defer statements
- Create response structs matching the API schema from [Openprovider Swagger docs](https://docs.openprovider.com/swagger.json)

### Testing
- All tests use the Prism mock server (started automatically by `./scripts/test`)
- Test files: `*_test.go` alongside implementation files
- Use `package <name>_test` for tests
- Use `testutils.MockTransport` for HTTP client testing
- Default mock server: `http://localhost:4010` (override with `TEST_API_BASE_URL`)

### Validation Before Completing
1. Run `./scripts/lint` - must pass
2. Run `./scripts/test` - all tests must pass
3. Ensure code is formatted with `gofmt`
4. Update `API.md` with usage examples if adding new public APIs

## Common Patterns

### Adding New API Endpoints
1. Create implementation file (e.g., `domains/get.go`)
2. Create test file (e.g., `domains/get_test.go`)
3. Add usage example to `API.md`
4. Run `./scripts/lint` and `./scripts/test`

### Example API Function Structure
```go
func Get(c *client.Client, id int) (*Resource, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1beta/resource/%d", c.BaseURL, id), nil)
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
	
	var result GetResourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return &result.Data, nil
}
```

### Example Test Structure
```go
func TestGetResource(t *testing.T) {
	baseURL := os.Getenv("TEST_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4010"
	}
	
	httpClient := &http.Client{
		Transport: &testutils.MockTransport{RT: http.DefaultTransport},
	}
	
	config := client.Config{
		BaseURL:    baseURL,
		Username:   "test",
		Password:   "test",
		HTTPClient: httpClient,
	}
	c := client.NewClient(config)
	
	resource, err := Get(c, 123)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if resource == nil {
		t.Log("Note: No resource returned by mock server")
	}
}
```

## Key References
- **API Documentation**: https://docs.openprovider.com/swagger.json
- **Detailed Agent Instructions**: [agents/coding-agent.md](agents/coding-agent.md)
- **Go Version**: 1.24 (see `go.mod`)
