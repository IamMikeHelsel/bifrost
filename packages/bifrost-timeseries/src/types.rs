//! Core data types for the time-series storage engine

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[cfg(feature = "python-bindings")]
use pyo3::prelude::*;

/// Timestamp type for time-series data
pub type Timestamp = i64;

/// Value types supported by the time-series engine
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub enum Value {
    /// Integer value
    Integer(i64),
    /// Floating point value  
    Float(f64),
    /// Boolean value
    Boolean(bool),
    /// String value
    String(String),
    /// Byte array value
    Bytes(Vec<u8>),
}

impl Value {
    /// Convert value to bytes for storage
    pub fn to_bytes(&self) -> Vec<u8> {
        bincode::serialize(self).unwrap_or_default()
    }

    /// Create value from bytes
    pub fn from_bytes(bytes: &[u8]) -> Option<Self> {
        bincode::deserialize(bytes).ok()
    }

    /// Get the size in bytes for memory calculations
    pub fn size_bytes(&self) -> usize {
        match self {
            Value::Integer(_) => 8,
            Value::Float(_) => 8,
            Value::Boolean(_) => 1,
            Value::String(s) => s.len(),
            Value::Bytes(b) => b.len(),
        }
    }
}

/// A single data point in the time-series
#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct DataPoint {
    /// Timestamp in nanoseconds since Unix epoch
    pub timestamp: Timestamp,
    /// The value at this timestamp
    pub value: Value,
    /// Optional tags for metadata
    pub tags: Option<std::collections::HashMap<String, String>>,
}

impl DataPoint {
    /// Create a new data point with current timestamp
    pub fn new(value: Value) -> Self {
        Self {
            timestamp: Utc::now().timestamp_nanos_opt().unwrap_or(0),
            value,
            tags: None,
        }
    }

    /// Create a new data point with specific timestamp
    pub fn with_timestamp(timestamp: Timestamp, value: Value) -> Self {
        Self {
            timestamp,
            value,
            tags: None,
        }
    }

    /// Create a new data point with tags
    pub fn with_tags(
        timestamp: Timestamp,
        value: Value,
        tags: std::collections::HashMap<String, String>,
    ) -> Self {
        Self {
            timestamp,
            value,
            tags: Some(tags),
        }
    }

    /// Get the data point as a DateTime
    pub fn datetime(&self) -> DateTime<Utc> {
        DateTime::from_timestamp_nanos(self.timestamp)
    }

    /// Get the size in bytes for memory calculations
    pub fn size_bytes(&self) -> usize {
        let base_size = 8 + self.value.size_bytes(); // timestamp + value
        let tags_size = self
            .tags
            .as_ref()
            .map(|tags| {
                tags.iter()
                    .map(|(k, v)| k.len() + v.len())
                    .sum::<usize>()
            })
            .unwrap_or(0);
        base_size + tags_size
    }
}

/// Configuration for time-series storage
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TimeSeriesConfig {
    /// Maximum number of data points to store
    pub max_capacity: usize,
    /// Time-to-live for data points in seconds (0 = no expiration)
    pub ttl_seconds: u64,
    /// Enable compression
    pub enable_compression: bool,
    /// Compression level (1-22 for zstd)
    pub compression_level: i32,
    /// Enable memory-mapped persistence
    pub enable_persistence: bool,
    /// Path for persistent storage
    pub storage_path: Option<String>,
    /// Flush interval in seconds for persistence
    pub flush_interval_seconds: u64,
}

impl Default for TimeSeriesConfig {
    fn default() -> Self {
        Self {
            max_capacity: 1_000_000,  // 1M data points
            ttl_seconds: 3600,        // 1 hour
            enable_compression: true,
            compression_level: 3,     // Balanced compression
            enable_persistence: false,
            storage_path: None,
            flush_interval_seconds: 60, // 1 minute
        }
    }
}

/// Aggregation types for queries
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum AggregationType {
    /// Count of data points
    Count,
    /// Minimum value
    Min,
    /// Maximum value
    Max,
    /// Average value
    Average,
    /// Sum of values
    Sum,
    /// First value in time range
    First,
    /// Last value in time range
    Last,
}

/// Query result containing aggregated data
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AggregationResult {
    /// The aggregation type performed
    pub aggregation: AggregationType,
    /// The aggregated value
    pub value: Option<Value>,
    /// Number of data points aggregated
    pub count: usize,
    /// Start timestamp of the aggregation window
    pub start_timestamp: Timestamp,
    /// End timestamp of the aggregation window
    pub end_timestamp: Timestamp,
}

// Python bindings for data types
#[cfg(feature = "python-bindings")]
#[pyclass]
#[derive(Clone)]
pub struct PyDataPoint {
    inner: DataPoint,
}

#[cfg(feature = "python-bindings")]
#[pymethods]
impl PyDataPoint {
    #[new]
    fn new(timestamp: i64, value: PyObject) -> pyo3::PyResult<Self> {
        let value = python_value_to_value(value)?;
        Ok(Self {
            inner: DataPoint::with_timestamp(timestamp, value),
        })
    }

    #[getter]
    fn timestamp(&self) -> i64 {
        self.inner.timestamp
    }

    #[getter]
    fn value(&self) -> PyObject {
        value_to_python_value(&self.inner.value)
    }

    fn __repr__(&self) -> String {
        format!("DataPoint(timestamp={}, value={:?})", self.inner.timestamp, self.inner.value)
    }
}

#[cfg(feature = "python-bindings")]
impl PyDataPoint {
    pub fn from_data_point(data_point: DataPoint) -> Self {
        Self { inner: data_point }
    }

    pub fn to_data_point(&self) -> DataPoint {
        self.inner.clone()
    }
}

#[cfg(feature = "python-bindings")]
pub fn python_value_to_value(obj: PyObject) -> pyo3::PyResult<Value> {
    Python::with_gil(|py| {
        if let Ok(val) = obj.extract::<i64>(py) {
            Ok(Value::Integer(val))
        } else if let Ok(val) = obj.extract::<f64>(py) {
            Ok(Value::Float(val))
        } else if let Ok(val) = obj.extract::<bool>(py) {
            Ok(Value::Boolean(val))
        } else if let Ok(val) = obj.extract::<String>(py) {
            Ok(Value::String(val))
        } else if let Ok(val) = obj.extract::<Vec<u8>>(py) {
            Ok(Value::Bytes(val))
        } else {
            Err(pyo3::exceptions::PyTypeError::new_err(
                "Unsupported value type",
            ))
        }
    })
}

#[cfg(feature = "python-bindings")]
pub fn value_to_python_value(value: &Value) -> PyObject {
    Python::with_gil(|py| match value {
        Value::Integer(val) => val.to_object(py),
        Value::Float(val) => val.to_object(py),
        Value::Boolean(val) => val.to_object(py),
        Value::String(val) => val.to_object(py),
        Value::Bytes(val) => val.to_object(py),
    })
}