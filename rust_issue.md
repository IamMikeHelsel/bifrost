# Rust Build Issue in Bazel

## Problem Description

Rust projects within this repository are currently failing to build using the Bazel build system. Specifically, the `modbus_native_lib` target, defined in `packages/bifrost/native/BUILD.bazel`, is unable to resolve its Rust dependencies, such as `pyo3` and `tokio`.

The error message observed initially was:

```
ERROR: no such package '@@[unknown repo 'pyo3_macros' requested from @@]//': The repository '@@[unknown repo 'pyo3_macros' requested from @@]' could not be resolved: No repository visible as '@pyo3_macros' from main repository. Was the repository introduced in WORKSPACE? The WORKSPACE file is disabled by default in Bazel 8 (late 2024) and will be removed in Bazel 9 (late 2025), please migrate to Bzlmod.
```

## Analysis

1. **Dependency Declaration:** The `WORKSPACE.bazel` file attempts to define `pyo3`, `pyo3_macros`, and `tokio` using `http_archive` rules. While this approach can work for generic external archives, it appears to be incompatible with how `rules_rust` (the Bazel rules for Rust) expects to manage Rust dependencies.
2. **`rules_rust` and Cargo Integration:** `rules_rust` is designed to integrate closely with Cargo, Rust's official package manager. It typically expects to derive Bazel build rules from `Cargo.toml` and `Cargo.lock` files, which define a Rust project's dependencies. Manually defining individual Rust crates as `http_archive` in `WORKSPACE.bazel` bypasses this intended integration.
3. **Bzlmod Recommendation:** The initial error message explicitly recommended migrating to Bzlmod. Bzlmod is Bazel's modern external dependency management system, which is intended to replace the traditional `WORKSPACE` file for dependency resolution. `rules_rust` is designed to work seamlessly with Bzlmod, simplifying the declaration and resolution of Rust dependencies.

## Root Cause

The primary reason for the build failure is a mismatch between how Rust dependencies are currently declared in `WORKSPACE.bazel` (using `http_archive`) and how `rules_rust` expects to resolve them (preferably through Cargo integration and Bzlmod). The manual `http_archive` declarations for Rust crates are not being correctly recognized or utilized by the `rust_library` rule.

## Attempted Bzlmod Migration and Challenges

An attempt was made to migrate the Rust build to Bzlmod. This involved:

* Creating a `MODULE.bazel` file at the repository root.
* Removing `http_archive` and `git_repository` declarations from `WORKSPACE.bazel`.
* Modifying `packages/bifrost/native/BUILD.bazel` to reference dependencies from `@crates`.

However, the Bzlmod migration encountered several issues, primarily related to correctly configuring `rules_rust` within `MODULE.bazel`:

* **Incorrect `use_repo_rule` usage:** Initial attempts to use `use_repo_rule` for `crates_repository` were syntactically incorrect.
* **Incorrect `toolchain` attribute:** The `name` attribute was incorrectly used in `rust.toolchain()` and `python.toolchain()`.
* **`load` statement in `MODULE.bazel`:** Direct `load` statements for `crates_repository` are not allowed in `MODULE.bazel` files.
* **Unclear `rules_rust` Bzlmod API:** The correct way to enable Cargo dependency resolution (e.g., `rust.crates()`) within `MODULE.bazel` for `rules_rust` version 0.40.0 proved elusive without specific documentation or examples, leading to errors like "does not have a tag class named crates".

Due to these challenges and the inability to resolve the `rules_rust` Bzlmod configuration without further specific documentation, the Bzlmod migration attempt has been reverted.

## Current State

The Bazel configuration for Rust builds has been reverted to its original state. The issue of Rust projects not building in Bazel persists.

## Recommended Solution

To properly build Rust projects with Bazel in this repository, the following steps are recommended:

1. **Consult `rules_rust` Bzlmod Documentation:** Thoroughly review the official `rules_rust` documentation for version 0.40.0 (or the version being used) specifically regarding Bzlmod setup and Cargo dependency management. This is crucial to understand the correct API for `rust.crates()` or equivalent.
2. **Implement Bzlmod Migration Correctly:** Based on the documentation, re-attempt the Bzlmod migration, ensuring that `rules_rust` is configured to correctly resolve Rust dependencies from `Cargo.toml` and `Cargo.lock`.
3. **Leverage `rules_rust` Cargo Integration:** Ensure that `packages/bifrost/native/Cargo.toml` correctly lists all Rust dependencies. The `rust_library` rule in `packages/bifrost/native/BUILD.bazel` should then correctly reference these dependencies once Bzlmod and `rules_rust` are properly configured.

This migration will align the Bazel build system with the recommended practices for building Rust projects, ensuring proper dependency resolution and a more maintainable build configuration.
