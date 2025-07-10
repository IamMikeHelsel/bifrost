//! Bifrost Time-Series Storage Engine
//!
//! High-performance time-series storage optimized for edge devices.
//! Features circular buffering, compression, memory-mapped persistence,
//! and automatic data expiration.

use pyo3::prelude::*;

pub mod buffer;
pub mod compression;
pub mod error;
pub mod index;
pub mod persistence;
pub mod query;
pub mod storage;
pub mod types;

pub use buffer::CircularBuffer;
pub use error::{Result, TimeSeriesError};
pub use storage::TimeSeriesEngine;
pub use types::{DataPoint, Timestamp, Value};

/// Python module for bifrost-timeseries
#[cfg(feature = "python-bindings")]
#[pymodule]
fn bifrost_timeseries(m: &Bound<'_, PyModule>) -> PyResult<()> {
    m.add_class::<storage::PyTimeSeriesEngine>()?;
    m.add_class::<buffer::PyCircularBuffer>()?;
    m.add_class::<types::PyDataPoint>()?;
    Ok(())
}