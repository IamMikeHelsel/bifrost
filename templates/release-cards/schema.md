# Release Card Schema

This document defines the schema for Bifrost Gateway release cards.

## JSON Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "version": {
      "type": "string",
      "description": "Release version (e.g., v0.1.0)"
    },
    "release_date": {
      "type": "string",
      "format": "date-time",
      "description": "Release date in ISO format"
    },
    "release_type": {
      "type": "string",
      "enum": ["alpha", "beta", "rc", "stable"],
      "description": "Release type"
    },
    "protocols": {
      "type": "object",
      "description": "Protocol support information",
      "patternProperties": {
        "^[a-z]+$": {
          "type": "object",
          "properties": {
            "status": {
              "type": "string",
              "enum": ["stable", "experimental", "deprecated"]
            },
            "version": {
              "type": "string"
            },
            "limitations": {
              "type": "array",
              "items": {"type": "string"}
            },
            "tested_devices": {
              "type": "object",
              "properties": {
                "virtual": {"type": "array", "items": {"type": "string"}},
                "real": {"type": "array", "items": {"type": "string"}}
              }
            }
          }
        }
      }
    },
    "performance": {
      "type": "object",
      "properties": {
        "throughput": {
          "type": "object",
          "properties": {
            "ops_per_second": {"type": "number"},
            "target_achieved": {"type": "boolean"}
          }
        },
        "latency": {
          "type": "object",
          "properties": {
            "average_ms": {"type": "number"},
            "p95_ms": {"type": "number"},
            "target_achieved": {"type": "boolean"}
          }
        },
        "memory": {
          "type": "object",
          "properties": {
            "peak_mb": {"type": "number"},
            "target_achieved": {"type": "boolean"}
          }
        },
        "overall_score": {
          "type": "number",
          "minimum": 0,
          "maximum": 100
        }
      }
    },
    "testing": {
      "type": "object",
      "properties": {
        "virtual_devices": {
          "type": "object",
          "properties": {
            "total_tests": {"type": "number"},
            "passed": {"type": "number"},
            "failed": {"type": "number"},
            "protocols_tested": {"type": "array", "items": {"type": "string"}},
            "coverage": {"type": "object"}
          }
        },
        "go_gateway": {
          "type": "object",
          "properties": {
            "total_tests": {"type": "number"},
            "passed": {"type": "number"},
            "failed": {"type": "number"},
            "coverage_percent": {"type": "number"}
          }
        }
      }
    },
    "quality_gates": {
      "type": "object",
      "properties": {
        "test_coverage": {"type": "boolean"},
        "performance_targets": {"type": "boolean"},
        "documentation_complete": {"type": "boolean"},
        "approved_for_release": {"type": "boolean"}
      }
    }
  },
  "required": ["version", "release_date", "release_type", "protocols", "performance", "testing", "quality_gates"]
}
```

## YAML Example

```yaml
version: "v0.1.0"
release_date: "2024-12-01T10:00:00Z"
release_type: "alpha"

protocols:
  modbus:
    status: "stable"
    version: "1.1b3"
    limitations:
      - "No RTU over TCP support"
    tested_devices:
      virtual:
        - "modbus_tcp_simulator_v1.0"
      real: []
  
  opcua:
    status: "experimental"
    version: "0.1.0"
    limitations:
      - "Limited security features"
    tested_devices:
      virtual:
        - "opcua_simulator_v1.0"
      real: []

performance:
  throughput:
    ops_per_second: 18500
    target_achieved: true
  latency:
    average_ms: 0.65
    p95_ms: 1.2
    target_achieved: true
  memory:
    peak_mb: 42
    target_achieved: true
  overall_score: 88

testing:
  virtual_devices:
    total_tests: 25
    passed: 22
    failed: 3
    protocols_tested: ["modbus", "opcua", "ethernetip"]
    coverage:
      modbus: 85
      opcua: 60
      ethernetip: 40
  
  go_gateway:
    total_tests: 45
    passed: 42
    failed: 3
    coverage_percent: 78

quality_gates:
  test_coverage: true
  performance_targets: true
  documentation_complete: true
  approved_for_release: true
```

## Performance Targets

The following performance targets are used for quality gate evaluation:

- **Throughput**: ≥ 10,000 operations per second
- **Latency (P95)**: ≤ 2.0 milliseconds
- **Memory Usage**: ≤ 50 MB peak
- **Overall Score**: ≥ 80/100

## Test Coverage Requirements

- **Virtual Device Tests**: Must have > 0 passing tests
- **Go Gateway Tests**: Must have ≥ 70% code coverage
- **Protocol Coverage**: At least one protocol must be tested

## Release Types

- **alpha**: Early development, experimental features
- **beta**: Feature-complete, testing phase
- **rc**: Release candidate, production-ready
- **stable**: Production release, fully tested