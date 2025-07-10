//! Compression utilities for time-series data using zstd

use crate::error::{Result, TimeSeriesError};
use crate::types::DataPoint;
use serde::{Deserialize, Serialize};
use std::io::{Read, Write};

/// Compression engine using zstd
#[derive(Debug)]
pub struct ZstdCompressor {
    /// Compression level (1-22)
    level: i32,
}

impl ZstdCompressor {
    /// Create a new compressor with default level (3)
    pub fn new() -> Self {
        Self { level: 3 }
    }

    /// Create a new compressor with specified level
    pub fn with_level(level: i32) -> Result<Self> {
        if level < 1 || level > 22 {
            return Err(TimeSeriesError::configuration(
                "Compression level must be between 1 and 22",
            ));
        }
        Ok(Self { level })
    }

    /// Compress a single data point
    pub fn compress_data_point(&self, data_point: &DataPoint) -> Result<Vec<u8>> {
        let serialized = bincode::serialize(data_point)
            .map_err(|e| TimeSeriesError::configuration(format!("Serialization failed: {}", e)))?;
        
        self.compress_bytes(&serialized)
    }

    /// Decompress a data point
    pub fn decompress_data_point(&self, compressed: &[u8]) -> Result<DataPoint> {
        let decompressed = self.decompress_bytes(compressed)?;
        bincode::deserialize(&decompressed)
            .map_err(|e| TimeSeriesError::configuration(format!("Deserialization failed: {}", e)))
    }

    /// Compress a batch of data points
    pub fn compress_batch(&self, data_points: &[DataPoint]) -> Result<Vec<u8>> {
        let batch = CompressedBatch {
            data_points: data_points.to_vec(),
            compression_level: self.level,
            uncompressed_size: data_points.iter().map(|dp| dp.size_bytes()).sum(),
        };

        let serialized = bincode::serialize(&batch)
            .map_err(|e| TimeSeriesError::configuration(format!("Batch serialization failed: {}", e)))?;
        
        self.compress_bytes(&serialized)
    }

    /// Decompress a batch of data points
    pub fn decompress_batch(&self, compressed: &[u8]) -> Result<Vec<DataPoint>> {
        let decompressed = self.decompress_bytes(compressed)?;
        let batch: CompressedBatch = bincode::deserialize(&decompressed)
            .map_err(|e| TimeSeriesError::configuration(format!("Batch deserialization failed: {}", e)))?;
        
        Ok(batch.data_points)
    }

    /// Compress raw bytes
    pub fn compress_bytes(&self, data: &[u8]) -> Result<Vec<u8>> {
        let mut encoder = zstd::Encoder::new(Vec::new(), self.level)
            .map_err(|e| TimeSeriesError::compression(format!("Failed to create encoder: {}", e)))?;
        encoder.write_all(data)
            .map_err(|e| TimeSeriesError::compression(format!("Failed to write data: {}", e)))?;
        let compressed = encoder.finish()
            .map_err(|e| TimeSeriesError::compression(format!("Failed to finish encoding: {}", e)))?;
        Ok(compressed)
    }

    /// Decompress raw bytes
    pub fn decompress_bytes(&self, compressed: &[u8]) -> Result<Vec<u8>> {
        let mut decoder = zstd::Decoder::new(compressed)
            .map_err(|e| TimeSeriesError::compression(format!("Failed to create decoder: {}", e)))?;
        let mut decompressed = Vec::new();
        decoder.read_to_end(&mut decompressed)
            .map_err(|e| TimeSeriesError::compression(format!("Failed to read decompressed data: {}", e)))?;
        Ok(decompressed)
    }

    /// Get compression ratio for given data
    pub fn compression_ratio(&self, original: &[u8]) -> Result<f64> {
        let compressed = self.compress_bytes(original)?;
        Ok(compressed.len() as f64 / original.len() as f64)
    }

    /// Estimate compression savings for a batch
    pub fn estimate_batch_savings(&self, data_points: &[DataPoint]) -> Result<CompressionStats> {
        let original_serialized = bincode::serialize(data_points)
            .map_err(|e| TimeSeriesError::configuration(format!("Serialization failed: {}", e)))?;
        
        let compressed = self.compress_bytes(&original_serialized)?;
        
        Ok(CompressionStats {
            original_size: original_serialized.len(),
            compressed_size: compressed.len(),
            compression_ratio: compressed.len() as f64 / original_serialized.len() as f64,
            space_saved: original_serialized.len().saturating_sub(compressed.len()),
        })
    }
}

impl Default for ZstdCompressor {
    fn default() -> Self {
        Self::new()
    }
}

/// A batch of compressed data points
#[derive(Debug, Serialize, Deserialize)]
struct CompressedBatch {
    data_points: Vec<DataPoint>,
    compression_level: i32,
    uncompressed_size: usize,
}

/// Compression statistics
#[derive(Debug, Clone)]
pub struct CompressionStats {
    pub original_size: usize,
    pub compressed_size: usize,
    pub compression_ratio: f64,
    pub space_saved: usize,
}

impl CompressionStats {
    /// Get compression percentage (0-100)
    pub fn compression_percentage(&self) -> f64 {
        (1.0 - self.compression_ratio) * 100.0
    }

    /// Check if compression is beneficial
    pub fn is_beneficial(&self) -> bool {
        self.compression_ratio < 0.9 // At least 10% savings
    }
}

/// Adaptive compressor that chooses compression based on data characteristics
#[derive(Debug)]
pub struct AdaptiveCompressor {
    compressor: ZstdCompressor,
    /// Minimum size threshold for compression
    min_size_threshold: usize,
    /// Minimum compression ratio to apply compression
    min_compression_ratio: f64,
}

impl AdaptiveCompressor {
    /// Create a new adaptive compressor
    pub fn new() -> Self {
        Self {
            compressor: ZstdCompressor::new(),
            min_size_threshold: 1024, // 1KB
            min_compression_ratio: 0.8, // At least 20% savings
        }
    }

    /// Create with custom thresholds
    pub fn with_thresholds(min_size: usize, min_ratio: f64) -> Self {
        Self {
            compressor: ZstdCompressor::new(),
            min_size_threshold: min_size,
            min_compression_ratio: min_ratio,
        }
    }

    /// Compress data only if beneficial
    pub fn compress_if_beneficial(&self, data_points: &[DataPoint]) -> Result<CompressedData> {
        let serialized = bincode::serialize(data_points)
            .map_err(|e| TimeSeriesError::configuration(format!("Serialization failed: {}", e)))?;

        // Skip compression if data is too small
        if serialized.len() < self.min_size_threshold {
            return Ok(CompressedData {
                data: serialized.clone(),
                is_compressed: false,
                original_size: serialized.len(),
                compressed_size: serialized.len(),
            });
        }

        // Try compression and check if beneficial
        let compressed = self.compressor.compress_bytes(&serialized)?;
        let ratio = compressed.len() as f64 / serialized.len() as f64;

        if ratio < self.min_compression_ratio {
            Ok(CompressedData {
                data: compressed.clone(),
                is_compressed: true,
                original_size: serialized.len(),
                compressed_size: compressed.len(),
            })
        } else {
            Ok(CompressedData {
                data: serialized.clone(),
                is_compressed: false,
                original_size: serialized.len(),
                compressed_size: serialized.len(),
            })
        }
    }

    /// Decompress data
    pub fn decompress(&self, compressed_data: &CompressedData) -> Result<Vec<DataPoint>> {
        let decompressed_bytes = if compressed_data.is_compressed {
            self.compressor.decompress_bytes(&compressed_data.data)?
        } else {
            compressed_data.data.clone()
        };

        bincode::deserialize(&decompressed_bytes)
            .map_err(|e| TimeSeriesError::configuration(format!("Deserialization failed: {}", e)))
    }
}

impl Default for AdaptiveCompressor {
    fn default() -> Self {
        Self::new()
    }
}

/// Compressed data container
#[derive(Debug, Clone)]
pub struct CompressedData {
    pub data: Vec<u8>,
    pub is_compressed: bool,
    pub original_size: usize,
    pub compressed_size: usize,
}

impl CompressedData {
    /// Get compression ratio
    pub fn compression_ratio(&self) -> f64 {
        self.compressed_size as f64 / self.original_size as f64
    }

    /// Get space saved in bytes
    pub fn space_saved(&self) -> usize {
        self.original_size.saturating_sub(self.compressed_size)
    }

    /// Get compression percentage
    pub fn compression_percentage(&self) -> f64 {
        if self.is_compressed {
            (1.0 - self.compression_ratio()) * 100.0
        } else {
            0.0
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{DataPoint, Value};

    #[test]
    fn test_zstd_compression_basic() {
        let compressor = ZstdCompressor::new();
        // Use larger, more repetitive data for compression
        let data = b"Hello, World! This is a test string for compression. ".repeat(10);
        
        let compressed = compressor.compress_bytes(&data).unwrap();
        let decompressed = compressor.decompress_bytes(&compressed).unwrap();
        
        assert_eq!(data, decompressed.as_slice());
        assert!(compressed.len() < data.len()); // Should be smaller with repetitive data
    }

    #[test]
    fn test_data_point_compression() {
        let compressor = ZstdCompressor::new();
        let dp = DataPoint::with_timestamp(1000, Value::String("Test data".to_string()));
        
        let compressed = compressor.compress_data_point(&dp).unwrap();
        let decompressed = compressor.decompress_data_point(&compressed).unwrap();
        
        assert_eq!(dp, decompressed);
    }

    #[test]
    fn test_batch_compression() {
        let compressor = ZstdCompressor::new();
        let data_points = vec![
            DataPoint::with_timestamp(1000, Value::Integer(1)),
            DataPoint::with_timestamp(2000, Value::Integer(2)),
            DataPoint::with_timestamp(3000, Value::Integer(3)),
        ];
        
        let compressed = compressor.compress_batch(&data_points).unwrap();
        let decompressed = compressor.decompress_batch(&compressed).unwrap();
        
        assert_eq!(data_points, decompressed);
    }

    #[test]
    fn test_compression_stats() {
        let compressor = ZstdCompressor::new();
        let data_points = vec![
            DataPoint::with_timestamp(1000, Value::String("A".repeat(100))),
            DataPoint::with_timestamp(2000, Value::String("B".repeat(100))),
            DataPoint::with_timestamp(3000, Value::String("C".repeat(100))),
        ];
        
        let stats = compressor.estimate_batch_savings(&data_points).unwrap();
        
        assert!(stats.compression_ratio < 1.0);
        assert!(stats.space_saved > 0);
        assert!(stats.is_beneficial());
    }

    #[test]
    fn test_adaptive_compression() {
        let compressor = AdaptiveCompressor::new();
        
        // Small data - should not compress
        let small_data = vec![DataPoint::with_timestamp(1000, Value::Integer(1))];
        let result = compressor.compress_if_beneficial(&small_data).unwrap();
        assert!(!result.is_compressed);
        
        // Large repetitive data - should compress
        let large_data: Vec<DataPoint> = (0..100)
            .map(|i| DataPoint::with_timestamp(i * 1000, Value::String("A".repeat(50))))
            .collect();
        let result = compressor.compress_if_beneficial(&large_data).unwrap();
        assert!(result.is_compressed);
        
        // Verify decompression works
        let decompressed = compressor.decompress(&result).unwrap();
        assert_eq!(large_data, decompressed);
    }
}