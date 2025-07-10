#!/usr/bin/env python3
"""
Template generator for Bifrost release cards.
Generates Markdown and HTML documentation from release card data.
"""

import json
import yaml
import sys
import subprocess
from pathlib import Path

def load_data(card_path):
    """Load release card data from YAML or JSON file."""
    card_path = Path(card_path)
    
    if card_path.suffix.lower() in ['.yaml', '.yml']:
        with open(card_path, 'r') as f:
            return yaml.safe_load(f)
    elif card_path.suffix.lower() == '.json':
        with open(card_path, 'r') as f:
            return json.load(f)
    else:
        raise ValueError(f"Unsupported file format: {card_path.suffix}")

def generate_with_mustache(data, template_path, output_path):
    """Generate output using mustache template engine."""
    # Convert data to JSON for mustache
    json_data = json.dumps(data, indent=2)
    
    try:
        # Use mustache CLI if available
        result = subprocess.run([
            'mustache', '-', str(template_path)
        ], input=json_data, text=True, capture_output=True, check=True)
        
        with open(output_path, 'w') as f:
            f.write(result.stdout)
        return True
    except (subprocess.CalledProcessError, FileNotFoundError):
        return False

def generate_simple_markdown(data, output_path):
    """Generate simple markdown without mustache (fallback)."""
    with open(output_path, 'w') as f:
        f.write(f"# Bifrost Release Card - {data.get('version', 'Unknown')}\n\n")
        f.write(f"**Release Type:** {data.get('release_type', 'Unknown')} | ")
        f.write(f"**Release Date:** {data.get('release_date', 'Unknown')}\n\n")
        
        # Major features
        if 'major_features' in data:
            f.write("## 🚀 Major Features\n\n")
            for feature in data['major_features']:
                f.write(f"- {feature}\n")
            f.write("\n")
        
        # Protocols
        if 'protocols' in data:
            f.write("## 📡 Protocol Support\n\n")
            for protocol_family, variants in data['protocols'].items():
                f.write(f"### {protocol_family.title()}\n\n")
                for variant, details in variants.items():
                    f.write(f"#### {variant.upper()}\n")
                    f.write(f"- **Status:** {details.get('status', 'unknown')}\n")
                    if 'version' in details:
                        f.write(f"- **Version:** {details['version']}\n")
                    if 'performance' in details:
                        perf = details['performance']
                        if 'throughput' in perf:
                            f.write(f"- **Throughput:** {perf['throughput']}\n")
                        if 'latency' in perf:
                            f.write(f"- **Latency:** {perf['latency']}\n")
                    f.write("\n")
        
        # Testing summary
        if 'testing_summary' in data:
            ts = data['testing_summary']
            f.write("## 🧪 Testing Summary\n\n")
            f.write(f"- **Total Tests:** {ts.get('total_tests', 0)}\n")
            f.write(f"- **Passed:** {ts.get('passed', 0)}\n")
            f.write(f"- **Failed:** {ts.get('failed', 0)}\n")
            if 'coverage_percentage' in ts:
                f.write(f"- **Coverage:** {ts['coverage_percentage']}\n")
            f.write("\n")
        
        # Known issues
        if 'known_issues' in data:
            f.write("## ⚠️ Known Issues\n\n")
            for issue in data['known_issues']:
                f.write(f"- **{issue.get('severity', 'unknown').upper()}:** {issue.get('issue', 'No description')}\n")
                if 'workaround' in issue:
                    f.write(f"  - **Workaround:** {issue['workaround']}\n")
                f.write("\n")

def main():
    """Main template generation function."""
    if len(sys.argv) < 2:
        print("Usage: python generate-templates.py <release-card-file> [output-prefix]")
        print("Example: python generate-templates.py examples/v0.1.0-release-card.yaml v0.1.0")
        return 1
    
    card_path = sys.argv[1]
    output_prefix = sys.argv[2] if len(sys.argv) > 2 else "release-card"
    
    base_dir = Path(__file__).parent
    
    try:
        # Load release card data
        print(f"📖 Loading release card: {card_path}")
        data = load_data(card_path)
        version = data.get('version', 'unknown')
        print(f"✅ Loaded {version}")
        
        # Generate Markdown
        markdown_output = f"{output_prefix}.md"
        markdown_template = base_dir / "templates" / "release-card-markdown.mustache"
        
        print(f"📝 Generating Markdown: {markdown_output}")
        if markdown_template.exists() and generate_with_mustache(data, markdown_template, markdown_output):
            print("✅ Markdown generated with mustache template")
        else:
            print("⚠️  Mustache not available, generating simple markdown")
            generate_simple_markdown(data, markdown_output)
            print("✅ Simple markdown generated")
        
        # Generate HTML
        html_output = f"{output_prefix}.html"
        html_template = base_dir / "templates" / "release-card-html.mustache"
        
        print(f"🌐 Generating HTML: {html_output}")
        if html_template.exists() and generate_with_mustache(data, html_template, html_output):
            print("✅ HTML generated with mustache template")
        else:
            print("⚠️  Cannot generate HTML without mustache")
        
        # Generate JSON (clean format)
        json_output = f"{output_prefix}.json"
        print(f"📋 Generating clean JSON: {json_output}")
        with open(json_output, 'w') as f:
            json.dump(data, f, indent=2, sort_keys=True)
        print("✅ JSON generated")
        
        print(f"\n🎉 Generated files:")
        print(f"  - {markdown_output}")
        if Path(html_output).exists():
            print(f"  - {html_output}")
        print(f"  - {json_output}")
        
        print(f"\n💡 To view HTML: open {html_output} in your browser")
        print(f"💡 To convert to PDF: wkhtmltopdf {html_output} {output_prefix}.pdf")
        
        return 0
        
    except Exception as e:
        print(f"❌ Error: {e}")
        return 1

if __name__ == "__main__":
    sys.exit(main())