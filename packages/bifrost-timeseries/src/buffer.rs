//! High-performance circular buffer for time-series data

use crate::error::{Result, TimeSeriesError};
use crate::types::{DataPoint, Timestamp};
use std::collections::VecDeque;
use std::sync::{Arc, RwLock};

#[cfg(feature = "python-bindings")]
use pyo3::prelude::*;

/// High-performance circular buffer optimized for time-series data
#[derive(Debug)]
pub struct CircularBuffer {
    /// Internal ring buffer storage
    buffer: VecDeque<DataPoint>,
    /// Maximum capacity
    capacity: usize,
    /// TTL for data points in seconds (0 = no expiration)
    ttl_seconds: u64,
    /// Total number of points written (including evicted)
    total_written: u64,
    /// Total number of points evicted
    total_evicted: u64,
    /// Current memory usage in bytes
    memory_usage: usize,
}

impl CircularBuffer {
    /// Create a new circular buffer with specified capacity
    pub fn new(capacity: usize) -> Self {
        Self {
            buffer: VecDeque::with_capacity(capacity),
            capacity,
            ttl_seconds: 0,
            total_written: 0,
            total_evicted: 0,
            memory_usage: 0,
        }
    }

    /// Create a new circular buffer with capacity and TTL
    pub fn with_ttl(capacity: usize, ttl_seconds: u64) -> Self {
        Self {
            buffer: VecDeque::with_capacity(capacity),
            capacity,
            ttl_seconds,
            total_written: 0,
            total_evicted: 0,
            memory_usage: 0,
        }
    }

    /// Push a new data point into the buffer
    pub fn push(&mut self, data_point: DataPoint) -> Result<()> {
        // Remove expired data points first
        self.remove_expired();

        // Check if buffer is at capacity
        if self.buffer.len() >= self.capacity {
            // Remove oldest entry
            if let Some(removed) = self.buffer.pop_front() {
                self.memory_usage = self.memory_usage.saturating_sub(removed.size_bytes());
                self.total_evicted += 1;
            }
        }

        // Add new data point
        self.memory_usage += data_point.size_bytes();
        self.buffer.push_back(data_point);
        self.total_written += 1;

        Ok(())
    }

    /// Get data points in a time range
    pub fn get_range(&self, start: Timestamp, end: Timestamp) -> Vec<DataPoint> {
        self.buffer
            .iter()
            .filter(|dp| dp.timestamp >= start && dp.timestamp <= end)
            .cloned()
            .collect()
    }

    /// Get the most recent N data points
    pub fn get_latest(&self, count: usize) -> Vec<DataPoint> {
        self.buffer
            .iter()
            .rev()
            .take(count)
            .cloned()
            .collect::<Vec<_>>()
            .into_iter()
            .rev()
            .collect()
    }

    /// Get all data points as a vector
    pub fn get_all(&self) -> Vec<DataPoint> {
        self.buffer.iter().cloned().collect()
    }

    /// Get the number of data points currently in the buffer
    pub fn len(&self) -> usize {
        self.buffer.len()
    }

    /// Check if the buffer is empty
    pub fn is_empty(&self) -> bool {
        self.buffer.is_empty()
    }

    /// Check if the buffer is full
    pub fn is_full(&self) -> bool {
        self.buffer.len() >= self.capacity
    }

    /// Get the buffer capacity
    pub fn capacity(&self) -> usize {
        self.capacity
    }

    /// Get current memory usage in bytes
    pub fn memory_usage(&self) -> usize {
        self.memory_usage
    }

    /// Get total number of data points written
    pub fn total_written(&self) -> u64 {
        self.total_written
    }

    /// Get total number of data points evicted
    pub fn total_evicted(&self) -> u64 {
        self.total_evicted
    }

    /// Clear all data from the buffer
    pub fn clear(&mut self) {
        self.buffer.clear();
        self.memory_usage = 0;
    }

    /// Remove expired data points based on TTL
    fn remove_expired(&mut self) {
        if self.ttl_seconds == 0 {
            return; // No expiration
        }

        let current_time = chrono::Utc::now().timestamp_nanos_opt().unwrap_or(0);
        let ttl_nanos = (self.ttl_seconds as i64) * 1_000_000_000;

        while let Some(front) = self.buffer.front() {
            if current_time - front.timestamp > ttl_nanos {
                if let Some(removed) = self.buffer.pop_front() {
                    self.memory_usage = self.memory_usage.saturating_sub(removed.size_bytes());
                    self.total_evicted += 1;
                }
            } else {
                break; // Since buffer is ordered by time, we can stop here
            }
        }
    }

    /// Compact the buffer to optimize memory usage
    pub fn compact(&mut self) {
        self.remove_expired();
        self.buffer.shrink_to_fit();
    }

    /// Get buffer statistics
    pub fn stats(&self) -> BufferStats {
        BufferStats {
            capacity: self.capacity,
            current_size: self.buffer.len(),
            memory_usage: self.memory_usage,
            total_written: self.total_written,
            total_evicted: self.total_evicted,
            is_full: self.is_full(),
            oldest_timestamp: self.buffer.front().map(|dp| dp.timestamp),
            newest_timestamp: self.buffer.back().map(|dp| dp.timestamp),
        }
    }
}

/// Thread-safe circular buffer
#[derive(Debug, Clone)]
pub struct ThreadSafeCircularBuffer {
    inner: Arc<RwLock<CircularBuffer>>,
}

impl ThreadSafeCircularBuffer {
    /// Create a new thread-safe circular buffer
    pub fn new(capacity: usize) -> Self {
        Self {
            inner: Arc::new(RwLock::new(CircularBuffer::new(capacity))),
        }
    }

    /// Create a new thread-safe circular buffer with TTL
    pub fn with_ttl(capacity: usize, ttl_seconds: u64) -> Self {
        Self {
            inner: Arc::new(RwLock::new(CircularBuffer::with_ttl(capacity, ttl_seconds))),
        }
    }

    /// Push a data point into the buffer
    pub fn push(&self, data_point: DataPoint) -> Result<()> {
        self.inner
            .write()
            .map_err(|_| TimeSeriesError::configuration("Lock poisoned"))?
            .push(data_point)
    }

    /// Get data points in a time range
    pub fn get_range(&self, start: Timestamp, end: Timestamp) -> Result<Vec<DataPoint>> {
        Ok(self
            .inner
            .read()
            .map_err(|_| TimeSeriesError::configuration("Lock poisoned"))?
            .get_range(start, end))
    }

    /// Get buffer statistics
    pub fn stats(&self) -> Result<BufferStats> {
        Ok(self
            .inner
            .read()
            .map_err(|_| TimeSeriesError::configuration("Lock poisoned"))?
            .stats())
    }
}

/// Buffer statistics
#[derive(Debug, Clone)]
pub struct BufferStats {
    pub capacity: usize,
    pub current_size: usize,
    pub memory_usage: usize,
    pub total_written: u64,
    pub total_evicted: u64,
    pub is_full: bool,
    pub oldest_timestamp: Option<Timestamp>,
    pub newest_timestamp: Option<Timestamp>,
}

// Python bindings for circular buffer
#[cfg(feature = "python-bindings")]
#[pyclass]
pub struct PyCircularBuffer {
    inner: ThreadSafeCircularBuffer,
}

#[cfg(feature = "python-bindings")]
#[pymethods]
impl PyCircularBuffer {
    #[new]
    #[pyo3(signature = (capacity, ttl_seconds=None))]
    fn new(capacity: usize, ttl_seconds: Option<u64>) -> Self {
        let inner = match ttl_seconds {
            Some(ttl) => ThreadSafeCircularBuffer::with_ttl(capacity, ttl),
            None => ThreadSafeCircularBuffer::new(capacity),
        };
        Self { inner }
    }

    fn push(&self, data_point: &PyDataPoint) -> pyo3::PyResult<()> {
        self.inner
            .push(data_point.to_data_point())
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))
    }

    fn get_range(&self, start: i64, end: i64) -> pyo3::PyResult<Vec<PyDataPoint>> {
        let data_points = self
            .inner
            .get_range(start, end)
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;
        
        Ok(data_points
            .into_iter()
            .map(PyDataPoint::from_data_point)
            .collect())
    }

    fn stats(&self) -> pyo3::PyResult<PyObject> {
        let stats = self
            .inner
            .stats()
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;

        Python::with_gil(|py| {
            let dict = pyo3::types::PyDict::new_bound(py);
            dict.set_item("capacity", stats.capacity)?;
            dict.set_item("current_size", stats.current_size)?;
            dict.set_item("memory_usage", stats.memory_usage)?;
            dict.set_item("total_written", stats.total_written)?;
            dict.set_item("total_evicted", stats.total_evicted)?;
            dict.set_item("is_full", stats.is_full)?;
            dict.set_item("oldest_timestamp", stats.oldest_timestamp)?;
            dict.set_item("newest_timestamp", stats.newest_timestamp)?;
            Ok(dict.to_object(py))
        })
    }

    fn __len__(&self) -> pyo3::PyResult<usize> {
        let stats = self
            .inner
            .stats()
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;
        Ok(stats.current_size)
    }

    fn __repr__(&self) -> pyo3::PyResult<String> {
        let stats = self
            .inner
            .stats()
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;
        Ok(format!(
            "CircularBuffer(capacity={}, size={}, memory={})",
            stats.capacity, stats.current_size, stats.memory_usage
        ))
    }
}

#[cfg(feature = "python-bindings")]
use crate::types::PyDataPoint;

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::Value;

    #[test]
    fn test_circular_buffer_basic() {
        let mut buffer = CircularBuffer::new(3);
        
        // Test empty buffer
        assert_eq!(buffer.len(), 0);
        assert!(buffer.is_empty());
        assert!(!buffer.is_full());

        // Add data points
        let dp1 = DataPoint::with_timestamp(1000, Value::Integer(1));
        let dp2 = DataPoint::with_timestamp(2000, Value::Integer(2));
        let dp3 = DataPoint::with_timestamp(3000, Value::Integer(3));

        buffer.push(dp1.clone()).unwrap();
        buffer.push(dp2.clone()).unwrap();
        buffer.push(dp3.clone()).unwrap();

        assert_eq!(buffer.len(), 3);
        assert!(buffer.is_full());

        // Test overflow
        let dp4 = DataPoint::with_timestamp(4000, Value::Integer(4));
        buffer.push(dp4.clone()).unwrap();

        assert_eq!(buffer.len(), 3);
        assert_eq!(buffer.total_evicted(), 1);

        // Should contain dp2, dp3, dp4
        let all_data = buffer.get_all();
        assert_eq!(all_data.len(), 3);
        assert_eq!(all_data[0].timestamp, 2000);
        assert_eq!(all_data[2].timestamp, 4000);
    }

    #[test]
    fn test_time_range_query() {
        let mut buffer = CircularBuffer::new(10);

        // Add data points with different timestamps
        for i in 1..=5 {
            let dp = DataPoint::with_timestamp(i * 1000, Value::Integer(i));
            buffer.push(dp).unwrap();
        }

        // Query range
        let range_data = buffer.get_range(2000, 4000);
        assert_eq!(range_data.len(), 3);
        assert_eq!(range_data[0].timestamp, 2000);
        assert_eq!(range_data[2].timestamp, 4000);
    }

    #[test]
    fn test_latest_data() {
        let mut buffer = CircularBuffer::new(10);

        // Add data points
        for i in 1..=5 {
            let dp = DataPoint::with_timestamp(i * 1000, Value::Integer(i));
            buffer.push(dp).unwrap();
        }

        // Get latest 3
        let latest = buffer.get_latest(3);
        assert_eq!(latest.len(), 3);
        assert_eq!(latest[0].timestamp, 3000);
        assert_eq!(latest[2].timestamp, 5000);
    }
}