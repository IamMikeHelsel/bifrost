//! Bifrost Core - Rust utilities for the Bifrost industrial `IoT` framework
//!
//! This module provides core Rust utilities that can be exposed to Python
//! for high-performance operations in industrial automation scenarios.

use pyo3::prelude::*;

/// Core utilities module for Bifrost
#[pymodule]
fn bifrost_core(_py: Python<'_>, m: &Bound<'_, PyModule>) -> PyResult<()> {
    m.add("__version__", env!("CARGO_PKG_VERSION"))?;

    // Add utility functions directly to the module
    m.add_function(wrap_pyfunction!(format_timestamp, m)?)?;
    m.add_function(wrap_pyfunction!(validate_device_id, m)?)?;

    Ok(())
}

/// Format a timestamp for display
#[pyfunction]
#[allow(clippy::cast_possible_truncation)]
fn format_timestamp(timestamp: f64) -> String {
    use chrono::{DateTime, Utc};
    let dt =
        DateTime::from_timestamp(timestamp as i64, 0).unwrap_or_else(Utc::now);
    dt.format("%Y-%m-%d %H:%M:%S UTC").to_string()
}

/// Validate a device ID according to Bifrost conventions
#[pyfunction]
fn validate_device_id(device_id: &str) -> bool {
    !device_id.is_empty()
        && device_id.len() <= 64
        && device_id
            .chars()
            .all(|c| c.is_alphanumeric() || c == '-' || c == '_')
}
