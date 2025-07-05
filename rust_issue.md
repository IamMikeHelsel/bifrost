# Rust Build Issue in Bazel

## Problem Description

Rust projects within this repository are currently failing to build using the Bazel build system. Specifically, the `modbus_native_lib` target, defined in `packages/bifrost/native/BUILD.bazel`, is unable to resolve its Rust dependencies, such as `pyo3` and `tokio`.

The error message observed is:
```
ERROR: no such package '@@[unknown repo 'pyo3_macros' requested from @@]//': The repository '@@[unknown repo 'pyo3_macros' requested from @@]' could not be resolved: No repository visible as '@pyo3_macros' from main repository. Was the repository introduced in WORKSPACE? The WORKSPACE file is disabled by default in Bazel 8 (late 2024) and will be removed in Bazel 9 (late 2025), please migrate to Bzlmod.
```

## Analysis

1.  **Dependency Declaration:** The `WORKSPACE.bazel` file attempts to define `pyo3`, `pyo3_macros`, and `tokio` using `http_archive` rules. While this approach can work for generic external archives, it appears to be incompatible with how `rules_rust` (the Bazel rules for Rust) expects to manage Rust dependencies.
2.  **`rules_rust` and Cargo Integration:** `rules_rust` is designed to integrate closely with Cargo, Rust's official package manager. It typically expects to derive Bazel build rules from `Cargo.toml` and `Cargo.lock` files, which define a Rust project's dependencies. Manually defining individual Rust crates as `http_archive` in `WORKSPACE.bazel` bypasses this intended integration.
3.  **Bzlmod Recommendation:** The error message explicitly recommends migrating to Bzlmod. Bzlmod is Bazel's modern external dependency management system, which is intended to replace the traditional `WORKSPACE` file for dependency resolution. `rules_rust` is designed to work seamlessly with Bzlmod, simplifying the declaration and resolution of Rust dependencies.

## Root Cause

The primary reason for the build failure is a mismatch between how Rust dependencies are currently declared in `WORKSPACE.bazel` (using `http_archive`) and how `rules_rust` expects to resolve them (preferably through Cargo integration and Bzlmod). The manual `http_archive` declarations for Rust crates are not being correctly recognized or utilized by the `rust_library` rule.

## Recommended Solution

To properly build Rust projects with Bazel in this repository, the following steps are recommended:

1.  **Migrate to Bzlmod:**
    *   Create a `MODULE.bazel` file at the root of the repository.
    *   Define the `rules_rust` module and its dependencies within `MODULE.bazel`.
    *   Remove the `http_archive` declarations for `pyo3`, `pyo3_macros`, and `tokio` from `WORKSPACE.bazel`.
2.  **Leverage `rules_rust` Cargo Integration:**
    *   Ensure that `packages/bifrost/native/Cargo.toml` correctly lists all Rust dependencies.
    *   `rules_rust` provides mechanisms (e.g., `cargo_build_script` or direct `rust_library` usage with `deps` that reference Cargo-managed dependencies) to build Rust projects by leveraging their `Cargo.toml` and `Cargo.lock` files. The existing `rust_library` rule might need adjustments to correctly reference these dependencies once Bzlmod and `rules_rust` are properly configured.

This migration will align the Bazel build system with the recommended practices for building Rust projects, ensuring proper dependency resolution and a more maintainable build configuration.
