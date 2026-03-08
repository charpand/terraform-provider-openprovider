# Claude Agent Instructions

These guidelines apply to Claude-based agents working in this repository.

## References

- Project overview: `README.md`
- Contributor notes: `CONTRIBUTING.md`
- API examples: `API.md`

## Workflow

1. Run `./scripts/bootstrap` to install dependencies
2. Run `./scripts/lint` to verify linting
3. Run `./scripts/test` to verify tests

## Code Standards

- Use `gofmt` for formatting (or run `./scripts/format`)
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- All API functions take `*client.Client` as the first parameter
- Close response bodies in defer statements with error checks

## Commit Messages

- Commit messages must follow the changelog-valid convention at
  https://ec.europa.eu/component-library/v1.15.0/eu/docs/conventions/git/
- Optional template: `templates/commit-message.txt`
