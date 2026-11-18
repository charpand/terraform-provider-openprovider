# AI Agent Instructions

These guidelines are written for Claude and apply to any AI agent (Copilot, Codex, Junie, etc.) working in this repository. Other agent-specific docs (`AGENTS.md`, `.github/copilot-instructions.md`, `.github/agents/coding-agent.md`) reference this file rather than duplicating it.

## Project Context

This repository contains a Terraform provider for Openprovider, which includes a Go client library for the Openprovider API.

- **Client Library**: Located in `internal/client/`.
- **Terraform Provider**: Located in `internal/provider/`.
- **Test Utilities**: Located in `internal/testutils/`.
- **Scripts**: Located in `scripts/` for bootstrap, lint, test, format, and docs.

## References

- **Project Overview**: `README.md`
- **Contributor Notes**: `CONTRIBUTING.md`
- **API Examples**: `API.md`
- **Openprovider API Documentation**: [swagger.json](https://developer.openprovider.com/data/swagger.json) (API reference portal: [developer.openprovider.com](https://developer.openprovider.com))
- **Openprovider MCP Server**: [mcp.openprovider.com/docs](https://mcp.openprovider.com/docs). A hosted MCP server (`https://mcp.openprovider.com/mcp`) that wraps the API for AI assistants, with a tool reference at `/docs/tools`. It is not a spec source for this repository.

## Workflow

1.  **Bootstrap**: Run `./scripts/bootstrap` to install or update dependencies.
2.  **Linting**: Run `./scripts/lint` to verify code quality.
3.  **Testing**: Run `./scripts/test`. This automatically starts a [Prism](https://mgoblin.github.io/prism/) mock server at `http://localhost:4010`.
4.  **Formatting**: Always run `./scripts/format` (uses `gofmt`).
5.  **Documentation**: Run `./scripts/docs` to regenerate Terraform documentation if resources or data sources are modified.

## Code Standards

- **Go Guidelines**: Follow [Effective Go](https://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- **API Functions**: All API functions MUST take `*client.Client` as the first parameter.
- **Response Handling**: Prefer `Client.JSON`. Where a response is read by hand, close the body in a `defer` and handle the error.
- **Testing Patterns**:
  - Place tests in `*_test.go` files alongside implementation.
  - Use `package <name>_test` for external tests.
  - Use `internal/testutils.SetupTestClient()` to initialize the test client with mock transport.
- **Commit Messages**: Must follow the [changelog-valid convention](https://ec.europa.eu/component-library/v1.15.0/eu/docs/conventions/git/).
  - Optional template: `templates/commit-message.txt`

## Common Patterns

### API Function Implementation
Send the request with `Client.JSON`; it closes the body and decodes the answer. A 404 that means "gone" is `client.IsNotFound(err)`.
```go
func GetResource(c *client.Client, id int) (*Resource, error) {
	var result ResponseEnvelope
	err := c.JSON("GET", fmt.Sprintf("/v1/resource/%d", id), nil, &result)
	if client.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &result.Data, nil
}
```
A reply that needs its own envelope handling can still use `http.NewRequest` with `c.Do`; always close the response body in a `defer`.

### Mock Testing
```go
func TestGetResource(t *testing.T) {
	c := testutils.SetupTestClient()
	
	resource, err := GetResource(c, 123)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if resource == nil {
		t.Fatal("Expected a resource from the mock server, got nil")
	}
}
```

## Validation Checklist Before Completion
1. Code is formatted with `gofmt`.
2. `./scripts/lint` passes.
3. `./scripts/test` passes.
4. `API.md` is updated if new public client APIs were added.
5. `CHANGELOG.md` is updated with significant changes.
