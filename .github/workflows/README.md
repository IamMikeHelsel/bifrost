# GitHub Actions Workflows

This directory contains the GitHub Actions workflows for the Bifrost project.

## 🔄 Active Workflows

### CI/CD Pipeline (`ci.yml`)
**Triggers**: Push to main/develop, Pull requests
**Purpose**: Comprehensive build, test, and quality assurance

**Features**:
- Go gateway build and test with coverage
- VS Code extension build and packaging
- Security scanning with gosec and Trivy
- Container image building and registry push
- Dependency vulnerability scanning
- Integration testing
- Performance benchmarking

### Automated Release (`release.yml`)
**Triggers**: Weekly schedule (Sunday 2 AM UTC), Manual workflow dispatch
**Purpose**: Automated version management and release creation

**Features**:
- Intelligent change detection since last release
- Semantic version bumping (patch/minor/major)
- Multi-platform binary builds (Linux, macOS, Windows)
- VS Code extension packaging
- Container image publishing to ghcr.io
- GitHub release creation with artifacts
- Automated release notes generation

### Documentation Updates (`docs.yml`)
**Triggers**: Push to main (docs changes), Daily schedule (6 AM UTC), Manual
**Purpose**: Automated documentation generation and deployment

**Features**:
- API documentation generation from Go code
- PlantUML and Mermaid diagram rendering
- MkDocs site building and GitHub Pages deployment
- GitHub Wiki updates with latest documentation
- Architecture diagrams export (SVG/PNG)

## 🔐 Permissions

All workflows are configured with comprehensive permissions to ensure they can perform their tasks:

- `contents: write` - Repository content modifications
- `packages: write` - Container registry publishing
- `pages: write` - GitHub Pages deployment
- `pull-requests: write` - PR interactions
- `security-events: write` - Security scan uploads
- `actions: write` - Workflow interactions

## 🎯 Usage

### Running CI/CD
The CI/CD pipeline runs automatically on:
- Every push to `main` or `develop` branches
- Every pull request to `main`
- Manual trigger via GitHub Actions UI

### Creating Releases
**Automated** (Recommended):
- Releases are created automatically every Sunday if there are substantial changes
- Only creates releases when meaningful code changes are detected

**Manual Release**:
1. Go to Actions → Automated Release
2. Click "Run workflow"
3. Select release type (patch/minor/major)
4. Choose prerelease flag if needed

### Updating Documentation
Documentation updates automatically when:
- Changes are pushed to `docs/` directory
- Any `.md` files are modified
- Manual trigger for full regeneration

## 📊 Monitoring

### CI/CD Status
Monitor the health of the CI/CD pipeline:
- Build success rates
- Test coverage metrics
- Security scan results
- Performance benchmarks

### Release Metrics
Track release automation:
- Release frequency
- Change detection accuracy
- Artifact generation success
- Container image builds

### Documentation Health
Monitor documentation freshness:
- API docs generation
- Diagram rendering
- Site deployment success
- Wiki synchronization

## 🛠 Maintenance

### Workflow Updates
When modifying workflows:
1. Test changes in a feature branch
2. Verify permissions are adequate
3. Update this README if needed
4. Monitor first run after merge

### Dependency Management
Regular maintenance tasks:
- Update action versions (quarterly)
- Review and update Go/Node.js versions
- Check for deprecated actions
- Validate security scanning tools

### Security Considerations
- All workflows use pinned action versions
- Secrets are properly scoped
- SARIF uploads for security integration
- Container scanning for vulnerabilities

## 🔍 Troubleshooting

### Common Issues

**Permission Errors**:
- Verify GitHub token has required permissions
- Check repository settings for Actions permissions

**Build Failures**:
- Check Go/Node.js version compatibility
- Verify dependency availability
- Review security scan results

**Documentation Issues**:
- Ensure PlantUML/Mermaid syntax is valid
- Check MkDocs configuration
- Verify GitHub Pages is enabled

### Getting Help
- Check workflow run logs in GitHub Actions
- Review error messages in annotations
- Consult individual workflow documentation
- Create issue for persistent problems