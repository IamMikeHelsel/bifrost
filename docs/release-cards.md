# Release Card System Documentation

The Release Card System provides comprehensive documentation of tested fieldbus protocols and device compatibility for each Bifrost software release.

## Overview

Release cards automatically capture and document:

- **Protocol Support Matrix**: Which industrial protocols are supported and their implementation status
- **Device Compatibility**: Virtual and real hardware devices that have been tested
- **Performance Metrics**: Throughput, latency, and resource usage benchmarks
- **Testing Coverage**: Automated, manual, regression, and security test results
- **Quality Metrics**: Code coverage, security scores, and reliability assessments
- **Known Issues**: Documented problems with workarounds and target fixes

## System Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Test Results  │    │  Performance     │    │ Device Registry │
│   (JUnit XML)   │    │  Benchmarks      │    │ (Auto-detected) │
│                 │    │  (JSON)          │    │                 │
└─────────┬───────┘    └─────────┬────────┘    └─────────┬───────┘
          │                      │                       │
          └──────────────────────┼───────────────────────┘
                                 │
                        ┌────────▼────────┐
                        │  Data Collector │
                        │   (collect.py)  │
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │ Release Data    │
                        │    (YAML)       │
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │    Validator    │
                        │  (validate.py)  │
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │   Generator     │
                        │ (generate.py)   │
                        └────────┬────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          │                      │                      │
┌─────────▼─────────┐  ┌─────────▼─────────┐  ┌─────────▼─────────┐
│    Markdown       │  │       HTML        │  │      JSON         │
│  (GitHub Release) │  │  (Documentation)  │  │   (API Access)    │
└───────────────────┘  └───────────────────┘  └───────────────────┘
```

## Usage

### Manual Generation

1. **Collect Data:**
   ```bash
   python release-cards/tools/collect.py \
     --project-root . \
     --output release-data.yaml \
     --version "0.1.0" \
     --release-type alpha \
     --test-results test-results/junit.xml \
     --benchmarks test-results/benchmarks.json
   ```

2. **Validate Data:**
   ```bash
   python release-cards/tools/validate.py release-data.yaml --strict
   ```

3. **Generate Documentation:**
   ```bash
   python release-cards/tools/generate.py release-data.yaml \
     --output-dir ./output \
     --formats markdown,html,json
   ```

### Automated Generation (CI/CD)

The system automatically generates release cards when:

- **Version tags are pushed**: `git tag v0.1.0 && git push --tags`
- **Manual workflow trigger**: Use GitHub Actions "Generate Release Card" workflow

### Integration with Existing Workflows

The release card system integrates with:

- **Virtual Device Testing**: Automatically detects simulators in `virtual-devices/`
- **Performance Benchmarks**: Collects data from pytest-benchmark results
- **CI/CD Pipeline**: Generates cards on every release
- **GitHub Releases**: Automatically attaches cards to releases
- **Documentation Site**: Deploys to GitHub Pages

## Schema Definition

Release cards follow a structured schema with the following main sections:

### Release Information
```yaml
version: "0.1.0"
release_date: "2024-12-01"
release_type: "alpha"  # alpha, beta, rc, stable
release_notes: "Brief description of changes"
breaking_changes: []
migration_notes: "Migration guidance"
```

### Protocol Support Matrix
```yaml
protocols:
  modbus:
    tcp:
      status: "beta"  # stable, beta, experimental, unsupported
      version: "3.9.2"
      performance:
        throughput: "1000+ regs/sec"
        latency: "<1ms"
        concurrent_limit: 100
        memory_usage: "<10MB per connection"
      tested_devices:
        virtual: ["modbus_tcp_simulator_v1.0"]
        real: []
      limitations: ["No RTU over TCP support"]
      known_issues: ["Connection timeout handling needs improvement"]
```

### Device Registry
```yaml
device_registry:
  virtual_devices:
    - name: "modbus_tcp_simulator_v1.0"
      protocol: "modbus_tcp"
      version: "1.0.0"
      test_coverage: "85%"
      status: "passing"  # passing, failing, unstable
      last_tested: "2024-11-30"
  
  real_devices:
    - vendor: "Schneider Electric"
      model: "M340 PLC"
      firmware: "2.90"
      protocol: "modbus_tcp"
      test_coverage: "75%"
      status: "passing"
      last_tested: "2024-11-25"
      location: "Lab A"
      notes: "Full feature compatibility"
```

### Testing Summary
```yaml
testing_summary:
  automated_tests:
    total: 156
    passed: 142
    failed: 8
    skipped: 6
    coverage: "78%"
  
  manual_tests:
    total: 24
    passed: 20
    failed: 2
    pending: 2
    coverage: "65%"
```

### Performance Benchmarks
```yaml
performance_benchmarks:
  test_environment:
    os: "Ubuntu 22.04 LTS"
    cpu: "Intel Core i7-12700K @ 3.6GHz"
    memory: "32GB DDR4"
    network: "Gigabit Ethernet"
    load_conditions: "Standard lab environment"
  
  results:
    - protocol: "modbus_tcp"
      test_type: "throughput"
      metric: "regs/sec"
      value: 1200.0
      target: 1000.0
      status: "pass"  # pass, fail, warn
      notes: "Exceeded target performance"
```

## Output Formats

### Markdown Format
- **Use Case**: GitHub releases, documentation
- **Features**: Human-readable, emoji status indicators, tables
- **Template**: `templates/markdown.md.j2`

### HTML Format
- **Use Case**: Web documentation, customer deliverables
- **Features**: Styled presentation, responsive design, interactive elements
- **Template**: `templates/html.html.j2`

### JSON Format
- **Use Case**: API integration, programmatic access
- **Features**: Structured data, version metadata, API wrapper
- **Template**: `templates/json.json.j2`

## Validation Rules

The system enforces both schema validation and business rules:

### Schema Validation
- JSON Schema compliance
- Required field presence
- Data type validation
- Enum value checking

### Business Rules
- Test count consistency (total = passed + failed + skipped)
- Protocol status alignment with testing evidence
- Performance target vs. actual value consistency
- Version format compliance (semantic versioning)
- Coverage percentage ranges (0-100%)

## Best Practices

### Data Collection
1. **Run comprehensive tests** before generating release cards
2. **Include performance benchmarks** for all supported protocols
3. **Document virtual devices** with clear naming conventions
4. **Test real hardware** when available and document results
5. **Maintain device registry** with current firmware versions

### Release Process
1. **Validate data** before generation: `python tools/validate.py --strict`
2. **Review generated cards** for accuracy and completeness
3. **Update known issues** and workarounds as needed
4. **Archive previous versions** for historical reference
5. **Publish to multiple channels** (GitHub, docs site, customer portals)

### Schema Evolution
1. **Version the schema** using semantic versioning
2. **Maintain backward compatibility** when possible
3. **Document schema changes** in release notes
4. **Validate against multiple schema versions** during transitions
5. **Provide migration tools** for breaking changes

## Troubleshooting

### Common Issues

**Validation Errors:**
- Check YAML syntax with `yamllint`
- Verify all required fields are present
- Ensure test counts add up correctly
- Validate version format matches semantic versioning

**Generation Failures:**
- Install required dependencies: `pip install pyyaml jinja2 jsonschema`
- Check template file paths are correct
- Verify Jinja2 template syntax
- Ensure output directory is writable

**CI/CD Integration:**
- Check GitHub Actions workflow permissions
- Verify artifact upload/download steps
- Ensure test results are in expected format
- Check environment variable availability

### Debug Mode

Enable verbose output:
```bash
python tools/validate.py --verbose release-data.yaml
python tools/generate.py --debug release-data.yaml
```

### Logging

The tools provide structured logging:
- ✅ Success operations
- ⚠️ Warnings and non-critical issues  
- ❌ Errors and failures
- 📄 File operations
- 🎉 Completion messages

## Future Enhancements

### Phase 2 Features
- **Real Hardware Integration**: Automated testing with lab equipment
- **Performance Trending**: Historical performance comparison
- **Security Scanning**: Automated security assessment integration
- **Customer Dashboards**: Self-service compatibility checking

### Phase 3 Features
- **Multi-vendor Testing**: Expanded device compatibility matrix
- **Certification Tracking**: Standards compliance documentation
- **API Versioning**: RESTful API for programmatic access
- **Notification System**: Automated alerts for compatibility changes

## Support

For issues with the release card system:

1. **Check Documentation**: Review this guide and schema definitions
2. **Validate Data**: Use `validate.py` to identify data issues
3. **Check Logs**: Review CI/CD workflow logs for errors
4. **Open Issues**: Create GitHub issues with example data and error messages
5. **Contact Team**: Reach out to the development team for complex issues