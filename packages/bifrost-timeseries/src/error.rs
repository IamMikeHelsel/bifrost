//! Error types for the time-series storage engine

use thiserror::Error;

/// Time-series storage engine error types
#[derive(Error, Debug)]
pub enum TimeSeriesError {
    #[error("Buffer overflow: capacity {capacity}, attempted size {size}")]
    BufferOverflow { capacity: usize, size: usize },

    #[error("Invalid timestamp: {timestamp}")]
    InvalidTimestamp { timestamp: i64 },

    #[error("Compression error: {message}")]
    CompressionError { message: String },

    #[error("Index error: {message}")]
    Index { message: String },

    #[error("Persistence error: {message}")]
    Persistence { message: String },

    #[error("Query error: {message}")]
    Query { message: String },

    #[error("Serialization error: {source}")]
    Serialization {
        #[from]
        source: serde_json::Error,
    },

    #[error("Memory mapping error: {source}")]
    MemoryMap {
        #[from]
        source: std::io::Error,
    },

    #[error("Configuration error: {message}")]
    Configuration { message: String },
}

/// Result type for time-series operations
pub type Result<T> = std::result::Result<T, TimeSeriesError>;

impl TimeSeriesError {
    pub fn index(message: impl Into<String>) -> Self {
        Self::Index {
            message: message.into(),
        }
    }

    pub fn persistence(message: impl Into<String>) -> Self {
        Self::Persistence {
            message: message.into(),
        }
    }

    pub fn query(message: impl Into<String>) -> Self {
        Self::Query {
            message: message.into(),
        }
    }

    pub fn configuration(message: impl Into<String>) -> Self {
        Self::Configuration {
            message: message.into(),
        }
    }

    pub fn compression(message: impl Into<String>) -> Self {
        Self::CompressionError {
            message: message.into(),
        }
    }
}

#[cfg(feature = "python-bindings")]
impl From<TimeSeriesError> for pyo3::PyErr {
    fn from(err: TimeSeriesError) -> Self {
        pyo3::exceptions::PyRuntimeError::new_err(err.to_string())
    }
}