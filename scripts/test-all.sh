#!/bin/bash
# Complete test runner - runs all tests including slow ones

set -e

echo "🔬 Running all tests (including slow tests)..."

# Run all tests
echo "📦 Testing all packages..."
.venv1/bin/python -m pytest packages/ \
  -m "not benchmark" \
  --tb=line \
  --durations=10 \
  -x

echo "✅ All tests completed!"