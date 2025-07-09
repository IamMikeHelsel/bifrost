#!/bin/bash

# Integration test for release card CI/CD workflow

set -e

echo "🧪 Running release card CI/CD integration test..."

# Clean up any existing test files
rm -rf test-results release-cards _site

# Create mock test results
echo "📊 Creating mock test results..."
./scripts/create-mock-test-results.sh

# Test release card generation
echo "🚀 Testing release card generation..."
python tools/generate-release-card.py --version v0.1.0-test --test-results test-results --verbose

# Verify release card files were created
if [ ! -f "release-cards/release-card-0.1.0-test.md" ]; then
    echo "❌ Release card markdown not created"
    exit 1
fi

if [ ! -f "release-cards/release-card-0.1.0-test.json" ]; then
    echo "❌ Release card JSON not created"
    exit 1
fi

if [ ! -f "release-cards/release-card-0.1.0-test.yaml" ]; then
    echo "❌ Release card YAML not created"
    exit 1
fi

# Test documentation deployment
echo "🌐 Testing documentation deployment..."
python tools/deploy-docs.py --dry-run --verbose

# Verify documentation site was created
if [ ! -d "_site" ]; then
    echo "❌ Documentation site not created"
    exit 1
fi

if [ ! -f "_site/index.html" ]; then
    echo "❌ Documentation index not created"
    exit 1
fi

if [ ! -f "_site/release-cards/index.html" ]; then
    echo "❌ Release cards index not created"
    exit 1
fi

# Verify release card is in the site
if [ ! -f "_site/release-cards/release-card-0.1.0-test.md" ]; then
    echo "❌ Release card not deployed to site"
    exit 1
fi

# Test quality gates validation
echo "🔍 Testing quality gates..."
python3 << 'EOF'
import json

# Load the generated release card
with open('release-cards/release-card-0.1.0-test.json') as f:
    card = json.load(f)

# Check quality gates
gates = card.get('quality_gates', {})
required_gates = ['test_coverage', 'performance_targets', 'documentation_complete', 'approved_for_release']

for gate in required_gates:
    if not gates.get(gate, False):
        print(f"❌ Quality gate '{gate}' failed")
        exit(1)

print("✅ All quality gates passed")
EOF

echo "✅ Integration test completed successfully!"
echo ""
echo "📋 Summary:"
echo "  🚀 Release card generation: ✅"
echo "  📊 Quality gates validation: ✅"
echo "  🌐 Documentation deployment: ✅"
echo "  🔍 File integrity checks: ✅"
echo ""
echo "🎉 CI/CD integration for release cards is working correctly!"