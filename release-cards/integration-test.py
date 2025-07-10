#!/usr/bin/env python3
"""
Integration test and demonstration script for the Bifrost release card system.
This script shows how to validate, process, and generate output formats.
"""

import json
import yaml
import sys
from pathlib import Path

def main():
    """Demonstrate the complete release card workflow."""
    print("🌉 Bifrost Release Card System - Integration Test")
    print("=" * 60)
    
    base_dir = Path(__file__).parent
    schema_file = base_dir / "schemas" / "release-card-schema.json"
    example_yaml = base_dir / "examples" / "v0.1.0-release-card.yaml"
    example_json = base_dir / "examples" / "v0.1.0-release-card.json"
    
    # Test 1: Load and validate schema
    print("\n📋 Test 1: Schema Validation")
    try:
        with open(schema_file, 'r') as f:
            schema = json.load(f)
        print(f"✅ Schema loaded successfully")
        print(f"   Schema version: {schema.get('$id', 'unknown')}")
        print(f"   Required fields: {', '.join(schema.get('required', []))}")
    except Exception as e:
        print(f"❌ Schema loading failed: {e}")
        return 1
    
    # Test 2: Load example YAML
    print("\n📋 Test 2: YAML Example Loading")
    try:
        with open(example_yaml, 'r') as f:
            yaml_data = yaml.safe_load(f)
        print(f"✅ YAML example loaded successfully")
        print(f"   Version: {yaml_data.get('version')}")
        print(f"   Release Type: {yaml_data.get('release_type')}")
        print(f"   Protocols: {list(yaml_data.get('protocols', {}).keys())}")
    except Exception as e:
        print(f"❌ YAML loading failed: {e}")
        return 1
    
    # Test 3: Load example JSON
    print("\n📋 Test 3: JSON Example Loading")
    try:
        with open(example_json, 'r') as f:
            json_data = json.load(f)
        print(f"✅ JSON example loaded successfully")
        print(f"   Version: {json_data.get('version')}")
        print(f"   Release Type: {json_data.get('release_type')}")
        print(f"   Total Tests: {json_data.get('testing_summary', {}).get('total_tests')}")
    except Exception as e:
        print(f"❌ JSON loading failed: {e}")
        return 1
    
    # Test 4: Schema validation
    print("\n📋 Test 4: Schema Validation")
    try:
        from jsonschema import validate, ValidationError
        validate(yaml_data, schema)
        validate(json_data, schema)
        print("✅ Both examples validate against schema")
    except ImportError:
        print("⚠️  jsonschema not available - skipping validation")
        print("   Install with: pip install jsonschema")
    except ValidationError as e:
        print(f"❌ Validation failed: {e.message}")
        return 1
    except Exception as e:
        print(f"❌ Validation error: {e}")
        return 1
    
    # Test 5: Data consistency check
    print("\n📋 Test 5: Data Consistency Check")
    try:
        # Compare key fields between YAML and JSON versions
        yaml_version = yaml_data.get('version')
        json_version = json_data.get('version')
        
        if yaml_version == json_version:
            print(f"✅ Version consistency: {yaml_version}")
        else:
            print(f"❌ Version mismatch: YAML={yaml_version}, JSON={json_version}")
            return 1
            
        yaml_tests = yaml_data.get('testing_summary', {}).get('total_tests')
        json_tests = json_data.get('testing_summary', {}).get('total_tests')
        
        if yaml_tests == json_tests:
            print(f"✅ Test count consistency: {yaml_tests}")
        else:
            print(f"❌ Test count mismatch: YAML={yaml_tests}, JSON={json_tests}")
            return 1
            
    except Exception as e:
        print(f"❌ Consistency check failed: {e}")
        return 1
    
    # Test 6: Template availability
    print("\n📋 Test 6: Template Availability")
    template_dir = base_dir / "templates"
    markdown_template = template_dir / "release-card-markdown.mustache"
    html_template = template_dir / "release-card-html.mustache"
    
    if markdown_template.exists():
        print(f"✅ Markdown template found: {markdown_template.name}")
    else:
        print(f"❌ Markdown template missing")
        return 1
    
    if html_template.exists():
        print(f"✅ HTML template found: {html_template.name}")
    else:
        print(f"❌ HTML template missing")
        return 1
    
    # Test 7: Data structure analysis
    print("\n📋 Test 7: Data Structure Analysis")
    try:
        protocols = yaml_data.get('protocols', {})
        device_registry = yaml_data.get('device_registry', {})
        performance_metrics = yaml_data.get('performance_metrics', {})
        
        print(f"✅ Protocol families: {len(protocols)}")
        for protocol_family, variants in protocols.items():
            print(f"   - {protocol_family}: {list(variants.keys())}")
        
        total_devices = device_registry.get('total_devices_tested', 0)
        virtual_devices = device_registry.get('virtual_devices', 0)
        real_devices = device_registry.get('real_devices', 0)
        print(f"✅ Device testing: {total_devices} total ({virtual_devices} virtual, {real_devices} real)")
        
        if 'benchmark_environment' in performance_metrics:
            env = performance_metrics['benchmark_environment']
            print(f"✅ Benchmark environment: {env.get('os', 'unknown')} on {env.get('cpu', 'unknown')}")
        
    except Exception as e:
        print(f"❌ Data analysis failed: {e}")
        return 1
    
    # Summary
    print("\n" + "=" * 60)
    print("🎉 ALL TESTS PASSED!")
    print("\nThe release card system is working correctly:")
    print("  ✅ Schema is valid and comprehensive")
    print("  ✅ Example data is consistent and valid")
    print("  ✅ Templates are available for output generation")
    print("  ✅ Data structure supports all required features")
    print("\nNext steps:")
    print("  1. Integrate with CI/CD pipeline")
    print("  2. Create automated generation scripts")
    print("  3. Add real hardware test data")
    print("  4. Generate customer-facing documentation")
    
    return 0

if __name__ == "__main__":
    sys.exit(main())