# Diagram Creation Quick Start Guide

This guide will help you quickly create professional diagrams for the Bifrost project using our standardized toolchain.

## 🚀 Setup (One-time)

### 1. Install VS Code Extensions

Open VS Code and install these extensions:

- **PlantUML** (`jebbs.plantuml`) - For detailed technical diagrams
- **Markdown Mermaid** (`bierner.markdown-mermaid`) - For inline diagrams
- **Markdown Preview Enhanced** (`shd101wyy.markdown-preview-enhanced`) - For preview
- **Draw.io Integration** (`hediet.vscode-drawio`) - For complex diagrams

### 2. Verify Setup

1. Open any `.puml` file in `docs/diagrams/templates/`
2. Press `Ctrl+Shift+P` → "PlantUML: Preview Current Diagram"
3. You should see a rendered diagram preview

## 📋 Quick Reference

### When to Use Each Tool

| Use Case | Tool | File Type | Location |
|----------|------|-----------|----------|
| System overviews | **Mermaid** | `.md` (inline) | Inline in markdown |
| API sequences | **PlantUML** | `.puml` | `docs/diagrams/sequences/` |
| Component details | **PlantUML** | `.puml` | `docs/diagrams/architecture/` |
| Deployment architecture | **PlantUML** | `.puml` | `docs/diagrams/deployment/` |

### File Naming Convention

- Use **kebab-case**: `modbus-read-sequence.puml`
- Be **descriptive**: `production-deployment.puml` not `deploy.puml`
- Include **diagram type**: `gateway-components.puml`, `api-sequence.puml`

## 🎨 Creating Your First Diagram

### Option 1: Mermaid (For Simple Diagrams)

1. **Add directly to markdown**:
   ```markdown
   ```mermaid
   graph LR
       A[Component A] --> B[Component B]
       B --> C[Component C]
   ```
   ```

2. **Use our template**: Copy from `docs/diagrams/templates/mermaid-template.md`

### Option 2: PlantUML (For Detailed Diagrams)

1. **Copy a template**:
   ```bash
   cp docs/diagrams/templates/sequence-template.puml docs/diagrams/sequences/my-new-sequence.puml
   ```

2. **Edit in VS Code**:
   - Replace template content with your actual flow
   - Use `Ctrl+Shift+P` → "PlantUML: Preview Current Diagram" to see live preview

3. **Export for documentation**:
   - `Ctrl+Shift+P` → "PlantUML: Export Current Diagram"
   - Choose SVG format (recommended)
   - File saves to `docs/diagrams/exports/svg/`

## 📖 Common Patterns

### Performance Annotations

Always include performance metrics where relevant:

```plantuml
note right of gateway
  **Performance Metrics**
  - 18,879 ops/sec throughput
  - 53µs average latency
  - < 50MB memory footprint
end note
```

### Component Styling

Use consistent colors and themes:

```plantuml
@startuml
!theme blueprint

' Your diagram content here

@enduml
```

### Error Handling Flows

Include error scenarios in sequences:

```plantuml
group Error Handling
    User -> System : Invalid Request
    System -> User : Error Response
end
```

## 🔄 Workflow Integration

### 1. Create Diagram

- Copy appropriate template
- Edit with your content
- Preview in VS Code
- Export to SVG

### 2. Use in Documentation

```markdown
![My Diagram](./diagrams/exports/svg/my-diagram.svg)
```

### 3. Commit Changes

```bash
git add docs/diagrams/
git commit -m "Add new sequence diagram for feature X"
```

## 🏗️ Templates Available

### Ready-to-Use Templates

- **`component-template.puml`** - Component architecture diagrams
- **`sequence-template.puml`** - API and protocol sequences
- **`deployment-template.puml`** - Infrastructure and deployment
- **`mermaid-template.md`** - System overviews and flowcharts

### Example Usage

```bash
# Create a new component diagram
cp docs/diagrams/templates/component-template.puml docs/diagrams/architecture/auth-components.puml

# Edit the file
code docs/diagrams/architecture/auth-components.puml

# Preview and export
# Use VS Code PlantUML extension
```

## 🎯 Best Practices

### Content Guidelines

1. **Clear titles**: Always include a descriptive title
2. **Consistent naming**: Use same component names across diagrams
3. **Performance data**: Include metrics when relevant
4. **Error scenarios**: Show error handling flows
5. **Helpful notes**: Add explanatory notes for complex parts

### Visual Guidelines

1. **Use our theme**: Always start with `!theme blueprint`
2. **Consistent colors**: Follow the established color scheme
3. **Readable fonts**: Ensure text is legible at normal size
4. **Proper spacing**: Don't overcrowd diagrams

### File Organization

1. **Use correct directories**: Follow the structure in `docs/diagrams/`
2. **Export SVG files**: For best quality and GitHub compatibility
3. **Update references**: Link to new diagrams in relevant documentation
4. **Version control**: Commit both source `.puml` and exported files

## 🛠️ Troubleshooting

### Common Issues

**PlantUML not rendering:**
- Check internet connection (uses online server)
- Verify file syntax with `@startuml` and `@enduml`
- Try local PlantUML server if online fails

**Mermaid not showing:**
- Check markdown syntax has proper code block
- Verify mermaid extension is installed
- Try refreshing markdown preview

**Export not working:**
- Ensure output directory exists: `docs/diagrams/exports/svg/`
- Check file permissions
- Try manual export command

### Getting Help

1. **Check existing diagrams**: Browse `docs/diagrams/` for examples
2. **Use templates**: Start with our proven templates
3. **Team review**: Ask for feedback on complex diagrams
4. **Documentation**: Check PlantUML and Mermaid official docs

## 🌟 Next Steps

1. **Practice**: Create a simple diagram using a template
2. **Explore**: Look at existing diagrams in `docs/diagrams/`
3. **Contribute**: Add new diagrams for features you're working on
4. **Share**: Help teammates adopt the new toolchain

---

**Remember**: Good diagrams are worth a thousand words. Take time to create clear, accurate, and helpful visualizations!

For questions or improvements to this guide, reach out to the development team.