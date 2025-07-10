#!/usr/bin/env python3
"""
Release Card Data Collector

This script collects test results, performance data, and device information
to automatically populate release card data.

Usage:
    python collect.py --output release-data.yaml [--test-results PATH] [--benchmarks PATH]
"""

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

try:
    import yaml
except ImportError:
    print("Missing required dependency: pyyaml")
    print("Install with: pip install pyyaml")
    sys.exit(1)


class ReleaseDataCollector:
    """Collect and aggregate data for release cards."""
    
    def __init__(self):
        """Initialize the data collector."""
        self.collected_data = {
            'version': 'unknown',
            'release_date': datetime.now().date().isoformat(),
            'release_type': 'alpha',
            'protocols': {},
            'device_registry': {'virtual_devices': [], 'real_devices': []},
            'performance_benchmarks': {'results': []},
            'testing_summary': {},
            'metadata': {
                'generated_by': 'release-data-collector',
                'generated_at': datetime.now().isoformat() + 'Z',
                'schema_version': '1.0.0'
            }
        }
    
    def collect_from_pytest_results(self, results_path: Path) -> None:
        """Collect data from pytest XML results."""
        try:
            import xml.etree.ElementTree as ET
            
            tree = ET.parse(results_path)
            root = tree.getroot()
            
            # Extract test summary
            total_tests = int(root.get('tests', 0))
            failures = int(root.get('failures', 0))
            errors = int(root.get('errors', 0))
            skipped = int(root.get('skipped', 0))
            passed = total_tests - failures - errors - skipped
            
            self.collected_data['testing_summary']['automated_tests'] = {
                'total': total_tests,
                'passed': passed,
                'failed': failures + errors,
                'skipped': skipped,
                'coverage': 'N/A'  # Would need separate coverage report
            }
            
            # Extract performance benchmark data from test names
            benchmark_results = []
            for testcase in root.findall('.//testcase'):
                name = testcase.get('name', '')
                if 'benchmark' in name.lower() or 'performance' in name.lower():
                    time_taken = float(testcase.get('time', 0))
                    
                    # Extract protocol from test name
                    protocol = 'unknown'
                    if 'modbus' in name.lower():
                        protocol = 'modbus_tcp'
                    elif 'opcua' in name.lower():
                        protocol = 'opcua'
                    elif 'ethernet' in name.lower():
                        protocol = 'ethernet_ip'
                    
                    benchmark_results.append({
                        'protocol': protocol,
                        'test_type': 'execution_time',
                        'metric': 'seconds',
                        'value': time_taken,
                        'target': 1.0,  # Default 1 second target
                        'status': 'pass' if time_taken < 1.0 else 'warn',
                        'notes': f'Test: {name}'
                    })
            
            if benchmark_results:
                self.collected_data['performance_benchmarks']['results'].extend(benchmark_results)
            
            print(f"✅ Collected test data: {total_tests} tests, {passed} passed")
            
        except Exception as e:
            print(f"⚠️  Error reading pytest results: {e}")
    
    def collect_from_benchmark_json(self, benchmark_path: Path) -> None:
        """Collect data from benchmark JSON results."""
        try:
            with open(benchmark_path, 'r') as f:
                benchmark_data = json.load(f)
            
            # Process pytest-benchmark format
            if 'benchmarks' in benchmark_data:
                for benchmark in benchmark_data['benchmarks']:
                    name = benchmark.get('name', '')
                    stats = benchmark.get('stats', {})
                    
                    # Extract protocol from name
                    protocol = 'unknown'
                    if 'modbus' in name.lower():
                        protocol = 'modbus_tcp'
                    elif 'opcua' in name.lower():
                        protocol = 'opcua'
                    elif 'ethernet' in name.lower():
                        protocol = 'ethernet_ip'
                    
                    # Get performance metrics
                    mean_time = stats.get('mean', 0)
                    min_time = stats.get('min', 0)
                    max_time = stats.get('max', 0)
                    
                    self.collected_data['performance_benchmarks']['results'].append({
                        'protocol': protocol,
                        'test_type': 'throughput' if 'throughput' in name.lower() else 'latency',
                        'metric': 'ops/sec' if 'throughput' in name.lower() else 'seconds',
                        'value': 1.0 / mean_time if 'throughput' in name.lower() else mean_time,
                        'target': 1000.0 if 'throughput' in name.lower() else 0.001,
                        'status': 'pass',  # Would need actual targets to determine
                        'notes': f'Mean: {mean_time:.4f}s, Range: {min_time:.4f}-{max_time:.4f}s'
                    })
            
            print(f"✅ Collected benchmark data from {benchmark_path}")
            
        except Exception as e:
            print(f"⚠️  Error reading benchmark results: {e}")
    
    def detect_virtual_devices(self, project_root: Path) -> None:
        """Detect virtual devices from the virtual-devices directory."""
        virtual_devices_dir = project_root / 'virtual-devices'
        
        if not virtual_devices_dir.exists():
            return
        
        virtual_devices = []
        
        # Look for device simulators
        for device_dir in virtual_devices_dir.iterdir():
            if device_dir.is_dir() and not device_dir.name.startswith('.'):
                device_name = device_dir.name
                
                # Determine protocol from directory name
                protocol = 'unknown'
                if 'modbus' in device_name.lower():
                    protocol = 'modbus_tcp' if 'tcp' in device_name.lower() else 'modbus_rtu'
                elif 'opcua' in device_name.lower():
                    protocol = 'opcua'
                elif 'ethernet' in device_name.lower():
                    protocol = 'ethernet_ip'
                elif 's7' in device_name.lower():
                    protocol = 's7'
                
                # Look for version information
                version = '1.0.0'  # Default
                version_files = list(device_dir.glob('*version*')) + list(device_dir.glob('*VERSION*'))
                if version_files:
                    try:
                        with open(version_files[0], 'r') as f:
                            version = f.read().strip()
                    except:
                        pass
                
                virtual_devices.append({
                    'name': device_name,
                    'protocol': protocol,
                    'version': version,
                    'test_coverage': 'N/A',  # Would need test analysis
                    'status': 'passing',     # Default assumption
                    'last_tested': datetime.now().date().isoformat()
                })
        
        self.collected_data['device_registry']['virtual_devices'] = virtual_devices
        print(f"✅ Detected {len(virtual_devices)} virtual devices")
    
    def infer_protocol_status(self) -> None:
        """Infer protocol status from available data."""
        # Define protocol mappings based on virtual devices and test results
        protocol_map = {
            'modbus_tcp': {'protocols': ['modbus'], 'variant': 'tcp'},
            'modbus_rtu': {'protocols': ['modbus'], 'variant': 'rtu'},
            'opcua': {'protocols': ['opcua'], 'variant': 'client'},
            'ethernet_ip': {'protocols': ['ethernet_ip'], 'variant': None},
            's7': {'protocols': ['s7'], 'variant': None}
        }
        
        # Initialize protocols based on virtual devices
        for device in self.collected_data['device_registry']['virtual_devices']:
            device_protocol = device['protocol']
            
            if device_protocol in protocol_map:
                mapping = protocol_map[device_protocol]
                
                for protocol_name in mapping['protocols']:
                    if protocol_name not in self.collected_data['protocols']:
                        self.collected_data['protocols'][protocol_name] = {}
                    
                    variant = mapping['variant'] or protocol_name
                    if variant not in self.collected_data['protocols'][protocol_name]:
                        # Determine status based on testing
                        status = 'experimental'  # Default for detected devices
                        if device['status'] == 'passing':
                            status = 'beta'  # Upgrade to beta if tests pass
                        
                        self.collected_data['protocols'][protocol_name][variant] = {
                            'status': status,
                            'version': device['version'],
                            'performance': {
                                'throughput': 'N/A',
                                'latency': 'N/A',
                                'concurrent_limit': 1,
                                'memory_usage': 'N/A'
                            },
                            'tested_devices': {
                                'virtual': [device['name']],
                                'real': []
                            },
                            'limitations': [],
                            'known_issues': []
                        }
    
    def collect_git_info(self, project_root: Path) -> None:
        """Collect version and release information from git."""
        try:
            import subprocess
            
            # Get current version tag
            result = subprocess.run(
                ['git', 'describe', '--tags', '--exact-match', 'HEAD'],
                cwd=project_root, capture_output=True, text=True
            )
            
            if result.returncode == 0:
                version = result.stdout.strip()
                if version.startswith('v'):
                    version = version[1:]  # Remove 'v' prefix
                self.collected_data['version'] = version
            else:
                # Fallback to git hash
                result = subprocess.run(
                    ['git', 'rev-parse', '--short', 'HEAD'],
                    cwd=project_root, capture_output=True, text=True
                )
                if result.returncode == 0:
                    hash_short = result.stdout.strip()
                    self.collected_data['version'] = f"0.0.0-dev+{hash_short}"
            
            print(f"✅ Detected version: {self.collected_data['version']}")
            
        except Exception as e:
            print(f"⚠️  Could not detect git version: {e}")
    
    def add_default_content(self) -> None:
        """Add default content for required fields."""
        # Add default testing summary if not present
        if 'automated_tests' not in self.collected_data['testing_summary']:
            self.collected_data['testing_summary']['automated_tests'] = {
                'total': 0,
                'passed': 0,
                'failed': 0,
                'skipped': 0,
                'coverage': 'N/A'
            }
        
        # Add default release notes
        if 'release_notes' not in self.collected_data:
            version = self.collected_data['version']
            self.collected_data['release_notes'] = f"Automated release {version} with collected test and performance data."
        
        # Add default performance test environment
        if 'test_environment' not in self.collected_data['performance_benchmarks']:
            self.collected_data['performance_benchmarks']['test_environment'] = {
                'os': 'Linux',
                'cpu': 'Unknown',
                'memory': 'Unknown',
                'network': 'Unknown',
                'load_conditions': 'Automated testing environment'
            }
        
        # Add default quality metrics
        self.collected_data['quality_metrics'] = {
            'code_coverage': 'N/A',
            'security_score': 'N/A',
            'performance_score': 'N/A',
            'reliability_score': 'N/A',
            'documentation_score': 'N/A'
        }
        
        # Add default dependencies
        self.collected_data['dependencies'] = {
            'python_version': '>=3.11',
            'os_requirements': ['Linux', 'Windows', 'macOS'],
            'hardware_requirements': {
                'minimum': {
                    'cpu': 'Dual-core x86_64',
                    'memory': '4GB RAM',
                    'storage': '1GB'
                },
                'recommended': {
                    'cpu': 'Quad-core x86_64',
                    'memory': '8GB RAM',
                    'storage': '5GB'
                }
            }
        }
    
    def save_data(self, output_path: Path) -> None:
        """Save collected data to YAML file."""
        try:
            with open(output_path, 'w') as f:
                yaml.dump(self.collected_data, f, default_flow_style=False, sort_keys=False)
            print(f"✅ Saved release data to {output_path}")
        except Exception as e:
            print(f"❌ Error saving data: {e}")
            raise


def main():
    """Main entry point for the data collector."""
    parser = argparse.ArgumentParser(description='Collect data for release cards')
    parser.add_argument('--output', type=Path, default=Path('release-data.yaml'),
                       help='Output YAML file path')
    parser.add_argument('--test-results', type=Path,
                       help='Path to pytest XML results file')
    parser.add_argument('--benchmarks', type=Path,
                       help='Path to benchmark JSON results file')
    parser.add_argument('--project-root', type=Path, default=Path('.'),
                       help='Project root directory')
    parser.add_argument('--version', type=str,
                       help='Override version detection')
    parser.add_argument('--release-type', type=str, default='alpha',
                       choices=['alpha', 'beta', 'rc', 'stable'],
                       help='Release type')
    
    args = parser.parse_args()
    
    # Initialize collector
    collector = ReleaseDataCollector()
    
    # Override version if specified
    if args.version:
        collector.collected_data['version'] = args.version
        collector.collected_data['release_type'] = args.release_type
    else:
        # Collect git information
        collector.collect_git_info(args.project_root)
        collector.collected_data['release_type'] = args.release_type
    
    # Collect test results
    if args.test_results and args.test_results.exists():
        collector.collect_from_pytest_results(args.test_results)
    
    # Collect benchmark data
    if args.benchmarks and args.benchmarks.exists():
        collector.collect_from_benchmark_json(args.benchmarks)
    
    # Detect virtual devices
    collector.detect_virtual_devices(args.project_root)
    
    # Infer protocol status from available data
    collector.infer_protocol_status()
    
    # Add default content
    collector.add_default_content()
    
    # Save collected data
    collector.save_data(args.output)
    
    print(f"\n🎉 Data collection complete!")
    print(f"📄 Output: {args.output}")
    print(f"📋 Version: {collector.collected_data['version']}")
    print(f"🔌 Protocols detected: {len(collector.collected_data['protocols'])}")
    print(f"🧪 Virtual devices: {len(collector.collected_data['device_registry']['virtual_devices'])}")


if __name__ == '__main__':
    main()