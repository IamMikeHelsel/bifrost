#!/bin/bash

# Format all markdown files in the project
echo "Formatting markdown files..."

# Format all .md files
find . -name "*.md" -not -path "./.*" -exec mdformat {} \;

echo "✅ All markdown files formatted!"
echo ""
echo "To run this script:"
echo "  chmod +x format-docs.sh"
echo "  ./format-docs.sh"