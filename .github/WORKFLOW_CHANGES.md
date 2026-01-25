# GitHub Actions Workflow Changes

## Problem

The documentation workflow (`.github/workflows/docs.yml`) was failing due to branch protection rules that require changes to be made through pull requests. The workflow attempted to automatically generate and commit documentation changes directly to the `main` branch, which was blocked with the following error:

```
remote: error: GH013: Repository rule violations found for refs/heads/main.
remote: - Changes must be made through a pull request.
remote: ! [remote rejected] main -> main (push declined due to repository rule violations)
```

## Solution

The workflow has been restructured to comply with branch protection rules while ensuring documentation remains up-to-date.

### Changes Made

#### 1. **Docs Workflow (`.github/workflows/docs.yml`)**

**Before:** 
- Triggered on push to `main` branch
- Generated docs and automatically committed them to `main`
- Required `contents: write` permissions

**After:**
- Triggers on `pull_request` events and manual `workflow_dispatch`
- Generates docs and validates them
- Checks for uncommitted changes and fails if docs are out-of-date
- Only requires `contents: read` permissions
- No longer attempts to commit changes

#### 2. **CI Workflow (`.github/workflows/ci.yml`)**

**Added:** New `docs` job that runs alongside `lint` and `test` jobs
- Generates documentation
- Validates documentation structure
- Verifies that all documentation changes are committed

This ensures that every pull request is automatically checked for up-to-date documentation.

#### 3. **Contributing Guidelines (`CONTRIBUTING.md`)**

**Added:** Clear instructions for contributors to regenerate documentation when modifying provider resources or data sources.

### Developer Workflow

When making changes that affect provider resources or data sources:

1. Make your code changes
2. Run `./scripts/docs` to regenerate documentation
3. Commit both code and documentation changes
4. Create a pull request

The CI workflow will automatically verify that documentation is up-to-date. If you forget to regenerate docs, the CI check will fail with a clear error message:

```
Documentation is out of date. Please run './scripts/docs' and commit the changes.
```

### Benefits

1. **Complies with branch protection rules** - No direct pushes to `main`
2. **Maintains documentation quality** - Automated checks ensure docs stay in sync
3. **Clear feedback** - Contributors get immediate feedback if docs are out-of-date
4. **Transparent process** - All changes, including docs, are reviewed in PRs
5. **No special permissions needed** - Works with read-only access to the repository

### Alternative Considered

An alternative would have been to configure branch protection rules with an exclusion for the `github-actions[bot]` user. However, this approach:
- Reduces security by allowing automated commits to bypass review
- Makes it harder to track when and why documentation changed
- Could potentially allow broken documentation to be committed without review

The chosen solution provides better transparency and code review while maintaining automation through CI checks.

## Testing

To test the new workflow locally:

```bash
# Bootstrap dependencies
./scripts/bootstrap

# Generate documentation
./scripts/docs

# Validate documentation
go tool tfplugindocs validate --provider-name openprovider

# Check for uncommitted changes
git diff --exit-code docs/ templates/
```

If the last command exits with a non-zero status, you need to commit the documentation changes.
