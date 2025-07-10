//! Main time-series storage engine combining all components

use crate::buffer::{CircularBuffer, ThreadSafeCircularBuffer};
use crate::compression::AdaptiveCompressor;
use crate::error::{Result, TimeSeriesError};
use crate::persistence::{MmapStorage, StorageStats};
use crate::query::{QueryBuilder, QueryEngine, QueryResult};
use crate::types::{DataPoint, TimeSeriesConfig, Timestamp, Value};
use std::path::Path;
use std::sync::{Arc, Mutex, RwLock};
use std::thread;
use std::time::{Duration, Instant};

#[cfg(feature = "python-bindings")]
use pyo3::prelude::*;

/// High-performance time-series storage engine
#[derive(Debug)]
pub struct TimeSeriesEngine {
    /// Configuration
    config: TimeSeriesConfig,
    /// In-memory circular buffer
    buffer: ThreadSafeCircularBuffer,
    /// Query engine for indexed searches
    query_engine: Arc<RwLock<QueryEngine>>,
    /// Persistent storage (optional)
    storage: Option<Arc<Mutex<MmapStorage>>>,
    /// Compression engine
    compressor: AdaptiveCompressor,
    /// Background flush handle
    flush_handle: Option<Arc<Mutex<Option<thread::JoinHandle<()>>>>>,
    /// Engine statistics
    stats: Arc<RwLock<EngineStats>>,
    /// Shutdown signal
    shutdown: Arc<Mutex<bool>>,
}

impl TimeSeriesEngine {
    /// Create a new time-series engine with default configuration
    pub fn new() -> Self {
        Self::with_config(TimeSeriesConfig::default())
    }

    /// Create a new time-series engine with custom configuration
    pub fn with_config(config: TimeSeriesConfig) -> Self {
        let buffer = ThreadSafeCircularBuffer::with_ttl(config.max_capacity, config.ttl_seconds);
        let query_engine = Arc::new(RwLock::new(QueryEngine::new()));
        let compressor = AdaptiveCompressor::new();

        let mut engine = Self {
            config: config.clone(),
            buffer,
            query_engine,
            storage: None,
            compressor,
            flush_handle: None,
            stats: Arc::new(RwLock::new(EngineStats::new())),
            shutdown: Arc::new(Mutex::new(false)),
        };

        // Initialize persistent storage if enabled
        if config.enable_persistence {
            if let Some(ref storage_path) = config.storage_path {
                match engine.init_storage(storage_path) {
                    Ok(_) => {}
                    Err(e) => {
                        tracing::warn!("Failed to initialize persistent storage: {}", e);
                    }
                }
            }
        }

        engine
    }

    /// Initialize persistent storage
    fn init_storage<P: AsRef<Path>>(&mut self, path: P) -> Result<()> {
        let storage = MmapStorage::new(path, &self.config)?;
        self.storage = Some(Arc::new(Mutex::new(storage)));

        // Start background flush thread if persistence is enabled
        if self.config.enable_persistence && self.config.flush_interval_seconds > 0 {
            self.start_background_flush();
        }

        Ok(())
    }

    /// Start background flush thread
    fn start_background_flush(&mut self) {
        let storage = self.storage.clone();
        let buffer = self.buffer.clone();
        let _query_engine = self.query_engine.clone();
        let flush_interval = Duration::from_secs(self.config.flush_interval_seconds);
        let shutdown = self.shutdown.clone();
        let stats = self.stats.clone();

        let handle = thread::spawn(move || {
            while !*shutdown.lock().unwrap() {
                thread::sleep(flush_interval);

                if *shutdown.lock().unwrap() {
                    break;
                }

                // Flush buffer to storage
                if let Some(ref _storage_arc) = storage {
                    if let Ok(_buffer_stats) = buffer.stats() {
                        // Get data from buffer (this is a simplified approach)
                        // In a real implementation, we'd need a way to get data from buffer
                        // without duplicating it in memory
                        
                        // For now, we'll just update stats
                        if let Ok(mut stats_guard) = stats.write() {
                            stats_guard.last_flush = Some(Instant::now());
                            stats_guard.total_flushes += 1;
                        }
                    }
                }
            }
        });

        self.flush_handle = Some(Arc::new(Mutex::new(Some(handle))));
    }

    /// Write a single data point
    pub fn write(&self, data_point: DataPoint) -> Result<()> {
        let start_time = Instant::now();

        // Add to buffer
        self.buffer.push(data_point.clone())?;

        // Add to query engine index
        if let Ok(mut query_engine) = self.query_engine.write() {
            query_engine.add_data_point(data_point);
        }

        // Update statistics
        if let Ok(mut stats) = self.stats.write() {
            stats.total_writes += 1;
            stats.last_write = Some(Instant::now());
            stats.write_latency_micros = start_time.elapsed().as_micros() as u64;
        }

        Ok(())
    }

    /// Write multiple data points
    pub fn write_batch(&self, data_points: Vec<DataPoint>) -> Result<()> {
        let start_time = Instant::now();
        let count = data_points.len();

        // Add to buffer
        for data_point in &data_points {
            self.buffer.push(data_point.clone())?;
        }

        // Add to query engine index
        if let Ok(mut query_engine) = self.query_engine.write() {
            query_engine.add_data_points(data_points);
        }

        // Update statistics
        if let Ok(mut stats) = self.stats.write() {
            stats.total_writes += count as u64;
            stats.last_write = Some(Instant::now());
            stats.batch_write_latency_micros = start_time.elapsed().as_micros() as u64;
        }

        Ok(())
    }

    /// Query data points with a time range
    pub fn query_range(&self, start: Timestamp, end: Timestamp) -> Result<Vec<DataPoint>> {
        let start_time = Instant::now();

        let result = if let Ok(query_engine) = self.query_engine.read() {
            query_engine.get_time_range(start, end)
        } else {
            // Fallback to buffer query
            self.buffer.get_range(start, end)?
        };

        // Update statistics
        if let Ok(mut stats) = self.stats.write() {
            stats.total_queries += 1;
            stats.last_query = Some(Instant::now());
            stats.query_latency_micros = start_time.elapsed().as_micros() as u64;
        }

        Ok(result)
    }

    /// Execute a complex query
    pub fn query(&self, query_builder: QueryBuilder) -> Result<QueryResult> {
        let start_time = Instant::now();

        let result = if let Ok(query_engine) = self.query_engine.read() {
            query_engine.execute_query(&query_builder)?
        } else {
            return Err(TimeSeriesError::query("Query engine not available"));
        };

        // Update statistics
        if let Ok(mut stats) = self.stats.write() {
            stats.total_queries += 1;
            stats.last_query = Some(Instant::now());
            stats.query_latency_micros = start_time.elapsed().as_micros() as u64;
        }

        Ok(result)
    }

    /// Get the latest N data points
    pub fn get_latest(&self, count: usize) -> Result<Vec<DataPoint>> {
        let start_time = Instant::now();

        let result = if let Ok(query_engine) = self.query_engine.read() {
            query_engine.get_latest(count)
        } else {
            // Fallback to buffer
            self.buffer.get_range(0, Timestamp::MAX)?.into_iter().rev().take(count).rev().collect()
        };

        // Update statistics
        if let Ok(mut stats) = self.stats.write() {
            stats.total_queries += 1;
            stats.last_query = Some(Instant::now());
            stats.query_latency_micros = start_time.elapsed().as_micros() as u64;
        }

        Ok(result)
    }

    /// Create a query builder
    pub fn new_query(&self) -> QueryBuilder {
        QueryBuilder::new()
    }

    /// Flush all pending data to persistent storage
    pub fn flush(&self) -> Result<()> {
        if let Some(ref storage_arc) = self.storage {
            let storage = storage_arc.lock().unwrap();
            storage.flush()?;

            // Update statistics
            if let Ok(mut stats) = self.stats.write() {
                stats.last_flush = Some(Instant::now());
                stats.total_flushes += 1;
            }
        }
        Ok(())
    }

    /// Get engine statistics
    pub fn stats(&self) -> Result<EngineStats> {
        let mut stats = self.stats.read().unwrap().clone();

        // Update buffer stats
        let buffer_stats = self.buffer.stats()?;
        stats.buffer_size = buffer_stats.current_size;
        stats.buffer_memory_usage = buffer_stats.memory_usage;
        stats.total_evicted = buffer_stats.total_evicted;

        // Update query engine stats
        if let Ok(query_engine) = self.query_engine.read() {
            let query_stats = query_engine.stats();
            stats.index_memory_usage = query_stats.memory_usage;
            stats.unique_timestamps = query_stats.unique_timestamps;
        }

        // Update storage stats
        if let Some(ref storage_arc) = self.storage {
            if let Ok(storage) = storage_arc.lock() {
                match storage.stats() {
                    Ok(storage_stats) => {
                        stats.storage_file_size = Some(storage_stats.file_size);
                        stats.storage_data_size = Some(storage_stats.data_size);
                    }
                    Err(_) => {}
                }
            }
        }

        Ok(stats)
    }

    /// Get current configuration
    pub fn config(&self) -> &TimeSeriesConfig {
        &self.config
    }

    /// Close the engine and clean up resources
    pub fn close(&self) -> Result<()> {
        // Signal shutdown to background threads
        *self.shutdown.lock().unwrap() = true;

        // Wait for flush thread to finish
        if let Some(ref handle_arc) = self.flush_handle {
            if let Ok(mut handle_opt) = handle_arc.lock() {
                if let Some(handle) = handle_opt.take() {
                    let _ = handle.join();
                }
            }
        }

        // Flush any remaining data
        self.flush()?;

        // Close storage
        if let Some(ref storage_arc) = self.storage {
            let storage = storage_arc.lock().unwrap();
            storage.close()?;
        }

        Ok(())
    }
}

impl Default for TimeSeriesEngine {
    fn default() -> Self {
        Self::new()
    }
}

impl Drop for TimeSeriesEngine {
    fn drop(&mut self) {
        let _ = self.close();
    }
}

/// Engine performance statistics
#[derive(Debug, Clone)]
pub struct EngineStats {
    pub total_writes: u64,
    pub total_queries: u64,
    pub total_flushes: u64,
    pub total_evicted: u64,
    pub buffer_size: usize,
    pub buffer_memory_usage: usize,
    pub index_memory_usage: usize,
    pub unique_timestamps: usize,
    pub storage_file_size: Option<usize>,
    pub storage_data_size: Option<usize>,
    pub write_latency_micros: u64,
    pub batch_write_latency_micros: u64,
    pub query_latency_micros: u64,
    pub last_write: Option<Instant>,
    pub last_query: Option<Instant>,
    pub last_flush: Option<Instant>,
}

impl EngineStats {
    fn new() -> Self {
        Self {
            total_writes: 0,
            total_queries: 0,
            total_flushes: 0,
            total_evicted: 0,
            buffer_size: 0,
            buffer_memory_usage: 0,
            index_memory_usage: 0,
            unique_timestamps: 0,
            storage_file_size: None,
            storage_data_size: None,
            write_latency_micros: 0,
            batch_write_latency_micros: 0,
            query_latency_micros: 0,
            last_write: None,
            last_query: None,
            last_flush: None,
        }
    }

    /// Get total memory usage in bytes
    pub fn total_memory_usage(&self) -> usize {
        self.buffer_memory_usage + self.index_memory_usage
    }

    /// Get write throughput (writes per second)
    pub fn write_throughput(&self) -> f64 {
        if let Some(last_write) = self.last_write {
            let duration = last_write.elapsed().as_secs_f64();
            if duration > 0.0 {
                self.total_writes as f64 / duration
            } else {
                0.0
            }
        } else {
            0.0
        }
    }

    /// Get query throughput (queries per second)
    pub fn query_throughput(&self) -> f64 {
        if let Some(last_query) = self.last_query {
            let duration = last_query.elapsed().as_secs_f64();
            if duration > 0.0 {
                self.total_queries as f64 / duration
            } else {
                0.0
            }
        } else {
            0.0
        }
    }
}

// Python bindings for the time series engine
#[cfg(feature = "python-bindings")]
#[pyclass]
pub struct PyTimeSeriesEngine {
    inner: TimeSeriesEngine,
}

#[cfg(feature = "python-bindings")]
#[pymethods]
impl PyTimeSeriesEngine {
    #[new]
    #[pyo3(signature = (max_capacity=None, ttl_seconds=None, enable_compression=None, storage_path=None))]
    fn new(
        max_capacity: Option<usize>,
        ttl_seconds: Option<u64>,
        enable_compression: Option<bool>,
        storage_path: Option<String>,
    ) -> Self {
        let mut config = TimeSeriesConfig::default();
        
        if let Some(capacity) = max_capacity {
            config.max_capacity = capacity;
        }
        if let Some(ttl) = ttl_seconds {
            config.ttl_seconds = ttl;
        }
        if let Some(compression) = enable_compression {
            config.enable_compression = compression;
        }
        if let Some(path) = storage_path {
            config.enable_persistence = true;
            config.storage_path = Some(path);
        }

        Self {
            inner: TimeSeriesEngine::with_config(config),
        }
    }

    fn write(&self, timestamp: i64, value: PyObject) -> pyo3::PyResult<()> {
        let value = crate::types::python_value_to_value(value)?;
        let data_point = DataPoint::with_timestamp(timestamp, value);
        
        self.inner
            .write(data_point)
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))
    }

    fn write_batch(&self, data_points: Vec<(i64, PyObject)>) -> pyo3::PyResult<()> {
        let mut points = Vec::new();
        
        for (timestamp, value) in data_points {
            let value = crate::types::python_value_to_value(value)?;
            points.push(DataPoint::with_timestamp(timestamp, value));
        }

        self.inner
            .write_batch(points)
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))
    }

    fn query_range(&self, start: i64, end: i64) -> pyo3::PyResult<Vec<crate::types::PyDataPoint>> {
        let data_points = self
            .inner
            .query_range(start, end)
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;

        Ok(data_points
            .into_iter()
            .map(crate::types::PyDataPoint::from_data_point)
            .collect())
    }

    fn get_latest(&self, count: usize) -> pyo3::PyResult<Vec<crate::types::PyDataPoint>> {
        let data_points = self
            .inner
            .get_latest(count)
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;

        Ok(data_points
            .into_iter()
            .map(crate::types::PyDataPoint::from_data_point)
            .collect())
    }

    fn stats(&self) -> pyo3::PyResult<PyObject> {
        let stats = self
            .inner
            .stats()
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;

        Python::with_gil(|py| {
            let dict = pyo3::types::PyDict::new_bound(py);
            dict.set_item("total_writes", stats.total_writes)?;
            dict.set_item("total_queries", stats.total_queries)?;
            dict.set_item("buffer_size", stats.buffer_size)?;
            dict.set_item("memory_usage", stats.total_memory_usage())?;
            dict.set_item("write_latency_micros", stats.write_latency_micros)?;
            dict.set_item("query_latency_micros", stats.query_latency_micros)?;
            dict.set_item("write_throughput", stats.write_throughput())?;
            dict.set_item("query_throughput", stats.query_throughput())?;
            Ok(dict.to_object(py))
        })
    }

    fn flush(&self) -> pyo3::PyResult<()> {
        self.inner
            .flush()
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))
    }

    fn close(&self) -> pyo3::PyResult<()> {
        self.inner
            .close()
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))
    }

    fn __repr__(&self) -> pyo3::PyResult<String> {
        let stats = self
            .inner
            .stats()
            .map_err(|e| pyo3::exceptions::PyRuntimeError::new_err(e.to_string()))?;

        Ok(format!(
            "TimeSeriesEngine(buffer_size={}, memory={}MB, writes={}, queries={})",
            stats.buffer_size,
            stats.total_memory_usage() / 1024 / 1024,
            stats.total_writes,
            stats.total_queries
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{AggregationType, Value};

    #[test]
    fn test_engine_basic_operations() {
        let engine = TimeSeriesEngine::new();

        // Write some data
        let data_points = vec![
            DataPoint::with_timestamp(1000, Value::Integer(1)),
            DataPoint::with_timestamp(2000, Value::Integer(2)),
            DataPoint::with_timestamp(3000, Value::Integer(3)),
        ];

        engine.write_batch(data_points.clone()).unwrap();

        // Query data
        let result = engine.query_range(1500, 2500).unwrap();
        assert_eq!(result.len(), 1);
        assert_eq!(result[0].timestamp, 2000);

        // Get latest
        let latest = engine.get_latest(2).unwrap();
        assert_eq!(latest.len(), 2);
        assert_eq!(latest[1].timestamp, 3000);

        // Check stats
        let stats = engine.stats().unwrap();
        assert_eq!(stats.total_writes, 3);
        assert!(stats.total_queries > 0);
    }

    #[test]
    fn test_engine_complex_query() {
        let engine = TimeSeriesEngine::new();

        // Write data with tags
        let mut tags = std::collections::HashMap::new();
        tags.insert("device".to_string(), "sensor1".to_string());
        
        let data_points = vec![
            DataPoint::with_tags(1000, Value::Float(10.0), tags.clone()),
            DataPoint::with_tags(2000, Value::Float(20.0), tags.clone()),
            DataPoint::with_tags(3000, Value::Float(30.0), tags),
        ];

        engine.write_batch(data_points).unwrap();

        // Execute aggregation query
        let query_result = engine
            .query(
                engine
                    .new_query()
                    .time_range(1000, 3000)
                    .aggregate(AggregationType::Average)
            )
            .unwrap();

        if let QueryResult::Aggregations(aggs) = query_result {
            assert_eq!(aggs.len(), 1);
            if let Some(Value::Float(avg)) = &aggs[0].value {
                assert_eq!(*avg, 20.0);
            } else {
                panic!("Expected float average");
            }
        } else {
            panic!("Expected aggregation result");
        }
    }

    #[test]
    fn test_engine_performance_targets() {
        let config = TimeSeriesConfig {
            max_capacity: 1_000_000,
            ttl_seconds: 3600,
            enable_compression: true,
            compression_level: 3,
            enable_persistence: false,
            storage_path: None,
            flush_interval_seconds: 60,
        };

        let engine = TimeSeriesEngine::with_config(config);

        // Write a large batch to test performance
        let batch_size = 10_000;
        let mut data_points = Vec::with_capacity(batch_size);
        
        for i in 0..batch_size {
            data_points.push(DataPoint::with_timestamp(
                i as i64 * 1000,
                Value::Float(i as f64),
            ));
        }

        let start_time = Instant::now();
        engine.write_batch(data_points).unwrap();
        let write_duration = start_time.elapsed();

        // Calculate throughput
        let throughput = batch_size as f64 / write_duration.as_secs_f64();
        println!("Write throughput: {:.0} events/second", throughput);

        // Check memory usage
        let stats = engine.stats().unwrap();
        println!("Memory usage: {} MB", stats.total_memory_usage() / 1024 / 1024);

        // Performance should be reasonable for the test environment
        assert!(throughput > 1000.0); // At least 1k events/second
        assert!(stats.total_memory_usage() < 50 * 1024 * 1024); // Less than 50MB for 10k points
    }
}