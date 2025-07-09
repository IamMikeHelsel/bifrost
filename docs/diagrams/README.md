# Bifrost Diagrams

This directory contains all diagram files for the Bifrost project documentation using our standardized diagram toolchain.

## 🚀 Quick Start

**New to diagramming?** Start with **[QUICK_START.md](./QUICK_START.md)** for a step-by-step guide.

**Need standards?** Check **[STANDARDS.md](./STANDARDS.md)** for detailed guidelines and best practices.

## 📁 Directory Structure

```
docs/diagrams/
├── README.md (this file)
├── QUICK_START.md          # Quick start guide for new users
├── STANDARDS.md            # Detailed standards and guidelines
├── templates/              # Reusable diagram templates
│   ├── mermaid-template.md # Mermaid examples and patterns
│   ├── component-template.puml
│   ├── sequence-template.puml
│   └── deployment-template.puml
├── architecture/           # System and component diagrams
│   ├── system-overview.puml
│   ├── gateway-components.puml
│   └── protocol-architecture.puml
├── sequences/              # API interactions and protocol flows
│   ├── modbus-read-sequence.puml
│   ├── websocket-streaming.puml
│   └── device-discovery.puml
├── deployment/             # Infrastructure and deployment
│   ├── production-deployment.puml
│   ├── cloud-integration.puml
│   └── edge-deployment.puml
└── exports/                # Generated files (SVG, PNG, PDF)
    ├── svg/               # Vector format (recommended)
    ├── png/               # Raster format
    └── pdf/               # Document format
```

## 🛠️ Toolchain

### Primary Tools (Production Ready)

| Tool | Use Case | File Type | Integration |
|------|----------|-----------|-------------|
| **Mermaid** | System overviews, flowcharts | `.md` (inline) | Native GitHub rendering |
| **PlantUML** | Detailed technical diagrams | `.puml` | VS Code extension + export |

### VS Code Extensions (Installed)

- **PlantUML** (`jebbs.plantuml`) - Detailed technical diagrams
- **Markdown Mermaid** (`bierner.markdown-mermaid`) - Inline diagram support  
- **Markdown Preview Enhanced** (`shd101wyy.markdown-preview-enhanced`) - Enhanced preview
- **Draw.io Integration** (`hediet.vscode-drawio`) - Complex diagrams when needed

## 🎯 Usage Guidelines

### When to Use Each Tool

**Use Mermaid for:**
- High-level system overviews
- Simple flowcharts
- Architecture summaries
- README diagrams

**Use PlantUML for:**
- Detailed API sequences
- Component architecture
- Deployment diagrams
- Protocol specifications

### File Naming Conventions

- **kebab-case**: `modbus-read-sequence.puml`
- **descriptive**: `production-deployment.puml` not `deploy.puml`
- **type-specific**: `gateway-components.puml`, `api-sequence.puml`

## 📈 Current Diagram Inventory

### Architecture Diagrams
- **[System Overview](./architecture/system-overview.puml)** - Complete system architecture
- **[Gateway Components](./architecture/gateway-components.puml)** - Detailed Go gateway structure
- **[Protocol Architecture](./architecture/protocol-architecture.puml)** - Protocol handler design

### Sequence Diagrams  
- **[Modbus Read Sequence](./sequences/modbus-read-sequence.puml)** - Complete Modbus operation flow
- **[WebSocket Streaming](./sequences/websocket-streaming.puml)** - Real-time data streaming
- **[Device Discovery](./sequences/device-discovery.puml)** - Automatic device detection

### Deployment Diagrams
- **[Production Deployment](./deployment/production-deployment.puml)** - Production environment setup
- **[Cloud Integration](./deployment/cloud-integration.puml)** - Hybrid edge-cloud architecture
- **[Edge Deployment](./deployment/edge-deployment.puml)** - Industrial gateway setup

## 🔄 Development Workflow

### Creating New Diagrams

1. **Choose template**: Copy from `templates/` directory
2. **Edit content**: Replace template with your actual content
3. **Preview**: Use VS Code PlantUML/Mermaid extensions
4. **Export**: Generate SVG files to `exports/` directory
5. **Document**: Reference in relevant markdown files
6. **Commit**: Include both source and exported files

### Example Workflow

```bash
# 1. Copy template
cp docs/diagrams/templates/sequence-template.puml docs/diagrams/sequences/new-api-flow.puml

# 2. Edit in VS Code
code docs/diagrams/sequences/new-api-flow.puml

# 3. Preview with Ctrl+Shift+P → "PlantUML: Preview Current Diagram"

# 4. Export with Ctrl+Shift+P → "PlantUML: Export Current Diagram"

# 5. Reference in documentation
echo "![API Flow](./diagrams/exports/svg/new-api-flow.svg)" >> relevant-doc.md

# 6. Commit changes
git add docs/diagrams/
git commit -m "Add new API flow sequence diagram"
```

## 🌟 Benefits Achieved

- **Standardized toolchain** across all team members
- **GitHub native rendering** for Mermaid diagrams
- **Professional quality** outputs for client presentations
- **Version control friendly** text-based diagram sources
- **VS Code integration** for seamless development workflow
- **Template system** for consistent diagram styles
- **Performance annotations** integrated into technical diagrams

## 🤝 Contributing

1. **Follow standards**: See [STANDARDS.md](./STANDARDS.md) for detailed guidelines
2. **Use templates**: Start with proven templates in `templates/`
3. **Include metrics**: Add performance data where relevant
4. **Export consistently**: Always generate SVG exports
5. **Update documentation**: Link to new diagrams in relevant docs

## 📚 Resources

- **[Quick Start Guide](./QUICK_START.md)** - Get started in 5 minutes
- **[Standards Document](./STANDARDS.md)** - Comprehensive guidelines
- **[Mermaid Documentation](https://mermaid.js.org/)** - Official Mermaid docs
- **[PlantUML Documentation](https://plantuml.com/)** - Official PlantUML docs
- **[Template Examples](./templates/)** - Ready-to-use templates

---

**Next Steps**: Follow the [Quick Start Guide](./QUICK_START.md) to create your first diagram using our standardized toolchain!