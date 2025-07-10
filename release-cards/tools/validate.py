#!/usr/bin/env python3
"""
Release Card Validator

This script validates release card YAML files against the schema and performs
additional consistency checks.

Usage:
    python validate.py <input.yaml> [--schema SCHEMA_PATH]
    
Example:
    python validate.py ../examples/v0.1.0.yaml
"""

import argparse
import json
import sys
from pathlib import Path
from typing import Any, Dict, List

try:
    import yaml
    from jsonschema import validate, ValidationError, Draft7Validator
except ImportError as e:
    print(f"Missing required dependency: {e}")
    print("Install with: pip install pyyaml jsonschema")
    sys.exit(1)


class ReleaseCardValidator:
    """Validate release card data against schema and business rules."""
    
    def __init__(self, schema_path: Path):
        """Initialize validator with schema path."""
        self.schema_path = schema_path
        self.schema = self._load_schema()
    
    def _load_schema(self) -> Dict[str, Any]:
        """Load and validate the JSON schema."""
        try:
            with open(self.schema_path, 'r') as f:
                schema = json.load(f)
            
            # Validate the schema itself
            Draft7Validator.check_schema(schema)
            return schema
            
        except FileNotFoundError:
            raise FileNotFoundError(f"Schema file not found: {self.schema_path}")
        except json.JSONDecodeError as e:
            raise ValueError(f"Invalid JSON in schema: {e}")
        except Exception as e:
            raise ValueError(f"Invalid schema: {e}")
    
    def load_release_card(self, yaml_path: Path) -> Dict[str, Any]:
        """Load release card data from YAML file."""
        try:
            with open(yaml_path, 'r') as f:
                data = yaml.safe_load(f)
            
            if not isinstance(data, dict):
                raise ValueError("Release card must be a YAML object/dictionary")
            
            return data
            
        except FileNotFoundError:
            raise FileNotFoundError(f"Release card file not found: {yaml_path}")
        except yaml.YAMLError as e:
            raise ValueError(f"Invalid YAML: {e}")
    
    def validate_schema(self, data: Dict[str, Any]) -> List[str]:
        """Validate data against JSON schema."""
        errors = []
        
        try:
            validate(instance=data, schema=self.schema)
        except ValidationError as e:
            # Get all validation errors, not just the first one
            validator = Draft7Validator(self.schema)
            for error in validator.iter_errors(data):
                # Format error path for better readability
                path = " -> ".join(str(p) for p in error.absolute_path)
                if path:
                    errors.append(f"At '{path}': {error.message}")
                else:
                    errors.append(f"Root level: {error.message}")
        
        return errors
    
    def validate_business_rules(self, data: Dict[str, Any]) -> List[str]:
        """Validate business logic and consistency rules."""
        errors = []
        
        # Check required fields that should be meaningful
        errors.extend(self._validate_required_content(data))
        
        # Check test count consistency
        errors.extend(self._validate_test_counts(data))
        
        # Check protocol status consistency
        errors.extend(self._validate_protocol_status(data))
        
        # Check performance benchmark consistency
        errors.extend(self._validate_performance_data(data))
        
        # Check device registry consistency
        errors.extend(self._validate_device_registry(data))
        
        # Check version format
        errors.extend(self._validate_version_format(data))
        
        return errors
    
    def _validate_required_content(self, data: Dict[str, Any]) -> List[str]:
        """Validate that required content is meaningful."""
        errors = []
        
        # Check that release notes aren't empty
        if not data.get('release_notes', '').strip():
            errors.append("Release notes should not be empty")
        
        # Check that at least one protocol is defined
        protocols = data.get('protocols', {})
        if not protocols:
            errors.append("At least one protocol should be defined")
        
        # Check that testing summary has meaningful data
        testing = data.get('testing_summary', {})
        if testing:
            total_tests = sum(
                testing.get(test_type, {}).get('total', 0)
                for test_type in ['automated_tests', 'manual_tests', 'regression_tests', 'security_tests']
            )
            if total_tests == 0:
                errors.append("Testing summary indicates no tests were run")
        
        return errors
    
    def _validate_test_counts(self, data: Dict[str, Any]) -> List[str]:
        """Validate that test counts are consistent."""
        errors = []
        
        testing = data.get('testing_summary', {})
        for test_type in ['automated_tests', 'manual_tests', 'regression_tests', 'security_tests']:
            if test_type in testing:
                test_data = testing[test_type]
                total = test_data.get('total', 0)
                passed = test_data.get('passed', 0)
                failed = test_data.get('failed', 0)
                skipped = test_data.get('skipped', 0)
                pending = test_data.get('pending', 0)
                
                # Calculate sum (pending only exists for manual tests)
                calculated_total = passed + failed + skipped
                if test_type == 'manual_tests':
                    calculated_total += pending
                
                if total != calculated_total:
                    errors.append(
                        f"{test_type}: Total ({total}) doesn't match sum of "
                        f"passed ({passed}) + failed ({failed}) + skipped ({skipped})"
                        + (f" + pending ({pending})" if test_type == 'manual_tests' else "")
                        + f" = {calculated_total}"
                    )
                
                # Validate that counts are non-negative
                for field, value in test_data.items():
                    if field in ['total', 'passed', 'failed', 'skipped', 'pending'] and value < 0:
                        errors.append(f"{test_type}.{field} cannot be negative")
        
        return errors
    
    def _validate_protocol_status(self, data: Dict[str, Any]) -> List[str]:
        """Validate protocol status consistency."""
        errors = []
        
        protocols = data.get('protocols', {})
        for protocol_name, protocol_data in protocols.items():
            if not isinstance(protocol_data, dict):
                continue
                
            for variant_name, variant_data in protocol_data.items():
                if not isinstance(variant_data, dict):
                    continue
                
                status = variant_data.get('status')
                tested_devices = variant_data.get('tested_devices', {})
                virtual_devices = tested_devices.get('virtual', [])
                real_devices = tested_devices.get('real', [])
                
                # Check that stable/beta protocols have virtual device testing
                if status in ['stable', 'beta'] and not virtual_devices:
                    errors.append(
                        f"{protocol_name}.{variant_name}: Status '{status}' but no virtual devices tested"
                    )
                
                # Check that unsupported protocols don't have performance data
                if status == 'unsupported':
                    performance = variant_data.get('performance', {})
                    if any(performance.get(key) not in [None, '', 'N/A', 0] for key in performance):
                        errors.append(
                            f"{protocol_name}.{variant_name}: Status 'unsupported' but has performance data"
                        )
                
                # Check that experimental protocols have limitations documented
                if status == 'experimental':
                    limitations = variant_data.get('limitations', [])
                    if not limitations:
                        errors.append(
                            f"{protocol_name}.{variant_name}: Experimental status should document limitations"
                        )
        
        return errors
    
    def _validate_performance_data(self, data: Dict[str, Any]) -> List[str]:
        """Validate performance benchmark data consistency."""
        errors = []
        
        benchmarks = data.get('performance_benchmarks', {})
        results = benchmarks.get('results', [])
        
        for i, result in enumerate(results):
            # Check that numeric values are reasonable
            value = result.get('value')
            target = result.get('target')
            
            if isinstance(value, (int, float)) and isinstance(target, (int, float)):
                if value < 0:
                    errors.append(f"Performance result {i}: Value cannot be negative")
                if target <= 0:
                    errors.append(f"Performance result {i}: Target must be positive")
                
                # Check status consistency with value vs target
                status = result.get('status')
                if status == 'pass' and value < target * 0.8:  # Allow 20% tolerance
                    errors.append(
                        f"Performance result {i}: Status 'pass' but value ({value}) is significantly below target ({target})"
                    )
                elif status == 'fail' and value >= target:
                    errors.append(
                        f"Performance result {i}: Status 'fail' but value ({value}) meets or exceeds target ({target})"
                    )
        
        return errors
    
    def _validate_device_registry(self, data: Dict[str, Any]) -> List[str]:
        """Validate device registry consistency."""
        errors = []
        
        registry = data.get('device_registry', {})
        
        # Check virtual devices
        virtual_devices = registry.get('virtual_devices', [])
        for i, device in enumerate(virtual_devices):
            # Check that test coverage is reasonable
            coverage = device.get('test_coverage', '')
            if coverage.endswith('%'):
                try:
                    coverage_value = float(coverage[:-1])
                    if not 0 <= coverage_value <= 100:
                        errors.append(f"Virtual device {i}: Test coverage must be 0-100%")
                except ValueError:
                    errors.append(f"Virtual device {i}: Invalid test coverage format")
        
        # Check real devices
        real_devices = registry.get('real_devices', [])
        for i, device in enumerate(real_devices):
            # Check that test coverage is reasonable
            coverage = device.get('test_coverage', '')
            if coverage.endswith('%'):
                try:
                    coverage_value = float(coverage[:-1])
                    if not 0 <= coverage_value <= 100:
                        errors.append(f"Real device {i}: Test coverage must be 0-100%")
                except ValueError:
                    errors.append(f"Real device {i}: Invalid test coverage format")
        
        return errors
    
    def _validate_version_format(self, data: Dict[str, Any]) -> List[str]:
        """Validate version format follows semantic versioning."""
        errors = []
        
        version = data.get('version', '')
        if version:
            # Basic semantic version pattern check
            import re
            pattern = r'^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+(\.[a-zA-Z0-9]+)*)?$'
            if not re.match(pattern, version):
                errors.append(f"Version '{version}' doesn't follow semantic versioning format")
        
        return errors
    
    def validate_all(self, data: Dict[str, Any]) -> tuple[List[str], List[str]]:
        """Perform all validations and return schema and business rule errors."""
        schema_errors = self.validate_schema(data)
        business_errors = self.validate_business_rules(data)
        return schema_errors, business_errors


def main():
    """Main entry point for the validator."""
    parser = argparse.ArgumentParser(description='Validate release card YAML files')
    parser.add_argument('input', type=Path, help='Input YAML file to validate')
    parser.add_argument('--schema', type=Path, 
                       help='Path to JSON schema file (default: auto-detect)')
    parser.add_argument('--strict', action='store_true',
                       help='Fail on business rule violations (not just schema errors)')
    parser.add_argument('--quiet', action='store_true',
                       help='Only output errors and warnings')
    
    args = parser.parse_args()
    
    # Determine schema path
    if args.schema:
        schema_path = args.schema
    else:
        # Auto-detect schema path relative to this script
        script_dir = Path(__file__).parent
        schema_path = script_dir.parent / "schema" / "release-card.schema.json"
    
    # Initialize validator
    try:
        validator = ReleaseCardValidator(schema_path)
    except Exception as e:
        print(f"Error initializing validator: {e}")
        sys.exit(1)
    
    # Load and validate data
    try:
        data = validator.load_release_card(args.input)
        
        if not args.quiet:
            print(f"📄 Validating: {args.input}")
            print(f"📋 Schema: {schema_path}")
            print()
        
        schema_errors, business_errors = validator.validate_all(data)
        
        # Report results
        exit_code = 0
        
        if schema_errors:
            print("❌ Schema Validation Errors:")
            for error in schema_errors:
                print(f"  - {error}")
            print()
            exit_code = 1
        elif not args.quiet:
            print("✅ Schema validation passed")
        
        if business_errors:
            print("⚠️  Business Rule Violations:")
            for error in business_errors:
                print(f"  - {error}")
            print()
            if args.strict:
                exit_code = 1
        elif not args.quiet:
            print("✅ Business rule validation passed")
        
        if exit_code == 0 and not args.quiet:
            print("🎉 All validations passed!")
        
        sys.exit(exit_code)
        
    except Exception as e:
        print(f"Error validating file: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()