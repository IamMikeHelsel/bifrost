# Bifrost Release Card System

A comprehensive system for documenting tested fieldbus protocols and device compatibility for each Bifrost software release.

## 🎯 Purpose

The release card system provides:
- **Protocol Compatibility Matrix**: Detailed support status for industrial protocols
- **Device Testing Coverage**: Virtual and real hardware validation results
- **Performance Benchmarks**: Throughput and latency metrics per protocol
- **Testing Documentation**: Comprehensive test results and coverage
- **Quality Assurance**: Known issues, limitations, and workarounds

## 📁 Directory Structure

```
release-cards/
├── schemas/                          # JSON Schema definitions
│   ├── release-card-schema.yaml      # Schema in YAML format
│   └── release-card-schema.json      # Schema in JSON format
├── templates/                        # Output format templates
│   ├── release-card-markdown.mustache # Markdown template
│   └── release-card-html.mustache    # HTML template
├── examples/                         # Example release cards
│   ├── v0.1.0-release-card.yaml      # YAML example
│   └── v0.1.0-release-card.json      # JSON example
└── docs/                            # Documentation
    └── maintenance-guide.md          # Comprehensive guide
```

## 🚀 Quick Start

### 1. View Example Release Card

Check out the v0.1.0 example release card:
- [YAML format](examples/v0.1.0-release-card.yaml)
- [JSON format](examples/v0.1.0-release-card.json)

### 2. Validate Schema

```bash
# Install validator
npm install -g ajv-cli

# Validate example
ajv validate -s schemas/release-card-schema.json \
              -d examples/v0.1.0-release-card.json
```

### 3. Generate Documentation

```bash
# Install template engine
npm install -g mustache

# Generate Markdown
mustache examples/v0.1.0-release-card.json \
         templates/release-card-markdown.mustache \
         > v0.1.0-release-card.md

# Generate HTML
mustache examples/v0.1.0-release-card.json \
         templates/release-card-html.mustache \
         > v0.1.0-release-card.html
```

## 📋 Schema Overview

### Required Components

Every release card must include:

```yaml
version: "v1.0.0"              # Semantic version
release_date: "2024-12-01"     # ISO 8601 date
release_type: "stable"         # alpha, beta, rc, stable

protocols:                     # Protocol support matrix
  modbus:
    tcp:
      status: "stable"         # Implementation status
      # ... performance, devices, limitations

testing_summary:               # Test execution results
  total_tests: 1250
  passed: 1200
  failed: 50
  coverage_percentage: "96%"
```

### Optional Components

- `major_features`: Key features in this release
- `breaking_changes`: Changes requiring migration
- `device_registry`: Device compatibility matrix
- `performance_metrics`: Benchmark results
- `known_issues`: Issues and workarounds
- `regression_notes`: Regression test results

## 🔧 Protocol Documentation

### Status Levels

- **stable**: Production-ready, thoroughly tested
- **beta**: Feature-complete, testing in progress
- **experimental**: Early implementation, limited testing
- **deprecated**: No longer supported
- **unsupported**: Not implemented

### Example Protocol Entry

```yaml
protocols:
  modbus:
    tcp:
      status: "stable"
      version: "1.1b3"
      performance:
        throughput: "1000+ regs/sec"
        latency: "<1ms"
        concurrent_connections: 100
      tested_devices:
        virtual: ["modbus_tcp_simulator_v1.0"]
        real:
          - device: "ModiconM340"
            vendor: "Schneider Electric"
            firmware_version: "2.70"
            test_date: "2024-11-15"
      limitations:
        - "No RTU over TCP support"
        - "Limited to 247 device addresses"
      vendor_compatibility:
        - vendor: "Schneider Electric"
          status: "full"
          notes: "Tested with Modicon series PLCs"
```

## 📊 Performance Metrics

### Benchmark Documentation

```yaml
performance_metrics:
  benchmark_environment:
    cpu: "Intel i7-12700K @ 3.6GHz"
    memory: "32GB DDR4-3200"
    os: "Ubuntu 22.04 LTS"
    go_version: "1.22.0"
  
  protocol_benchmarks:
    modbus_tcp:
      operations_per_second: 18879
      latency_p50: "45µs"
      latency_p95: "120µs"
      latency_p99: "250µs"
      max_concurrent_connections: 100
```

## 🧪 Testing Documentation

### Test Categories

```yaml
testing_summary:
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
```

### Manual Testing

```yaml
manual_tests:
  scenarios_tested: 45
  scenarios_passed: 42
  tester: "QA Team"
  test_date: "2024-11-29"
```

## 🏭 Industrial Protocols Supported

The release card system tracks compatibility for:

- **Modbus**: TCP/RTU variants
- **OPC UA**: Client/Server implementations
- **Ethernet/IP**: Scanner/Adapter modes
- **Siemens S7**: Communication protocols
- **BACnet**: Building automation protocols
- **DNP3**: SCADA protocols

## 📈 Device Compatibility Matrix

Track testing across device categories:

```yaml
device_registry:
  device_categories:
    PLCs:          # Programmable Logic Controllers
      - name: "Modicon M340"
        vendor: "Schneider Electric"
        status: "passed"
        protocols_supported: ["modbus_tcp"]
    
    HMIs:          # Human Machine Interfaces
      - name: "PanelView Plus"
        vendor: "Allen-Bradley"
        status: "partial"
        protocols_supported: ["ethernetip"]
    
    Gateways:      # Protocol Gateways
      - name: "MB Gateway"
        vendor: "Anybus"
        status: "untested"
```

## 🔄 CI/CD Integration

### Automated Generation

The release card system integrates with CI/CD pipelines:

```yaml
# GitHub Actions workflow snippet
- name: Generate Release Card
  run: |
    python tools/generate-release-card.py \
      --version ${{ github.ref_name }} \
      --test-results build/test-results.json \
      --benchmark-results build/benchmarks.json \
      --device-results build/device-tests.json \
      --output release-cards/${{ github.ref_name }}-release-card.yaml

- name: Validate Schema
  run: |
    ajv validate -s release-cards/schemas/release-card-schema.json \
                  -d release-cards/${{ github.ref_name }}-release-card.yaml

- name: Generate Documentation
  run: |
    mustache release-cards/${{ github.ref_name }}-release-card.yaml \
             release-cards/templates/release-card-markdown.mustache \
             > ${{ github.ref_name }}-release-card.md
```

## 📝 Output Formats

### 1. Markdown
Human-readable format for GitHub releases and documentation.

### 2. JSON
Machine-readable format for API integration and data processing.

### 3. HTML
Web-friendly format with styling for online documentation.

### 4. PDF
Professional format for customer deliverables and reports.

## ⚠️ Known Issues Documentation

Track and communicate issues clearly:

```yaml
known_issues:
  - issue: "Modbus RTU timing inconsistencies on Windows"
    severity: "medium"
    workaround: "Use TCP variant or reduce polling frequency"
    tracking_url: "https://github.com/IamMikeHelsel/bifrost/issues/47"
    expected_fix: "v0.2.0"
```

## 🔧 Installation and Setup

### Prerequisites

```bash
# Node.js tools for validation and templating
npm install -g ajv-cli mustache

# Python tools for YAML processing
pip install jsonschema pyyaml

# Optional: PDF generation
sudo apt-get install wkhtmltopdf  # Linux
brew install wkhtmltopdf          # macOS
```

### Usage Examples

```bash
# Validate a release card
ajv validate -s schemas/release-card-schema.json \
              -d examples/v0.1.0-release-card.json

# Generate Markdown documentation
mustache examples/v0.1.0-release-card.json \
         templates/release-card-markdown.mustache \
         > output.md

# Generate HTML documentation
mustache examples/v0.1.0-release-card.json \
         templates/release-card-html.mustache \
         > output.html

# Convert HTML to PDF
wkhtmltopdf output.html output.pdf
```

## 📚 Documentation

- [Maintenance Guide](docs/maintenance-guide.md): Comprehensive guide for maintaining release cards
- [Schema Reference](schemas/release-card-schema.yaml): Complete schema documentation
- [Examples](examples/): Example release cards in multiple formats
- [Templates](templates/): Output format templates

## 🤝 Contributing

1. Follow the established schema structure
2. Validate all changes against the schema
3. Test template rendering with example data
4. Update documentation for new features
5. Include comprehensive test coverage

## 📄 License

This release card system is part of the Bifrost project and follows the same licensing terms.

---

**Note**: This release card system is designed specifically for industrial automation and fieldbus protocol testing. It provides comprehensive documentation for users evaluating protocol compatibility and device support in industrial environments.