# Phase 2: OPC UA Integration Plan

This document outlines the design and implementation plan for adding OPC UA support to Bifrost, as defined in `bifrost_dev_roadmap.md`.

## 1. Goals

- High-performance OPC UA client based on `open62541`.
- Safe Rust wrapper around the C library.
- Async Python API, consistent with the existing `ModbusConnection`.
- Basic OPC UA server implementation.
- Meet performance targets defined in the roadmap.

## 2. Core Components

### 2.1. Rust Wrapper (`opcua-wrapper`)

A new Rust crate/module will be created at `packages/bifrost/native/src/opcua/` to wrap `open62541`.

**Key responsibilities:**
- Provide safe abstractions over `unsafe` C functions.
- Manage memory for `open62541` objects.
- Handle data type conversions between Rust and C.
- Implement error handling, converting C error codes to Rust `Result`s.
- Expose a high-level Rust API for OPC UA operations (connect, browse, read, write, subscribe).

### 2.2. Build System Integration

- The `open62541` library located in `third_party/open62541` needs to be compiled and linked with the Rust wrapper.
- We will use a `build.rs` script in the Rust crate to handle the C library compilation, likely using the `cc` crate.
- The `Cargo.toml` for the native package will be updated to include the new `opcua-wrapper` module and its dependencies (e.g., `bindgen` to generate bindings).

### 2.3. Python Async Client (`bifrost.opcua`)

A new Python module `packages/bifrost/src/bifrost/opcua.py` will be created.

**Features:**
- `OpcuaConnection` class, following the async context manager pattern (`async with`).
- Methods for `connect`, `browse`, `read`, `write`, and `subscribe`.
- The Python layer will handle `asyncio` integration, calling the blocking Rust functions in a thread pool.
- Data will be represented using Pydantic models for validation and consistency.

## 3. Implementation Steps

1.  **Setup Build System:**
    - Create `packages/bifrost/native/src/opcua/mod.rs`.
    - Create a `build.rs` script for the native crate to compile `open62541`.
    - Use `bindgen` to generate Rust bindings for `open62541.h`.
    - Configure `Cargo.toml` to enable the new module.

2.  **Develop Rust Wrapper (Client):**
    - Implement `UA_Client` wrapper.
    - Implement `connect` and `disconnect` functionality.
    - Implement `read` and `write` for single nodes.
    - Implement `browse` functionality.
    - Implement subscriptions and monitored items.
    - Add comprehensive error handling.

3.  **Expose to Python:**
    - Create PyO3 wrappers for the client functions.
    - Handle data conversion between Python types and Rust/C types.
    - Release the GIL where appropriate for long-running operations.

4.  **Develop Python Client:**
    - Create `bifrost.opcua.OpcuaConnection`.
    - Implement the async methods, using `asyncio.to_thread`.
    - Add examples and documentation.

5.  **Benchmarking:**
    - Create a benchmark suite in `packages/bifrost/.benchmarks/` to test against the performance targets.

6.  **OPC UA Server (Stretch Goal):**
    - After the client is mature, implement a basic OPC UA server using the same wrapper architecture.

## 4. Risks and Mitigation

- **Complexity of `open62541`:** The library is large. We will focus on the client API first and wrap features incrementally.
- **`unsafe` code:** All `unsafe` blocks will be clearly documented and minimized. We will rely on Rust's ownership model to ensure safety.
- **Build complexity:** The `build.rs` script will be the most complex part of the integration. We will start with a minimal build and add features (like security policies) incrementally.

This plan will be updated as development progresses.
