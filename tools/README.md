# Release Card CI/CD Integration Tools

This directory contains tools for automated release card generation and deployment as part of the Bifrost Gateway CI/CD pipeline.

## Tools

### `generate-release-card.py`
Generates release cards documenting tested fieldbus protocols, device compatibility, performance metrics, and testing coverage.

**Usage:**
```bash
python tools/generate-release-card.py --version v0.1.0 --test-results test-results --verbose
```

**Features:**
- Collects test results from virtual device tests and Go gateway tests
- Processes performance benchmark data
- Evaluates quality gates for release approval
- Generates release cards in multiple formats (Markdown, JSON, YAML)

### `deploy-docs.py`
Deploys generated documentation and release cards to the documentation site.

**Usage:**
```bash
python tools/deploy-docs.py --verbose
python tools/deploy-docs.py --dry-run  # Test mode
```

**Features:**
- Prepares documentation site with release cards
- Supports GitHub Pages deployment
- Creates navigation indexes for release cards
- Handles artifact uploads for CI/CD

## Scripts

### `scripts/create-mock-test-results.sh`
Creates mock test results for testing the release card generation workflow.

### `scripts/test-release-card-integration.sh`
Integration test that validates the complete release card workflow.

## GitHub Actions Workflow

The main workflow is defined in `.github/workflows/release-cards.yml` and includes:

1. **Virtual Device Testing** - Runs tests against protocol simulators
2. **Performance Benchmarking** - Executes performance tests and collects metrics
3. **Release Card Generation** - Creates release cards from test results
4. **Quality Gate Validation** - Ensures release meets quality standards
5. **Documentation Deployment** - Deploys documentation and release cards
6. **GitHub Release Creation** - Creates GitHub release with artifacts

## Workflow Triggers

- **Tag Push**: Automatically triggers on version tags (`v*`)
- **Manual Dispatch**: Can be triggered manually with custom version

## Quality Gates

The workflow enforces the following quality gates:
- **Test Coverage**: Virtual device tests and Go gateway tests must pass
- **Performance Targets**: Benchmarks must meet performance criteria
- **Documentation Completeness**: Release card must have complete information

## Usage Examples

### Generate a release card locally:
```bash
# Create mock test results
just mock-test-results

# Generate release card
just release-card v0.1.0

# Deploy documentation
just deploy-docs
```

### Test the complete integration:
```bash
just test-release-cards
```

### Trigger the GitHub Actions workflow:
```bash
# Create and push a version tag
git tag v0.1.0
git push origin v0.1.0

# Or trigger manually via GitHub web interface
```

## Configuration

### Documentation Deployment
Configure deployment settings in `docs/deploy-config.yaml`:

```yaml
github_pages:
  enabled: true
  branch: gh-pages
  directory: _site

artifacts:
  enabled: true
  retention_days: 90
```

## Output Formats

Release cards are generated in multiple formats:
- **Markdown** (`.md`): Human-readable format for GitHub releases
- **JSON** (`.json`): Structured data for API consumption
- **YAML** (`.yaml`): Configuration-friendly format

## Dependencies

- Python 3.12+
- PyYAML for configuration handling
- Go 1.22+ for performance testing
- GitHub Actions environment for CI/CD

## Integration with Existing Workflows

This integrates with the existing CI/CD infrastructure:
- Builds on existing `ci.yml` and `release.yml` workflows
- Uses existing Go performance testing framework
- Leverages virtual device testing infrastructure
- Extends documentation deployment capabilities