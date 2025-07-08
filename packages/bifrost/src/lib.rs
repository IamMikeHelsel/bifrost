//! Bifrost - Main Rust components for industrial `IoT` framework
//!
//! This module contains the main Rust implementations for high-performance
//! industrial protocol handling, data processing, and automation tasks.

use pyo3::prelude::*;

/// Main Bifrost module with protocol implementations
#[pymodule]
fn bifrost(_py: Python<'_>, m: &Bound<'_, PyModule>) -> PyResult<()> {
    m.add("__version__", env!("CARGO_PKG_VERSION"))?;

    // Add classes and functions directly to the module
    m.add_class::<ModbusFunctionCode>()?;
    m.add_function(wrap_pyfunction!(parse_modbus_frame, m)?)?;
    m.add_function(wrap_pyfunction!(calculate_statistics, m)?)?;

    Ok(())
}

/// Modbus function codes
#[pyclass(eq, eq_int)]
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum ModbusFunctionCode {
    ReadCoils = 1,
    ReadDiscreteInputs = 2,
    ReadHoldingRegisters = 3,
    ReadInputRegisters = 4,
    WriteSingleCoil = 5,
    WriteSingleRegister = 6,
    WriteMultipleCoils = 15,
    WriteMultipleRegisters = 16,
}

/// High-performance Modbus frame parser (placeholder)
#[pyfunction]
fn parse_modbus_frame(data: &[u8]) -> PyResult<String> {
    if data.len() < 6 {
        return Err(pyo3::exceptions::PyValueError::new_err(
            "Modbus frame too short",
        ));
    }

    // Placeholder implementation
    Ok(format!("Parsed frame with {} bytes", data.len()))
}

/// Fast statistical calculations for sensor data
#[pyfunction]
#[allow(clippy::cast_precision_loss)]
fn calculate_statistics(values: Vec<f64>) -> PyResult<(f64, f64, f64)> {
    if values.is_empty() {
        return Err(pyo3::exceptions::PyValueError::new_err(
            "Cannot calculate statistics for empty dataset",
        ));
    }

    let sum: f64 = values.iter().sum();
    let mean = sum / values.len() as f64;

    let variance = values.iter().map(|x| (x - mean).powi(2)).sum::<f64>()
        / values.len() as f64;
    let std_dev = variance.sqrt();

    let mut sorted = values;
    sorted.sort_by(|a, b| a.partial_cmp(b).expect("NaN in data"));
    let median = if sorted.len() % 2 == 0 {
        f64::midpoint(sorted[sorted.len() / 2 - 1], sorted[sorted.len() / 2])
    } else {
        sorted[sorted.len() / 2]
    };

    Ok((mean, std_dev, median))
}
