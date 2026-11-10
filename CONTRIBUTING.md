# Contributing to Openprovider Go API Library

First off, thank you for considering contributing to this project! It's people like you that make the open source community such a great place to learn, inspire, and create.

## Code of Conduct

By participating in this project, you are expected to uphold our Code of Conduct. (Note: You can link to a specific CoC here if you have one).

## How Can I Contribute?

### Reporting Bugs

This section guides you through submitting a bug report. Following these guidelines helps maintainers and the community understand your report, reproduce the behavior, and find related reports.

*   **Use a clear and descriptive title** for the issue to identify the problem.
*   **Describe the exact steps which reproduce the problem** in as many details as possible.
*   **Explain which behavior you expected to see and why.**

### Suggesting Enhancements

This section guides you through submitting an enhancement suggestion, including completely new features and minor improvements to existing functionality.

*   **Use a clear and descriptive title.**
*   **Provide a step-by-step description of the suggested enhancement** in as many details as possible.
*   **Explain why this enhancement would be useful** to most users.

### Pull Requests

*   Fill in the required template (if available).
*   Do not include issue numbers in the PR title.
*   Follow the Go style guidelines (use `gofmt`).
*   Ensure the test suite passes by running `./scripts/test`.
*   Ensure the linter passes by running `./scripts/lint`.
*   **If you modify provider resources or data sources**, regenerate documentation by running `./scripts/docs` and commit the changes. The CI will verify that docs are up-to-date.

#### Sending several pull requests: chain them

Branches must be up to date with `main` before they can merge, and most changes touch the same few files (`internal/provider/provider.go`, `API.md`, `CHANGELOG.md`). Several independent pull requests that all start from `main` thus conflict with each other, and every merge forces a rebase of the rest.

When you have more than one change in flight, please send them as a **chained PR train** instead:

1.  Branch the first PR from `main`.
2.  Branch each next PR from the branch of the PR before it, and set the **base branch** of that PR to the branch before it, not to `main`. Each PR then shows only its own commits.
3.  Keep every PR small and complete: it builds, passes the tests, and stands on its own once the PRs before it are merged.
4.  Mention the order in the description, for example "Follows #195".
5.  We merge from the front of the train. When a PR is merged and its branch is deleted, GitHub retargets the next PR to `main`. If that PR then shows conflicts or extra commits, run `git rebase --onto main <old-parent-branch> <your-branch>` once and push with `--force-with-lease`. With a train this is a single, mechanical step, instead of resolving the same conflicts again for each PR.

If a PR in the middle needs a change, push it to that PR's branch and rebase the later branches on top of it.

## Styleguides

### Git Commit Messages

*   Use the present tense ("Add feature" not "Added feature").
*   Use the imperative mood ("Move cursor to..." not "Moves cursor to...").
*   Limit the first line to 72 characters or less.
*   Commit messages must follow the changelog-valid convention described at
    https://ec.europa.eu/component-library/v1.15.0/eu/docs/conventions/git/.
    This applies to all commits, including AI-assisted commits.
*   Optional template: `templates/commit-message.txt`. Enable with
    `git config commit.template templates/commit-message.txt`.

### Go Styleguide

All Go code is formatted with `gofmt`. We follow the official [Effective Go](https://golang.org/doc/effective_go.html) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) guidelines.

## License

By contributing, you agree that your contributions will be licensed under its [MIT License](LICENSE).
