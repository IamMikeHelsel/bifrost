# Release Card System Documentation

This document provides comprehensive guidance on maintaining and using the Bifrost release card system for documenting tested fieldbus protocols and device compatibility.

## Overview

The Bifrost release card system provides structured documentation for each software release, tracking:
- Protocol support and compatibility
- Device testing coverage (virtual and real hardware)
- Performance benchmarks and metrics
- Testing results and coverage
- Known issues and workarounds

## Directory Structure

```
release-cards/
├── schemas/
│   ├── release-card-schema.yaml    # JSON Schema in YAML format
│   └── release-card-schema.json    # JSON Schema
├── templates/
│   ├── release-card-markdown.mustache    # Markdown template
│   └── release-card-html.mustache        # HTML template
├── examples/
│   ├── v0.1.0-release-card.yaml          # Example YAML card
│   └── v0.1.0-release-card.json          # Example JSON card
└── docs/
    └── maintenance-guide.md               # This file
```

## Schema Overview

The release card schema defines the structure for comprehensive release documentation. Key components include:

### Required Fields
- `version`: Semantic version (e.g., "v0.1.0")
- `release_date`: ISO 8601 date format
- `release_type`: One of "alpha", "beta", "rc", "stable"
- `protocols`: Protocol support matrix
- `testing_summary`: Test execution results

### Optional Fields
- `major_features`: List of significant changes
- `breaking_changes`: Breaking changes with migration guides
- `device_registry`: Device compatibility matrix
- `performance_metrics`: Benchmark results
- `known_issues`: Issues and workarounds
- `regression_notes`: Regression test results
- `metadata`: Card generation metadata

## Creating Release Cards

### 1. Manual Creation

Create a YAML or JSON file following the schema:

```yaml
version: "v1.2.3"
release_date: "2024-12-01"
release_type: "stable"
protocols:
  modbus:
    tcp:
      status: "stable"
      version: "1.1b3"
      # ... additional fields
testing_summary:
  total_tests: 1500
  passed: 1475
  failed: 25
  coverage_percentage: "98.3%"
```

### 2. Automated Generation

The release card system is designed to integrate with CI/CD pipelines for automated generation:

```bash
# Generate from test results
python tools/generate-release-card.py \
  --version v1.2.3 \
  --test-results build/test-results.json \
  --benchmark-results build/benchmarks.json \
  --output release-cards/v1.2.3-release-card.yaml
```

### 3. Schema Validation

Validate release cards against the schema:

```bash
# Using ajv-cli (install with: npm install -g ajv-cli)
ajv validate -s release-cards/schemas/release-card-schema.json \
    -d release-cards/examples/v0.1.0-release-card.json

# Using Python jsonschema
python -c "
import json, yaml
from jsonschema import validate
with open('release-cards/schemas/release-card-schema.json') as f:
    schema = json.load(f)
with open('release-cards/examples/v0.1.0-release-card.yaml') as f:
    data = yaml.safe_load(f)
validate(data, schema)
print('Valid!')
"
```

## Output Formats

### 1. Markdown Generation

Generate human-readable markdown using Mustache templates:

```bash
# Using mustache CLI (install with: npm install -g mustache)
mustache release-cards/examples/v0.1.0-release-card.json \
         release-cards/templates/release-card-markdown.mustache \
         > v0.1.0-release-card.md
```

### 2. HTML Generation

Generate web-friendly HTML:

```bash
mustache release-cards/examples/v0.1.0-release-card.json \
         release-cards/templates/release-card-html.mustache \
         > v0.1.0-release-card.html
```

### 3. PDF Generation

Convert HTML to PDF using tools like wkhtmltopdf or Puppeteer:

```bash
# Using wkhtmltopdf
wkhtmltopdf v0.1.0-release-card.html v0.1.0-release-card.pdf

# Using Puppeteer (Node.js)
node -e "
const puppeteer = require('puppeteer');
(async () => {
  const browser = await puppeteer.launch();
  const page = await browser.newPage();
  await page.goto('file://' + process.cwd() + '/v0.1.0-release-card.html');
  await page.pdf({path: 'v0.1.0-release-card.pdf', format: 'A4'});
  await browser.close();
})();
"
```

## Protocol Documentation Guidelines

### Protocol Support Matrix

Document each protocol variant with:

```yaml
protocols:
  protocol_family:        # e.g., modbus, opcua, ethernetip
    variant:             # e.g., tcp, rtu, client, server
      status: "stable"   # stable, beta, experimental, deprecated
      version: "1.1b3"   # Protocol version implemented
      performance:
        throughput: "1000+ regs/sec"
        latency: "<1ms"
        concurrent_connections: 100
      tested_devices:
        virtual: ["simulator_name"]
        real:
          - device: "Device Model"
            vendor: "Vendor Name"
            firmware_version: "x.y.z"
            test_date: "2024-12-01"
      limitations:
        - "Specific limitation description"
```

### Status Definitions

- **stable**: Production-ready, thoroughly tested
- **beta**: Feature-complete, testing in progress
- **experimental**: Early implementation, limited testing
- **deprecated**: No longer supported, use alternatives
- **unsupported**: Not implemented or broken

## Device Testing Documentation

### Virtual Device Testing

```yaml
tested_devices:
  virtual:
    - "modbus_tcp_simulator_v1.0"
    - "opcua_server_mock_v2.1"
```

### Real Hardware Testing

```yaml
tested_devices:
  real:
    - device: "ModiconM340"
      vendor: "Schneider Electric"
      firmware_version: "2.70"
      test_date: "2024-11-15"
      notes: "Full register access verified"
```

### Device Registry

Organize devices by category:

```yaml
device_registry:
  total_devices_tested: 25
  virtual_devices: 15
  real_devices: 10
  device_categories:
    PLCs:
      - name: "Modicon M340"
        vendor: "Schneider Electric"
        status: "passed"
        protocols_supported: ["modbus_tcp", "modbus_rtu"]
    HMIs:
      - name: "PanelView Plus"
        vendor: "Allen-Bradley"
        status: "partial"
        protocols_supported: ["ethernetip"]
```

## Performance Metrics

### Benchmark Environment

Always document the testing environment:

```yaml
performance_metrics:
  benchmark_environment:
    cpu: "Intel i7-12700K @ 3.6GHz"
    memory: "32GB DDR4-3200"
    os: "Ubuntu 22.04 LTS"
    go_version: "1.22.0"
```

### Performance Measurements

Include comprehensive metrics:

```yaml
protocol_benchmarks:
  modbus_tcp:
    operations_per_second: 18879
    latency_p50: "45µs"    # 50th percentile
    latency_p95: "120µs"   # 95th percentile
    latency_p99: "250µs"   # 99th percentile
    max_concurrent_connections: 100
```

## Testing Documentation

### Test Categories

Organize test results by category:

```yaml
test_categories:
  unit_tests:
    total: 850
    passed: 840
    failed: 10
    coverage: "92%"
  
  integration_tests:
    total: 280
    passed: 265
    failed: 15
    coverage: "75%"
  
  performance_tests:
    total: 85
    passed: 70
    failed: 10
    coverage: "85%"
  
  security_tests:
    total: 35
    passed: 25
    failed: 0
    coverage: "71%"
```

### Manual Testing

Document manual test scenarios:

```yaml
manual_tests:
  scenarios_tested: 45
  scenarios_passed: 42
  tester: "QA Team"
  test_date: "2024-11-29"
```

## Known Issues and Regression Notes

### Issue Documentation

```yaml
known_issues:
  - issue: "Modbus RTU timing inconsistencies on Windows"
    severity: "medium"      # critical, high, medium, low
    workaround: "Use TCP variant or reduce polling frequency"
    tracking_url: "https://github.com/IamMikeHelsel/bifrost/issues/47"
    expected_fix: "v0.2.0"
```

### Regression Testing

```yaml
regression_notes:
  - test: "Modbus TCP connection stability"
    status: "passed"        # passed, failed, skipped, new
    previous_status: "failed"
    notes: "Fixed memory leak in connection pool"
```

## CI/CD Integration

### GitHub Actions Example

```yaml
# .github/workflows/release-card.yml
name: Generate Release Card

on:
  push:
    tags: ['v*']

jobs:
  generate-release-card:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Run Tests
        run: |
          make test-all
          make benchmark-all
      
      - name: Generate Release Card
        run: |
          python tools/generate-release-card.py \
            --version ${{ github.ref_name }} \
            --test-results build/test-results.json \
            --benchmark-results build/benchmarks.json
      
      - name: Create Release
        uses: ncipollo/release-action@v1
        with:
          artifacts: |
            release-cards/${{ github.ref_name }}-release-card.yaml
            release-cards/${{ github.ref_name }}-release-card.json
            release-cards/${{ github.ref_name }}-release-card.md
            release-cards/${{ github.ref_name }}-release-card.html
```

## Best Practices

### 1. Consistent Naming

- Use semantic versioning for release cards
- Name files: `vX.Y.Z-release-card.{yaml,json,md,html}`
- Store by version: `release-cards/v1.2.3/`

### 2. Comprehensive Testing

- Test all protocol variants before release
- Include both virtual and real device testing
- Document performance baselines
- Track regression test results

### 3. Clear Documentation

- Use descriptive limitation statements
- Provide specific workarounds for known issues
- Include migration guides for breaking changes
- Document vendor-specific compatibility notes

### 4. Automation

- Integrate with CI/CD pipelines
- Automate test result collection
- Generate cards from templates
- Validate against schema before release

### 5. Versioning

- Version the schema itself
- Track card format versions
- Maintain backward compatibility
- Document schema changes

## Tools and Dependencies

### Required Tools

- **JSON Schema validator**: ajv-cli or Python jsonschema
- **Template engine**: Mustache CLI or library
- **YAML parser**: PyYAML or js-yaml
- **PDF generator**: wkhtmltopdf or Puppeteer

### Installation

```bash
# Node.js tools
npm install -g ajv-cli mustache

# Python tools
pip install jsonschema pyyaml

# PDF generation
sudo apt-get install wkhtmltopdf  # Linux
brew install wkhtmltopdf          # macOS
```

## Troubleshooting

### Schema Validation Errors

1. Check required fields are present
2. Verify date formats (ISO 8601)
3. Ensure enum values are valid
4. Validate array structures

### Template Rendering Issues

1. Check JSON/YAML syntax
2. Verify template variable names
3. Test with minimal data first
4. Check for missing optional fields

### Performance Issues

1. Large release cards may be slow to render
2. Consider pagination for device lists
3. Optimize images and assets for HTML/PDF
4. Use streaming for large datasets

## Support and Contributions

For questions, issues, or contributions to the release card system:

1. Check existing issues in the GitHub repository
2. Review this documentation thoroughly
3. Test changes with example data
4. Submit pull requests with tests
5. Update documentation for new features

---

*This documentation is part of the Bifrost project. For more information, visit the [GitHub repository](https://github.com/IamMikeHelsel/bifrost).*