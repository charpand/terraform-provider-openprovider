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
- **Openprovider API Documentation**: [swagger.json](https://docs.openprovider.com/swagger.json)

## Workflow

1.  **Bootstrap**: Run `./scripts/bootstrap` to install or update dependencies.
2.  **Linting**: Run `./scripts/lint` to verify code quality.
3.  **Testing**: Run `./scripts/test`. This automatically starts a [Prism](https://mgoblin.github.io/prism/) mock server at `http://localhost:4010`.
4.  **Formatting**: Always run `./scripts/format` (uses `gofmt`).
5.  **Documentation**: Run `./scripts/docs` to regenerate Terraform documentation if resources or data sources are modified.

## Code Standards

- **Go Guidelines**: Follow [Effective Go](https://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).
- **API Functions**: All API functions MUST take `*client.Client` as the first parameter.
- **Response Handling**: Always close response bodies in `defer` statements and handle potential errors.
- **Testing Patterns**:
  - Place tests in `*_test.go` files alongside implementation.
  - Use `package <name>_test` for external tests.
  - Use `internal/testutils.SetupTestClient()` to initialize the test client with mock transport.
- **Commit Messages**: Must follow the [changelog-valid convention](https://ec.europa.eu/component-library/v1.15.0/eu/docs/conventions/git/).
  - Optional template: `templates/commit-message.txt`

## Common Patterns

### API Function Implementation
```go
func GetResource(c *client.Client, id int) (*Resource, error) {
	path := fmt.Sprintf("/v1beta/resource/%d", id)
	req, err := http.NewRequest("GET", fmt.Sprintf("%s%s", c.BaseURL, path), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		return nil, err
	}

	var result ResponseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}
```

### Mock Testing
```go
func TestGetResource(t *testing.T) {
	c := testutils.SetupTestClient()
	
	resource, err := GetResource(c, 123)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	if resource == nil {
		t.Log("Note: No resource returned by mock server")
	}
}
```

## Validation Checklist Before Completion
1. Code is formatted with `gofmt`.
2. `./scripts/lint` passes.
3. `./scripts/test` passes.
4. `API.md` is updated if new public client APIs were added.
5. `CHANGELOG.md` is updated with significant changes.
