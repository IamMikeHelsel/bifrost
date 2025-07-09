# Cross-Platform Build Configuration
# This document outlines the cross-platform build strategy for Bifrost Rust components

## Target Platforms

### Primary Targets (Tier 1 Support)
- **Linux x86_64** (x86_64-unknown-linux-gnu)
  - Primary development and CI platform
  - Used in most server deployments

- **Windows x86_64** (x86_64-pc-windows-msvc)
  - Windows desktop and server environments
  - Industrial Windows-based systems

- **macOS x86_64** (x86_64-apple-darwin)
  - Developer workstations
  - macOS-based industrial systems

### Secondary Targets (Tier 2 Support)
- **Linux ARM64** (aarch64-unknown-linux-gnu)
  - Raspberry Pi 4+
  - ARM-based industrial computers
  - Edge computing devices

- **macOS ARM64** (aarch64-apple-darwin)
  - Apple Silicon Macs (M1/M2/M3)
  - Future developer adoption

## Build Matrix Configuration

### GitHub Actions Matrix
```yaml
strategy:
  matrix:
    include:
      # Primary targets
      - os: ubuntu-latest
        target: x86_64-unknown-linux-gnu
        use-cross: false
      
      - os: windows-latest
        target: x86_64-pc-windows-msvc
        use-cross: false
      
      - os: macos-latest
        target: x86_64-apple-darwin
        use-cross: false
      
      # Secondary targets
      - os: ubuntu-latest
        target: aarch64-unknown-linux-gnu
        use-cross: true
      
      - os: macos-latest
        target: aarch64-apple-darwin
        use-cross: false
```

## Maturin Build Configuration

### Development Builds
```bash
# Local development (auto-detect platform)
maturin develop

# Specific platform development
maturin develop --target x86_64-unknown-linux-gnu
```

### Release Builds
```bash
# Single platform release
maturin build --release --target x86_64-unknown-linux-gnu

# Cross-compilation using cross
cross build --release --target aarch64-unknown-linux-gnu
```

### Wheel Distribution
```bash
# Build wheels for all supported platforms
maturin build --release --target x86_64-unknown-linux-gnu
maturin build --release --target x86_64-pc-windows-msvc
maturin build --release --target x86_64-apple-darwin
maturin build --release --target aarch64-unknown-linux-gnu
maturin build --release --target aarch64-apple-darwin
```

## Platform-Specific Considerations

### Linux x86_64
- Standard musl and glibc support
- Container-friendly builds
- Broad compatibility

### Windows x86_64
- MSVC toolchain required
- Windows-specific dependencies handled via Cargo
- .dll packaging with wheels

### macOS (x86_64 and ARM64)
- Universal binary support potential
- Code signing considerations for distribution
- Homebrew compatibility

### Linux ARM64 (Raspberry Pi)
- Cross-compilation from x86_64 hosts
- Optimized for ARMv8 architecture
- Container support for edge deployments

## Performance Optimization

### Target-Specific Features
```toml
[target.'cfg(target_arch = "x86_64")'.dependencies]
# x86_64 specific optimizations

[target.'cfg(target_arch = "aarch64")'.dependencies]
# ARM64 specific optimizations
```

### Conditional Compilation
```rust
#[cfg(target_arch = "x86_64")]
fn optimized_x86_64_function() {
    // x86_64 specific implementation
}

#[cfg(target_arch = "aarch64")]
fn optimized_arm64_function() {
    // ARM64 specific implementation
}
```

## Testing Strategy

### Platform Coverage
- Unit tests on all platforms
- Integration tests on primary platforms
- Performance benchmarks on representative hardware

### CI/CD Pipeline
1. **Lint and Format**: All platforms
2. **Unit Tests**: All platforms  
3. **Integration Tests**: Primary platforms
4. **Build Wheels**: All platforms
5. **Smoke Tests**: Representative platforms

## Dependencies

### Cross-Compilation Tools
- `cross` for Linux ARM64 builds
- Platform-specific Rust toolchains
- maturin with cross-compilation support

### Platform Installation
```bash
# Install targets
rustup target add x86_64-unknown-linux-gnu
rustup target add x86_64-pc-windows-msvc
rustup target add x86_64-apple-darwin
rustup target add aarch64-unknown-linux-gnu
rustup target add aarch64-apple-darwin

# Install cross for cross-compilation
cargo install cross
```

## Future Considerations

### Additional Targets
- **Linux ARM32** (armv7-unknown-linux-gnueabihf)
  - Legacy Raspberry Pi support
  - Older ARM industrial systems

- **RISC-V** (riscv64gc-unknown-linux-gnu)
  - Emerging industrial processor architecture
  - Future-proofing consideration

### Optimization Opportunities
- Link-time optimization (LTO)
- Profile-guided optimization (PGO)
- Platform-specific SIMD utilization
- Memory layout optimization per architecture