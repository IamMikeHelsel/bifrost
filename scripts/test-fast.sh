#!/bin/bash
# Fast test runner - runs only the fastest tests in under 750ms

set -e

echo "🚀 Running fast tests (target: <750ms)..."

# Run core tests (should be ~170ms)
echo "📦 Testing bifrost-core..."
.venv1/bin/python -m pytest packages/bifrost-core/tests/ --tb=line -q

# Run fast bifrost tests (should be ~170ms)  
echo "📦 Testing bifrost (fast subset)..."
.venv1/bin/python -m pytest \
  packages/bifrost/tests/test_cli.py \
  packages/bifrost/tests/test_cli_simple.py \
  packages/bifrost/tests/test_discovery.py \
  packages/bifrost/tests/test_discovery_new.py \
  packages/bifrost/tests/test_init.py \
  --tb=line -q

echo "✅ Fast tests completed!"