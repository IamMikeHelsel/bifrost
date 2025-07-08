#!/usr/bin/env python3
"""
Build system validation script for Bifrost packages.

This script validates the Bazel BUILD files and configuration
without requiring a full Bazel build (useful when network is limited).
"""

import os
import sys
from pathlib import Path


def validate_build_files():
    """Validate that BUILD.bazel files exist and have basic structure."""
    root_dir = Path(__file__).parent.parent
    packages_dir = root_dir / "packages"
    
    print("🔍 Validating Bazel BUILD files...")
    
    # Check that packages directory exists
    if not packages_dir.exists():
        print("❌ packages/ directory not found!")
        return False
    
    # Expected packages
    expected_packages = ["bifrost-core", "bifrost"]
    missing_packages = []
    
    for package in expected_packages:
        package_dir = packages_dir / package
        build_file = package_dir / "BUILD.bazel"
        
        if not package_dir.exists():
            missing_packages.append(package)
            continue
            
        if not build_file.exists():
            print(f"❌ BUILD.bazel missing for {package}")
            return False
        
        # Basic content validation
        with open(build_file) as f:
            content = f.read()
            
        # Check for required elements
        required_elements = [
            "py_library",
            "py_test", 
            "py_wheel",
            f'name = "{package.replace("-", "_")}"',
        ]
        
        for element in required_elements:
            if element not in content:
                print(f"❌ {package} BUILD.bazel missing: {element}")
                return False
        
        print(f"✅ {package} BUILD.bazel validated")
    
    if missing_packages:
        print(f"❌ Missing packages: {', '.join(missing_packages)}")
        return False
    
    # Check MODULE.bazel
    module_file = root_dir / "MODULE.bazel"
    if not module_file.exists():
        print("❌ MODULE.bazel not found!")
        return False
    
    with open(module_file) as f:
        module_content = f.read()
    
    if "rules_python" not in module_content:
        print("❌ MODULE.bazel missing rules_python")
        return False
    
    if "pip_deps" not in module_content:
        print("❌ MODULE.bazel missing pip configuration")
        return False
    
    print("✅ MODULE.bazel validated")
    
    # Check requirements_lock.txt
    req_file = root_dir / "requirements_lock.txt"
    if not req_file.exists():
        print("❌ requirements_lock.txt not found!")
        return False
    
    with open(req_file) as f:
        requirements = f.read()
    
    # Check for key dependencies
    key_deps = ["pydantic", "pymodbus", "pytest", "rich", "typer"]
    for dep in key_deps:
        if dep not in requirements:
            print(f"❌ requirements_lock.txt missing: {dep}")
            return False
    
    print("✅ requirements_lock.txt validated")
    
    print("\n🎉 All Bazel configuration files validated successfully!")
    return True


def show_build_structure():
    """Show the build structure that was created."""
    print("\n📁 Bazel Build Structure:")
    print("├── MODULE.bazel (Python 3.13 + rules_python)")
    print("├── WORKSPACE.bazel (minimal)")
    print("├── requirements_lock.txt (PyPI dependencies)")
    print("├── packages/")
    print("│   ├── bifrost-core/")
    print("│   │   └── BUILD.bazel (py_library + py_test + py_wheel)")
    print("│   └── bifrost/")
    print("│       └── BUILD.bazel (py_library + py_binary + py_test + py_wheel)")
    print("└── justfile (updated with Bazel commands)")


if __name__ == "__main__":
    success = validate_build_files()
    show_build_structure()
    
    if success:
        print("\n✅ Build system validation complete!")
        print("📝 To test with Bazel:")
        print("   bazel build //packages/...")
        print("   bazel test //packages/...")
        print("   bazel build //packages/...:wheel")
        sys.exit(0)
    else:
        print("\n❌ Build system validation failed!")
        sys.exit(1)