#!/usr/bin/env python3
"""
Release Card Generator for Bifrost Gateway

This script generates release cards documenting tested fieldbus protocols,
device compatibility, performance metrics, and testing coverage.
"""

import argparse
import json
import os
import sys
from datetime import datetime
from pathlib import Path
from typing import Dict, List, Any, Optional

import yaml


class ReleaseCardGenerator:
    """Generates release cards from test results and performance data."""
    
    def __init__(self, output_dir: str = "release-cards"):
        self.output_dir = Path(output_dir)
        self.output_dir.mkdir(exist_ok=True)
        self.data = {
            "version": "",
            "release_date": datetime.now().isoformat(),
            "release_type": "alpha",
            "protocols": {},
            "performance": {},
            "testing": {},
            "quality_gates": {}
        }
    
    def load_test_results(self, results_dir: str) -> None:
        """Load test results from test execution."""
        results_path = Path(results_dir)
        
        if not results_path.exists():
            print(f"Warning: Test results directory {results_dir} not found")
            return
            
        # Load virtual device test results
        virtual_tests_file = results_path / "virtual-device-tests.json"
        if virtual_tests_file.exists():
            with open(virtual_tests_file) as f:
                virtual_tests = json.load(f)
                self._process_virtual_device_tests(virtual_tests)
        
        # Load performance benchmark results
        benchmark_file = results_path / "benchmark-results.json"
        if benchmark_file.exists():
            with open(benchmark_file) as f:
                benchmarks = json.load(f)
                self._process_performance_benchmarks(benchmarks)
        
        # Load Go gateway test results
        go_tests_file = results_path / "go-test-results.json"
        if go_tests_file.exists():
            with open(go_tests_file) as f:
                go_tests = json.load(f)
                self._process_go_tests(go_tests)
    
    def _process_virtual_device_tests(self, tests: Dict[str, Any]) -> None:
        """Process virtual device test results."""
        self.data["testing"]["virtual_devices"] = {
            "total_tests": tests.get("total", 0),
            "passed": tests.get("passed", 0),
            "failed": tests.get("failed", 0),
            "protocols_tested": tests.get("protocols", []),
            "coverage": tests.get("coverage", {})
        }
    
    def _process_performance_benchmarks(self, benchmarks: Dict[str, Any]) -> None:
        """Process performance benchmark results."""
        self.data["performance"] = {
            "throughput": {
                "ops_per_second": benchmarks.get("throughput", {}).get("ops_per_sec", 0),
                "target_achieved": benchmarks.get("throughput", {}).get("target_achieved", False)
            },
            "latency": {
                "average_ms": benchmarks.get("latency", {}).get("average_ms", 0),
                "p95_ms": benchmarks.get("latency", {}).get("p95_ms", 0),
                "target_achieved": benchmarks.get("latency", {}).get("target_achieved", False)
            },
            "memory": {
                "peak_mb": benchmarks.get("memory", {}).get("peak_mb", 0),
                "target_achieved": benchmarks.get("memory", {}).get("target_achieved", False)
            },
            "overall_score": benchmarks.get("overall_score", 0)
        }
    
    def _process_go_tests(self, go_tests: Dict[str, Any]) -> None:
        """Process Go gateway test results."""
        self.data["testing"]["go_gateway"] = {
            "total_tests": go_tests.get("total", 0),
            "passed": go_tests.get("passed", 0),
            "failed": go_tests.get("failed", 0),
            "coverage_percent": go_tests.get("coverage", 0)
        }
    
    def set_version_info(self, version: str, release_type: str = "alpha") -> None:
        """Set release version and type."""
        self.data["version"] = version
        self.data["release_type"] = release_type
    
    def add_protocol_info(self, protocol: str, status: str, version: str, 
                         limitations: List[str] = None) -> None:
        """Add protocol support information."""
        self.data["protocols"][protocol] = {
            "status": status,
            "version": version,
            "limitations": limitations or [],
            "tested_devices": {
                "virtual": [],
                "real": []
            }
        }
    
    def evaluate_quality_gates(self) -> Dict[str, bool]:
        """Evaluate quality gates for release approval."""
        gates = {}
        
        # Test coverage gate
        virtual_tests = self.data["testing"].get("virtual_devices", {})
        go_tests = self.data["testing"].get("go_gateway", {})
        
        gates["test_coverage"] = (
            virtual_tests.get("passed", 0) > 0 and
            go_tests.get("coverage_percent", 0) >= 70
        )
        
        # Performance gates
        perf = self.data["performance"]
        gates["performance_targets"] = (
            perf.get("throughput", {}).get("target_achieved", False) and
            perf.get("latency", {}).get("target_achieved", False)
        )
        
        # Documentation completeness
        gates["documentation_complete"] = bool(
            self.data["protocols"] and
            self.data["version"]
        )
        
        # Overall approval
        gates["approved_for_release"] = all([
            gates["test_coverage"],
            gates["performance_targets"],
            gates["documentation_complete"]
        ])
        
        self.data["quality_gates"] = gates
        return gates
    
    def generate_markdown(self) -> str:
        """Generate Markdown release card."""
        md = f"""# Release Card: Bifrost Gateway {self.data['version']}

**Release Date:** {self.data['release_date'][:10]}  
**Release Type:** {self.data['release_type'].upper()}

## 🚀 Protocol Support

"""
        
        for protocol, info in self.data["protocols"].items():
            status_emoji = "✅" if info["status"] == "stable" else "🚧"
            md += f"### {protocol.upper()} {status_emoji}\n"
            md += f"- **Status:** {info['status']}\n"
            md += f"- **Version:** {info['version']}\n"
            if info["limitations"]:
                md += f"- **Limitations:** {', '.join(info['limitations'])}\n"
            md += "\n"
        
        md += f"""## 📊 Performance Metrics

- **Throughput:** {self.data['performance'].get('throughput', {}).get('ops_per_second', 0):,} ops/sec
- **Latency (P95):** {self.data['performance'].get('latency', {}).get('p95_ms', 0)}ms
- **Memory Usage:** {self.data['performance'].get('memory', {}).get('peak_mb', 0)}MB
- **Overall Score:** {self.data['performance'].get('overall_score', 0)}/100

## 🧪 Testing Coverage

### Virtual Device Tests
- **Total Tests:** {self.data['testing'].get('virtual_devices', {}).get('total_tests', 0)}
- **Passed:** {self.data['testing'].get('virtual_devices', {}).get('passed', 0)}
- **Failed:** {self.data['testing'].get('virtual_devices', {}).get('failed', 0)}

### Go Gateway Tests  
- **Total Tests:** {self.data['testing'].get('go_gateway', {}).get('total_tests', 0)}
- **Coverage:** {self.data['testing'].get('go_gateway', {}).get('coverage_percent', 0)}%

## ✅ Quality Gates

"""
        
        gates = self.data["quality_gates"]
        for gate, passed in gates.items():
            emoji = "✅" if passed else "❌"
            gate_name = gate.replace("_", " ").title()
            md += f"- **{gate_name}:** {emoji}\n"
        
        md += f"""
## 📋 Installation

```bash
# Download and install
wget https://github.com/IamMikeHelsel/bifrost/releases/download/{self.data['version']}/bifrost-gateway-linux-amd64
chmod +x bifrost-gateway-linux-amd64
./bifrost-gateway-linux-amd64
```

## 🔗 Resources

- [Documentation](https://github.com/IamMikeHelsel/bifrost/blob/main/README.md)
- [Performance Details](https://github.com/IamMikeHelsel/bifrost/blob/main/go-gateway/PERFORMANCE_OPTIMIZATIONS.md)
- [Production Deployment](https://github.com/IamMikeHelsel/bifrost/blob/main/go-gateway/docs/runbooks/production-deployment.md)
"""
        
        return md
    
    def generate_json(self) -> str:
        """Generate JSON release card for API consumption."""
        return json.dumps(self.data, indent=2)
    
    def generate_yaml(self) -> str:
        """Generate YAML release card."""
        return yaml.dump(self.data, default_flow_style=False, sort_keys=False)
    
    def save_release_cards(self, filename_base: str) -> List[str]:
        """Save release cards in multiple formats."""
        files_created = []
        
        # Markdown
        md_file = self.output_dir / f"{filename_base}.md"
        with open(md_file, 'w') as f:
            f.write(self.generate_markdown())
        files_created.append(str(md_file))
        
        # JSON
        json_file = self.output_dir / f"{filename_base}.json"
        with open(json_file, 'w') as f:
            f.write(self.generate_json())
        files_created.append(str(json_file))
        
        # YAML
        yaml_file = self.output_dir / f"{filename_base}.yaml"
        with open(yaml_file, 'w') as f:
            f.write(self.generate_yaml())
        files_created.append(str(yaml_file))
        
        return files_created


def main():
    """Main function for command-line usage."""
    parser = argparse.ArgumentParser(description="Generate Bifrost release cards")
    parser.add_argument("--version", required=True, help="Release version (e.g., v0.1.0)")
    parser.add_argument("--release-type", default="alpha", choices=["alpha", "beta", "rc", "stable"])
    parser.add_argument("--test-results", default="test-results", help="Test results directory")
    parser.add_argument("--output-dir", default="release-cards", help="Output directory")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    
    args = parser.parse_args()
    
    if args.verbose:
        print(f"🚀 Generating release card for {args.version}")
    
    # Create generator
    generator = ReleaseCardGenerator(args.output_dir)
    
    # Set version info
    generator.set_version_info(args.version, args.release_type)
    
    # Load test results
    generator.load_test_results(args.test_results)
    
    # Add default protocol information (this would typically come from test results)
    generator.add_protocol_info("modbus", "stable", "1.1b3", ["No RTU over TCP support"])
    generator.add_protocol_info("opcua", "experimental", "0.1.0", ["Limited security features"])
    generator.add_protocol_info("ethernetip", "experimental", "0.1.0", ["Basic read/write only"])
    
    # Evaluate quality gates
    gates = generator.evaluate_quality_gates()
    
    if args.verbose:
        print(f"📊 Quality Gates: {sum(gates.values())}/{len(gates)} passed")
    
    # Generate and save release cards
    filename_base = f"release-card-{args.version.replace('v', '')}"
    files_created = generator.save_release_cards(filename_base)
    
    print(f"✅ Release card generated successfully!")
    for file_path in files_created:
        print(f"   📄 {file_path}")
    
    # Exit with non-zero if not approved for release
    if not gates.get("approved_for_release", False):
        print("❌ Release not approved - quality gates failed")
        sys.exit(1)
    
    print("🎉 Release approved for deployment!")


if __name__ == "__main__":
    main()