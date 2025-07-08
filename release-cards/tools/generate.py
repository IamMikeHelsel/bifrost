#!/usr/bin/env python3
"""
Release Card Generator

This script generates release cards in multiple formats from YAML input data.
It validates the data against the schema and generates human-readable documentation.

Usage:
    python generate.py <input.yaml> [--output-dir OUTPUT] [--formats FORMAT1,FORMAT2]
    
Example:
    python generate.py examples/v0.1.0.yaml --output-dir ../docs --formats markdown,html
"""

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional

try:
    import yaml
    from jinja2 import Environment, FileSystemLoader, Template
    from jsonschema import validate, ValidationError
except ImportError as e:
    print(f"Missing required dependency: {e}")
    print("Install with: pip install pyyaml jinja2 jsonschema")
    sys.exit(1)


class ReleaseCardGenerator:
    """Generate release cards from YAML data using templates."""
    
    def __init__(self, schema_path: Optional[Path] = None, templates_dir: Optional[Path] = None):
        """Initialize the generator with schema and template paths."""
        self.script_dir = Path(__file__).parent
        self.schema_path = schema_path or self.script_dir.parent / "schema" / "release-card.schema.json"
        self.templates_dir = templates_dir or self.script_dir.parent / "templates"
        
        # Load schema for validation
        self.schema = self._load_schema()
        
        # Initialize Jinja2 environment
        self.jinja_env = Environment(
            loader=FileSystemLoader(str(self.templates_dir)),
            trim_blocks=True,
            lstrip_blocks=True
        )
        
        # Add custom filters
        self._setup_jinja_filters()
    
    def _load_schema(self) -> Dict[str, Any]:
        """Load the JSON schema for validation."""
        try:
            with open(self.schema_path, 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            print(f"Warning: Schema file not found at {self.schema_path}")
            return {}
        except json.JSONDecodeError as e:
            print(f"Error parsing schema JSON: {e}")
            return {}
    
    def _setup_jinja_filters(self) -> None:
        """Setup custom Jinja2 filters for formatting."""
        def status_emoji(status: str) -> str:
            """Convert status to emoji."""
            emoji_map = {
                'stable': '✅',
                'beta': '🔶',
                'experimental': '🧪',
                'unsupported': '❌',
                'passing': '✅',
                'failing': '❌',
                'unstable': '⚠️',
                'pass': '✅',
                'fail': '❌',
                'warn': '⚠️'
            }
            return emoji_map.get(status.lower(), '❓')
        
        def format_date(date_str: str) -> str:
            """Format date string for display."""
            try:
                dt = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
                return dt.strftime('%B %d, %Y')
            except:
                return date_str
        
        self.jinja_env.filters['status_emoji'] = status_emoji
        self.jinja_env.filters['format_date'] = format_date
    
    def validate_data(self, data: Dict[str, Any]) -> List[str]:
        """Validate release card data against schema."""
        errors = []
        
        if not self.schema:
            return ["Schema not available for validation"]
        
        try:
            validate(instance=data, schema=self.schema)
        except ValidationError as e:
            errors.append(f"Schema validation error: {e.message}")
        
        # Additional custom validations
        errors.extend(self._custom_validations(data))
        
        return errors
    
    def _custom_validations(self, data: Dict[str, Any]) -> List[str]:
        """Perform custom validation checks."""
        errors = []
        
        # Check that test totals are consistent
        if 'testing_summary' in data:
            for test_type in ['automated_tests', 'manual_tests', 'regression_tests', 'security_tests']:
                if test_type in data['testing_summary']:
                    test_data = data['testing_summary'][test_type]
                    total = test_data.get('total', 0)
                    passed = test_data.get('passed', 0)
                    failed = test_data.get('failed', 0)
                    skipped = test_data.get('skipped', 0)
                    pending = test_data.get('pending', 0)
                    
                    calculated_total = passed + failed + skipped + pending
                    if total != calculated_total:
                        errors.append(f"{test_type}: Total ({total}) doesn't match sum of individual counts ({calculated_total})")
        
        # Check that supported protocols have meaningful data
        if 'protocols' in data:
            for protocol_name, protocol_data in data['protocols'].items():
                if isinstance(protocol_data, dict):
                    for variant_name, variant_data in protocol_data.items():
                        if isinstance(variant_data, dict):
                            status = variant_data.get('status')
                            if status in ['stable', 'beta'] and not variant_data.get('tested_devices', {}).get('virtual'):
                                errors.append(f"{protocol_name}.{variant_name}: {status} status but no virtual devices tested")
        
        return errors
    
    def load_data(self, yaml_path: Path) -> Dict[str, Any]:
        """Load release card data from YAML file."""
        try:
            with open(yaml_path, 'r') as f:
                return yaml.safe_load(f)
        except FileNotFoundError:
            raise FileNotFoundError(f"Release card file not found: {yaml_path}")
        except yaml.YAMLError as e:
            raise ValueError(f"Error parsing YAML: {e}")
    
    def generate_markdown(self, data: Dict[str, Any]) -> str:
        """Generate Markdown release card."""
        template = self.jinja_env.get_template('markdown.md.j2')
        return template.render(**data)
    
    def generate_html(self, data: Dict[str, Any]) -> str:
        """Generate HTML release card."""
        try:
            template = self.jinja_env.get_template('html.html.j2')
            return template.render(**data)
        except Exception:
            # Fallback to markdown if HTML template not available
            markdown_content = self.generate_markdown(data)
            return f"""<!DOCTYPE html>
<html>
<head>
    <title>Bifrost Release Card: {data.get('version', 'Unknown')}</title>
    <meta charset="utf-8">
    <style>
        body {{ font-family: Arial, sans-serif; max-width: 1200px; margin: 0 auto; padding: 20px; }}
        h1, h2, h3 {{ color: #333; }}
        pre {{ background: #f5f5f5; padding: 10px; border-radius: 5px; overflow-x: auto; }}
        table {{ border-collapse: collapse; width: 100%; }}
        th, td {{ border: 1px solid #ddd; padding: 8px; text-align: left; }}
        th {{ background-color: #f2f2f2; }}
        .status-stable {{ color: #28a745; }}
        .status-beta {{ color: #ffc107; }}
        .status-experimental {{ color: #fd7e14; }}
        .status-unsupported {{ color: #dc3545; }}
    </style>
</head>
<body>
    <pre>{markdown_content}</pre>
</body>
</html>"""
    
    def generate_json(self, data: Dict[str, Any]) -> str:
        """Generate JSON API response format."""
        # Add metadata for API format
        api_data = {
            "release_card": data,
            "api_version": "1.0",
            "generated_at": datetime.utcnow().isoformat() + "Z"
        }
        return json.dumps(api_data, indent=2, default=str)
    
    def generate_all_formats(self, data: Dict[str, Any], output_dir: Path, formats: List[str]) -> Dict[str, Path]:
        """Generate release cards in all requested formats."""
        output_dir.mkdir(parents=True, exist_ok=True)
        generated_files = {}
        
        version = data.get('version', 'unknown')
        base_name = f"release-card-{version}"
        
        if 'markdown' in formats:
            md_content = self.generate_markdown(data)
            md_path = output_dir / f"{base_name}.md"
            with open(md_path, 'w') as f:
                f.write(md_content)
            generated_files['markdown'] = md_path
        
        if 'html' in formats:
            html_content = self.generate_html(data)
            html_path = output_dir / f"{base_name}.html"
            with open(html_path, 'w') as f:
                f.write(html_content)
            generated_files['html'] = html_path
        
        if 'json' in formats:
            json_content = self.generate_json(data)
            json_path = output_dir / f"{base_name}.json"
            with open(json_path, 'w') as f:
                f.write(json_content)
            generated_files['json'] = json_path
        
        return generated_files


def main():
    """Main entry point for the release card generator."""
    parser = argparse.ArgumentParser(description='Generate release cards from YAML data')
    parser.add_argument('input', type=Path, help='Input YAML file path')
    parser.add_argument('--output-dir', type=Path, default=Path('.'), 
                       help='Output directory for generated files')
    parser.add_argument('--formats', default='markdown', 
                       help='Comma-separated list of formats: markdown,html,json')
    parser.add_argument('--validate-only', action='store_true',
                       help='Only validate the input file without generating output')
    parser.add_argument('--quiet', action='store_true',
                       help='Suppress non-error output')
    
    args = parser.parse_args()
    
    # Parse formats
    formats = [f.strip().lower() for f in args.formats.split(',')]
    valid_formats = {'markdown', 'html', 'json'}
    invalid_formats = set(formats) - valid_formats
    if invalid_formats:
        print(f"Error: Invalid formats: {', '.join(invalid_formats)}")
        print(f"Valid formats: {', '.join(valid_formats)}")
        sys.exit(1)
    
    # Initialize generator
    try:
        generator = ReleaseCardGenerator()
    except Exception as e:
        print(f"Error initializing generator: {e}")
        sys.exit(1)
    
    # Load and validate data
    try:
        data = generator.load_data(args.input)
        
        validation_errors = generator.validate_data(data)
        if validation_errors:
            print("Validation errors found:")
            for error in validation_errors:
                print(f"  - {error}")
            
            if args.validate_only:
                sys.exit(1)
            else:
                print("\nProceeding with generation despite validation errors...")
        elif not args.quiet:
            print("✅ Data validation passed")
        
        if args.validate_only:
            print("✅ Validation complete")
            return
        
    except Exception as e:
        print(f"Error loading data: {e}")
        sys.exit(1)
    
    # Generate output files
    try:
        generated_files = generator.generate_all_formats(data, args.output_dir, formats)
        
        if not args.quiet:
            print(f"✅ Generated {len(generated_files)} file(s):")
            for format_name, file_path in generated_files.items():
                print(f"  - {format_name}: {file_path}")
    
    except Exception as e:
        print(f"Error generating files: {e}")
        sys.exit(1)


if __name__ == '__main__':
    main()