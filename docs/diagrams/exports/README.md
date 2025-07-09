# Diagram Exports

This directory contains exported versions of PlantUML and Mermaid diagrams in various formats.

## Directory Structure

- `svg/` - SVG exports (vector format, recommended for web)
- `png/` - PNG exports (raster format, good for presentations)  
- `pdf/` - PDF exports (document format, good for printing)

## Usage

### Automatic Export with VS Code

With the PlantUML extension installed:

1. Open any `.puml` file in VS Code
2. Press `Ctrl+Shift+P` and select "PlantUML: Export Current Diagram"
3. Choose your desired format (SVG recommended)
4. Files will be exported to the appropriate subdirectory

### Manual Export

```bash
# Export all PlantUML diagrams to SVG
find docs/diagrams -name "*.puml" -exec plantuml -tsvg -o exports/svg {} \;

# Export specific diagram
plantuml -tsvg -o exports/svg docs/diagrams/architecture/gateway-components.puml
```

### Export Configuration

The VS Code workspace is configured to:
- Export to SVG format by default (best quality and GitHub compatibility)
- Save exports in this directory structure
- Use the online PlantUML server for rendering

## Git Handling

- SVG files should be committed to version control (small, text-based)
- PNG and PDF files are typically excluded via `.gitignore` (large binary files)
- Exports are regenerated as needed from source `.puml` files

## Integration with Documentation

Reference exported diagrams in markdown:

```markdown
![Gateway Architecture](./diagrams/exports/svg/gateway-components.svg)
```

Note: GitHub renders SVG files natively, making them ideal for documentation.