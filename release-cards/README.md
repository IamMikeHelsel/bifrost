# Release Card System

This directory contains the Release Card System implementation for documenting tested fieldbus and endpoint compatibility for each software release.

## Directory Structure

```
release-cards/
├── README.md                   # This file
├── schema/                     # Schema definitions
│   ├── release-card.schema.json    # JSON Schema for release cards
│   └── release-card.schema.yaml    # YAML Schema definition
├── templates/                  # Output format templates
│   ├── markdown.md.j2              # Markdown template
│   ├── html.html.j2                # HTML template
│   └── json.json.j2                # JSON API template
├── examples/                   # Example release cards
│   ├── v0.1.0.yaml                 # Example release card
│   └── v0.1.0.md                   # Generated markdown
└── tools/                      # Generation and validation tools
    ├── generate.py                 # Main generation script
    ├── validate.py                 # Schema validation
    └── collect.py                  # Data collection utilities
```

## Usage

1. **Create Release Card Data**: Define release information in YAML format
2. **Generate Documentation**: Use tools to generate multiple output formats
3. **Validate**: Ensure data conforms to schema
4. **Publish**: Integrate with CI/CD for automated generation

## Features

- **Protocol Support Matrix**: Documents tested industrial protocols
- **Device Compatibility**: Tracks virtual and real hardware testing
- **Performance Metrics**: Includes throughput and latency benchmarks
- **Multiple Formats**: Supports Markdown, HTML, JSON, and PDF output
- **Automated Generation**: Integrates with CI/CD pipelines
- **Schema Validation**: Ensures data consistency and completeness