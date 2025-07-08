#!/usr/bin/env python3
"""
Documentation Deployment Script for Bifrost Gateway

This script deploys generated documentation and release cards to the documentation site.
"""

import argparse
import json
import os
import shutil
import subprocess
import sys
from pathlib import Path
from typing import List, Dict, Any


class DocumentationDeployer:
    """Handles deployment of documentation and release cards."""
    
    def __init__(self, docs_dir: str = "docs", release_cards_dir: str = "release-cards"):
        self.docs_dir = Path(docs_dir)
        self.release_cards_dir = Path(release_cards_dir)
        self.deploy_config = self._load_deploy_config()
    
    def _load_deploy_config(self) -> Dict[str, Any]:
        """Load deployment configuration."""
        config_file = self.docs_dir / "deploy-config.yaml"
        default_config = {
            "github_pages": {
                "enabled": True,
                "branch": "gh-pages",
                "directory": "docs"
            },
            "s3": {
                "enabled": False,
                "bucket": "",
                "region": "us-east-1"
            },
            "artifacts": {
                "enabled": True,
                "retention_days": 90
            }
        }
        
        if config_file.exists():
            import yaml
            with open(config_file) as f:
                return yaml.safe_load(f)
        
        return default_config
    
    def validate_environment(self) -> bool:
        """Validate deployment environment and requirements."""
        required_dirs = [self.docs_dir, self.release_cards_dir]
        missing_dirs = [d for d in required_dirs if not d.exists()]
        
        if missing_dirs:
            print(f"❌ Missing required directories: {missing_dirs}")
            return False
        
        # Check for GitHub Pages deployment
        if self.deploy_config.get("github_pages", {}).get("enabled"):
            if not os.getenv("GITHUB_TOKEN"):
                print("⚠️  GITHUB_TOKEN not found, GitHub Pages deployment may fail")
        
        return True
    
    def prepare_documentation_site(self) -> str:
        """Prepare documentation site for deployment."""
        site_dir = Path("_site")
        site_dir.mkdir(exist_ok=True)
        
        # Copy main documentation
        if self.docs_dir.exists():
            for item in self.docs_dir.iterdir():
                if item.is_file() and item.suffix in ['.md', '.html', '.css', '.js']:
                    shutil.copy2(item, site_dir)
                elif item.is_dir() and item.name not in ['_site', '.git']:
                    shutil.copytree(item, site_dir / item.name, dirs_exist_ok=True)
        
        # Create release cards section
        release_cards_site_dir = site_dir / "release-cards"
        release_cards_site_dir.mkdir(exist_ok=True)
        
        if self.release_cards_dir.exists():
            for card_file in self.release_cards_dir.iterdir():
                if card_file.is_file():
                    shutil.copy2(card_file, release_cards_site_dir)
        
        # Generate index pages
        self._generate_site_index(site_dir)
        self._generate_release_cards_index(release_cards_site_dir)
        
        return str(site_dir)
    
    def _generate_site_index(self, site_dir: Path) -> None:
        """Generate main site index page."""
        index_content = """<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bifrost Gateway Documentation</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; }
        h1 { color: #333; border-bottom: 2px solid #007acc; padding-bottom: 10px; }
        .section { margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 8px; }
        .link { color: #007acc; text-decoration: none; }
        .link:hover { text-decoration: underline; }
        .badge { background: #007acc; color: white; padding: 4px 8px; border-radius: 4px; font-size: 12px; }
    </style>
</head>
<body>
    <h1>🌉 Bifrost Gateway Documentation</h1>
    
    <div class="section">
        <h2>📚 Documentation</h2>
        <ul>
            <li><a href="README.html" class="link">Getting Started</a></li>
            <li><a href="go-gateway/README.html" class="link">Go Gateway</a></li>
            <li><a href="virtual-devices/README.html" class="link">Virtual Devices</a></li>
        </ul>
    </div>
    
    <div class="section">
        <h2>🚀 Release Cards</h2>
        <p>View compatibility and performance information for each release:</p>
        <a href="release-cards/" class="link">Browse Release Cards →</a>
    </div>
    
    <div class="section">
        <h2>🔗 Links</h2>
        <ul>
            <li><a href="https://github.com/IamMikeHelsel/bifrost" class="link">GitHub Repository</a></li>
            <li><a href="https://github.com/IamMikeHelsel/bifrost/releases" class="link">Releases</a></li>
            <li><a href="https://github.com/IamMikeHelsel/bifrost/issues" class="link">Issues</a></li>
        </ul>
    </div>
    
    <footer style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #666;">
        <p>Generated automatically by Bifrost CI/CD pipeline</p>
    </footer>
</body>
</html>"""
        
        with open(site_dir / "index.html", 'w') as f:
            f.write(index_content)
    
    def _generate_release_cards_index(self, cards_dir: Path) -> None:
        """Generate release cards index page."""
        release_cards = []
        
        for card_file in cards_dir.glob("*.json"):
            try:
                with open(card_file) as f:
                    card_data = json.load(f)
                    release_cards.append({
                        "version": card_data.get("version", "unknown"),
                        "release_date": card_data.get("release_date", ""),
                        "release_type": card_data.get("release_type", "alpha"),
                        "filename": card_file.stem
                    })
            except json.JSONDecodeError:
                continue
        
        # Sort by version (newest first)
        release_cards.sort(key=lambda x: x["release_date"], reverse=True)
        
        cards_html = ""
        for card in release_cards:
            badge_color = {
                "stable": "#28a745",
                "rc": "#ffc107", 
                "beta": "#fd7e14",
                "alpha": "#dc3545"
            }.get(card["release_type"], "#6c757d")
            
            cards_html += f"""
            <div class="card">
                <h3>{card['version']} <span class="badge" style="background: {badge_color}">{card['release_type']}</span></h3>
                <p>Released: {card['release_date'][:10]}</p>
                <div class="links">
                    <a href="{card['filename']}.md" class="link">View Card</a> |
                    <a href="{card['filename']}.json" class="link">JSON</a> |
                    <a href="{card['filename']}.yaml" class="link">YAML</a>
                </div>
            </div>"""
        
        index_content = f"""<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Bifrost Release Cards</title>
    <style>
        body {{ font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; }}
        h1 {{ color: #333; border-bottom: 2px solid #007acc; padding-bottom: 10px; }}
        .card {{ margin: 20px 0; padding: 20px; border: 1px solid #ddd; border-radius: 8px; }}
        .link {{ color: #007acc; text-decoration: none; }}
        .link:hover {{ text-decoration: underline; }}
        .badge {{ color: white; padding: 4px 8px; border-radius: 4px; font-size: 12px; margin-left: 10px; }}
        .back-link {{ margin-bottom: 20px; }}
    </style>
</head>
<body>
    <div class="back-link">
        <a href="../" class="link">← Back to Documentation</a>
    </div>
    
    <h1>🚀 Release Cards</h1>
    <p>Compatibility and performance information for Bifrost Gateway releases.</p>
    
    {cards_html}
    
    <footer style="margin-top: 40px; padding-top: 20px; border-top: 1px solid #ddd; color: #666;">
        <p>Generated automatically by Bifrost CI/CD pipeline</p>
    </footer>
</body>
</html>"""
        
        with open(cards_dir / "index.html", 'w') as f:
            f.write(index_content)
    
    def deploy_to_github_pages(self, site_dir: str) -> bool:
        """Deploy to GitHub Pages."""
        if not self.deploy_config.get("github_pages", {}).get("enabled"):
            print("📄 GitHub Pages deployment disabled")
            return True
        
        try:
            print("🚀 Deploying to GitHub Pages...")
            
            # Use GitHub Pages action or git commands
            if os.getenv("GITHUB_ACTIONS"):
                # In GitHub Actions, use the pages deployment action
                print("📦 Preparing for GitHub Pages deployment...")
                return True
            else:
                # Local deployment using git
                subprocess.run([
                    "git", "add", site_dir,
                    "git", "commit", "-m", "Deploy documentation",
                    "git", "push", "origin", "gh-pages"
                ], check=True)
                
            print("✅ GitHub Pages deployment successful")
            return True
            
        except subprocess.CalledProcessError as e:
            print(f"❌ GitHub Pages deployment failed: {e}")
            return False
    
    def upload_artifacts(self, site_dir: str) -> bool:
        """Upload documentation as CI artifacts."""
        if not self.deploy_config.get("artifacts", {}).get("enabled"):
            return True
        
        print("📦 Preparing documentation artifacts...")
        
        # Create archive for upload
        archive_name = "documentation-site"
        shutil.make_archive(archive_name, 'zip', site_dir)
        
        print(f"✅ Documentation archived as {archive_name}.zip")
        return True
    
    def deploy_all(self) -> bool:
        """Deploy documentation using all configured methods."""
        if not self.validate_environment():
            return False
        
        # Prepare site
        site_dir = self.prepare_documentation_site()
        print(f"📁 Documentation site prepared in {site_dir}")
        
        success = True
        
        # Deploy to GitHub Pages
        if not self.deploy_to_github_pages(site_dir):
            success = False
        
        # Upload artifacts
        if not self.upload_artifacts(site_dir):
            success = False
        
        return success


def main():
    """Main function for command-line usage."""
    parser = argparse.ArgumentParser(description="Deploy Bifrost documentation")
    parser.add_argument("--docs-dir", default="docs", help="Documentation directory")
    parser.add_argument("--release-cards-dir", default="release-cards", help="Release cards directory")
    parser.add_argument("--verbose", "-v", action="store_true", help="Verbose output")
    parser.add_argument("--dry-run", action="store_true", help="Dry run (prepare only, don't deploy)")
    
    args = parser.parse_args()
    
    if args.verbose:
        print("🚀 Starting documentation deployment...")
    
    # Create deployer
    deployer = DocumentationDeployer(args.docs_dir, args.release_cards_dir)
    
    if args.dry_run:
        print("🧪 Dry run mode - preparing site only")
        site_dir = deployer.prepare_documentation_site()
        print(f"✅ Site prepared in {site_dir}")
        return
    
    # Deploy
    success = deployer.deploy_all()
    
    if success:
        print("🎉 Documentation deployment completed successfully!")
    else:
        print("❌ Documentation deployment failed")
        sys.exit(1)


if __name__ == "__main__":
    main()