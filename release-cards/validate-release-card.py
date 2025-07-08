#!/usr/bin/env python3
"""
Simple validation script for Bifrost release cards.
This script validates YAML/JSON release cards against the schema.
"""

import json
import sys
import yaml
from pathlib import Path
try:
    from jsonschema import validate, ValidationError
except ImportError:
    print("jsonschema not installed. Install with: pip install jsonschema")
    sys.exit(1)

def load_schema():
    """Load the release card JSON schema."""
    schema_path = Path(__file__).parent / "schemas" / "release-card-schema.json"
    with open(schema_path, 'r') as f:
        return json.load(f)

def validate_release_card(card_path):
    """Validate a release card file against the schema."""
    schema = load_schema()
    
    card_path = Path(card_path)
    if not card_path.exists():
        print(f"Error: File {card_path} does not exist")
        return False
    
    try:
        if card_path.suffix.lower() == '.yaml' or card_path.suffix.lower() == '.yml':
            with open(card_path, 'r') as f:
                data = yaml.safe_load(f)
        elif card_path.suffix.lower() == '.json':
            with open(card_path, 'r') as f:
                data = json.load(f)
        else:
            print(f"Error: Unsupported file format {card_path.suffix}")
            return False
            
        validate(data, schema)
        print(f"✅ {card_path} is valid!")
        return True
        
    except yaml.YAMLError as e:
        print(f"❌ YAML parsing error in {card_path}: {e}")
        return False
    except json.JSONDecodeError as e:
        print(f"❌ JSON parsing error in {card_path}: {e}")
        return False
    except ValidationError as e:
        print(f"❌ Schema validation error in {card_path}:")
        print(f"   {e.message}")
        if e.path:
            print(f"   Path: {' -> '.join(str(p) for p in e.path)}")
        return False
    except Exception as e:
        print(f"❌ Unexpected error validating {card_path}: {e}")
        return False

def main():
    """Main validation function."""
    if len(sys.argv) < 2:
        print("Usage: python validate-release-card.py <path-to-release-card>")
        print("Example: python validate-release-card.py examples/v0.1.0-release-card.yaml")
        return 1
    
    success = True
    for card_path in sys.argv[1:]:
        if not validate_release_card(card_path):
            success = False
    
    return 0 if success else 1

if __name__ == "__main__":
    sys.exit(main())