#!/bin/bash

# Create mock test results for testing release card generation

echo "🧪 Creating mock test results for release card testing..."

mkdir -p test-results

# Create virtual device test results
cat > test-results/virtual-device-tests.json << 'EOF'
{
  "total": 25,
  "passed": 22,
  "failed": 3,
  "protocols": ["modbus", "opcua", "ethernetip"],
  "coverage": {
    "modbus": 85,
    "opcua": 60,
    "ethernetip": 40
  }
}
EOF

# Create Go test results
cat > test-results/go-test-results.json << 'EOF'
{
  "total": 45,
  "passed": 42,
  "failed": 3,
  "coverage": 78
}
EOF

# Create benchmark results
cat > test-results/benchmark-results.json << 'EOF'
{
  "throughput": {
    "ops_per_sec": 18500,
    "target_achieved": true
  },
  "latency": {
    "average_ms": 0.65,
    "p95_ms": 1.2,
    "target_achieved": true
  },
  "memory": {
    "peak_mb": 42,
    "target_achieved": true
  },
  "overall_score": 88
}
EOF

echo "✅ Mock test results created in test-results/"
echo "🚀 You can now test with: python tools/generate-release-card.py --version v0.1.0 --test-results test-results --verbose"