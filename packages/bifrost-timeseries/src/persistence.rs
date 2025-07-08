//! Memory-mapped persistence for time-series data

use crate::compression::{AdaptiveCompressor, CompressedData};
use crate::error::{Result, TimeSeriesError};
use crate::types::{DataPoint, TimeSeriesConfig, Timestamp};
use memmap2::{MmapMut, MmapOptions};
use serde::{Deserialize, Serialize};
use std::fs::{File, OpenOptions};
use std::io::{Read, Seek, SeekFrom, Write};
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};
use std::time::{SystemTime, UNIX_EPOCH};

/// Memory-mapped file header
#[derive(Debug, Clone, Serialize, Deserialize)]
struct FileHeader {
    /// Magic number for file format validation
    magic: u32,
    /// File format version
    version: u16,
    /// Header size in bytes
    header_size: u32,
    /// Total number of data points
    total_points: u64,
    /// Timestamp of first data point
    first_timestamp: Option<Timestamp>,
    /// Timestamp of last data point
    last_timestamp: Option<Timestamp>,
    /// Whether compression is enabled
    compression_enabled: bool,
    /// Compression level
    compression_level: i32,
    /// File creation timestamp
    created_at: u64,
    /// Last modified timestamp
    modified_at: u64,
    /// Data offset in file
    data_offset: u64,
    /// Checksum for integrity verification
    checksum: u64,
}

const MAGIC_NUMBER: u32 = 0x42495354; // "BIST" (Bifrost Time Series)
const FILE_VERSION: u16 = 1;
const MIN_FILE_SIZE: usize = 1024 * 1024; // 1MB minimum

impl Default for FileHeader {
    fn default() -> Self {
        let now = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();

        Self {
            magic: MAGIC_NUMBER,
            version: FILE_VERSION,
            header_size: std::mem::size_of::<FileHeader>() as u32,
            total_points: 0,
            first_timestamp: None,
            last_timestamp: None,
            compression_enabled: true,
            compression_level: 3,
            created_at: now,
            modified_at: now,
            data_offset: std::mem::size_of::<FileHeader>() as u64,
            checksum: 0,
        }
    }
}

impl FileHeader {
    /// Validate the file header
    fn validate(&self) -> Result<()> {
        if self.magic != MAGIC_NUMBER {
            return Err(TimeSeriesError::persistence(
                "Invalid file format: magic number mismatch",
            ));
        }
        if self.version != FILE_VERSION {
            return Err(TimeSeriesError::persistence(format!(
                "Unsupported file version: {}",
                self.version
            )));
        }
        Ok(())
    }

    /// Calculate checksum for integrity verification
    fn calculate_checksum(&self) -> u64 {
        // Simple checksum based on key fields
        let mut sum = 0u64;
        sum = sum.wrapping_add(self.magic as u64);
        sum = sum.wrapping_add(self.version as u64);
        sum = sum.wrapping_add(self.total_points);
        sum = sum.wrapping_add(self.created_at);
        sum
    }

    /// Update checksum
    fn update_checksum(&mut self) {
        self.checksum = self.calculate_checksum();
    }

    /// Verify checksum
    fn verify_checksum(&self) -> bool {
        self.checksum == self.calculate_checksum()
    }
}

/// Memory-mapped storage engine
#[derive(Debug)]
pub struct MmapStorage {
    /// Path to the storage file
    file_path: PathBuf,
    /// Memory-mapped file
    mmap: Arc<Mutex<Option<MmapMut>>>,
    /// File handle
    file: Arc<Mutex<File>>,
    /// Current file header
    header: Arc<Mutex<FileHeader>>,
    /// Compression engine
    compressor: AdaptiveCompressor,
    /// Current file size
    file_size: Arc<Mutex<usize>>,
    /// Data write offset
    write_offset: Arc<Mutex<u64>>,
}

impl MmapStorage {
    /// Create or open memory-mapped storage
    pub fn new<P: AsRef<Path>>(path: P, config: &TimeSeriesConfig) -> Result<Self> {
        let file_path = path.as_ref().to_path_buf();
        
        // Create parent directories if needed
        if let Some(parent) = file_path.parent() {
            std::fs::create_dir_all(parent)
                .map_err(|e| TimeSeriesError::persistence(format!("Failed to create directory: {}", e)))?;
        }

        let file_exists = file_path.exists();
        
        // Open or create file
        let file = OpenOptions::new()
            .read(true)
            .write(true)
            .create(true)
            .open(&file_path)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to open file: {}", e)))?;

        let mut storage = Self {
            file_path,
            mmap: Arc::new(Mutex::new(None)),
            file: Arc::new(Mutex::new(file)),
            header: Arc::new(Mutex::new(FileHeader::default())),
            compressor: AdaptiveCompressor::new(),
            file_size: Arc::new(Mutex::new(0)),
            write_offset: Arc::new(Mutex::new(0)),
        };

        if file_exists {
            storage.load_existing()?;
        } else {
            storage.initialize_new(config)?;
        }

        Ok(storage)
    }

    /// Initialize new storage file
    fn initialize_new(&mut self, config: &TimeSeriesConfig) -> Result<()> {
        let mut header = self.header.lock().unwrap();
        header.compression_enabled = config.enable_compression;
        header.compression_level = config.compression_level;
        header.update_checksum();

        // Ensure minimum file size
        let mut file = self.file.lock().unwrap();
        file.set_len(MIN_FILE_SIZE as u64)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to resize file: {}", e)))?;

        // Write header
        file.seek(SeekFrom::Start(0))
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to seek: {}", e)))?;
        
        let header_bytes = bincode::serialize(&*header)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to serialize header: {}", e)))?;
        
        file.write_all(&header_bytes)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to write header: {}", e)))?;

        file.flush()
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to flush: {}", e)))?;

        *self.file_size.lock().unwrap() = MIN_FILE_SIZE;
        *self.write_offset.lock().unwrap() = header.data_offset;

        // Create memory mapping
        self.create_mmap()?;

        Ok(())
    }

    /// Load existing storage file
    fn load_existing(&mut self) -> Result<()> {
        let mut file = self.file.lock().unwrap();
        
        // Read header
        file.seek(SeekFrom::Start(0))
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to seek: {}", e)))?;

        let mut header_bytes = vec![0u8; std::mem::size_of::<FileHeader>()];
        file.read_exact(&mut header_bytes)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to read header: {}", e)))?;

        let header: FileHeader = bincode::deserialize(&header_bytes)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to deserialize header: {}", e)))?;

        // Validate header
        header.validate()?;
        if !header.verify_checksum() {
            return Err(TimeSeriesError::persistence(
                "Header checksum verification failed",
            ));
        }

        let file_size = file.metadata()
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to get file metadata: {}", e)))?
            .len() as usize;

        *self.header.lock().unwrap() = header.clone();
        *self.file_size.lock().unwrap() = file_size;
        *self.write_offset.lock().unwrap() = self.find_write_offset()?;

        // Create memory mapping
        self.create_mmap()?;

        Ok(())
    }

    /// Create memory mapping for the file
    fn create_mmap(&self) -> Result<()> {
        let file = self.file.lock().unwrap();
        let file_size = *self.file_size.lock().unwrap();

        let mmap = unsafe {
            MmapOptions::new()
                .len(file_size)
                .map_mut(&*file)
                .map_err(|e| TimeSeriesError::persistence(format!("Failed to create memory map: {}", e)))?
        };

        *self.mmap.lock().unwrap() = Some(mmap);
        Ok(())
    }

    /// Find the current write offset by scanning the file
    fn find_write_offset(&self) -> Result<u64> {
        let header = self.header.lock().unwrap();
        let offset = header.data_offset;

        // For now, return data_offset. In a more sophisticated implementation,
        // we would scan the file to find the actual end of data
        Ok(offset)
    }

    /// Append data points to storage
    pub fn append_data_points(&self, data_points: &[DataPoint]) -> Result<()> {
        if data_points.is_empty() {
            return Ok(());
        }

        // Compress data if beneficial
        let compressed = self.compressor.compress_if_beneficial(data_points)?;
        
        // Create data block
        let data_block = DataBlock {
            timestamp: SystemTime::now()
                .duration_since(UNIX_EPOCH)
                .unwrap_or_default()
                .as_nanos() as u64,
            point_count: data_points.len() as u32,
            compressed_size: compressed.data.len() as u32,
            uncompressed_size: compressed.original_size as u32,
            is_compressed: compressed.is_compressed,
            checksum: Self::calculate_data_checksum(&compressed.data),
            data: compressed.data,
        };

        let block_bytes = bincode::serialize(&data_block)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to serialize data block: {}", e)))?;

        // Write to memory-mapped file
        self.write_data_block(&block_bytes)?;

        // Update header
        self.update_header_after_write(data_points)?;

        Ok(())
    }

    /// Write data block to memory-mapped file
    fn write_data_block(&self, block_bytes: &[u8]) -> Result<()> {
        let mut write_offset = self.write_offset.lock().unwrap();
        let mut file_size = self.file_size.lock().unwrap();

        // Check if we need to resize the file
        let required_size = *write_offset as usize + block_bytes.len();
        if required_size > *file_size {
            let new_size = (required_size * 2).max(*file_size * 2);
            self.resize_file(new_size)?;
            *file_size = new_size;
        }

        // Write to memory map
        let mut mmap_guard = self.mmap.lock().unwrap();
        if let Some(ref mut mmap) = *mmap_guard {
            let start = *write_offset as usize;
            let end = start + block_bytes.len();
            mmap[start..end].copy_from_slice(block_bytes);
            mmap.flush_range(start, block_bytes.len())
                .map_err(|e| TimeSeriesError::persistence(format!("Failed to flush memory map: {}", e)))?;
        }

        *write_offset += block_bytes.len() as u64;
        Ok(())
    }

    /// Resize the file and recreate memory mapping
    fn resize_file(&self, new_size: usize) -> Result<()> {
        // Close current memory mapping
        *self.mmap.lock().unwrap() = None;

        // Resize file
        let file = self.file.lock().unwrap();
        file.set_len(new_size as u64)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to resize file: {}", e)))?;
        drop(file);

        // Recreate memory mapping
        self.create_mmap()?;

        Ok(())
    }

    /// Update header after writing data
    fn update_header_after_write(&self, data_points: &[DataPoint]) -> Result<()> {
        let mut header = self.header.lock().unwrap();
        
        header.total_points += data_points.len() as u64;
        header.modified_at = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or_default()
            .as_secs();

        // Update timestamp range
        if let (Some(first), Some(last)) = (data_points.first(), data_points.last()) {
            if header.first_timestamp.is_none() || first.timestamp < header.first_timestamp.unwrap() {
                header.first_timestamp = Some(first.timestamp);
            }
            if header.last_timestamp.is_none() || last.timestamp > header.last_timestamp.unwrap() {
                header.last_timestamp = Some(last.timestamp);
            }
        }

        header.update_checksum();

        // Write updated header to file
        let header_bytes = bincode::serialize(&*header)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to serialize header: {}", e)))?;

        let mut mmap_guard = self.mmap.lock().unwrap();
        if let Some(ref mut mmap) = *mmap_guard {
            mmap[0..header_bytes.len()].copy_from_slice(&header_bytes);
            mmap.flush_range(0, header_bytes.len())
                .map_err(|e| TimeSeriesError::persistence(format!("Failed to flush header: {}", e)))?;
        }

        Ok(())
    }

    /// Calculate checksum for data
    fn calculate_data_checksum(data: &[u8]) -> u64 {
        // Simple CRC-like checksum
        let mut checksum = 0u64;
        for &byte in data {
            checksum = checksum.wrapping_mul(31).wrapping_add(byte as u64);
        }
        checksum
    }

    /// Read all data points from storage
    pub fn read_all_data_points(&self) -> Result<Vec<DataPoint>> {
        let header = self.header.lock().unwrap();
        let mut offset = header.data_offset;
        let mut all_points = Vec::new();

        let mmap_guard = self.mmap.lock().unwrap();
        if let Some(ref mmap) = *mmap_guard {
            while offset < *self.write_offset.lock().unwrap() {
                let (block, block_size) = self.read_data_block_at(mmap, offset)?;
                let points = self.compressor.decompress(&CompressedData {
                    data: block.data,
                    is_compressed: block.is_compressed,
                    original_size: block.uncompressed_size as usize,
                    compressed_size: block.compressed_size as usize,
                })?;
                
                all_points.extend(points);
                offset += block_size as u64;
            }
        }

        Ok(all_points)
    }

    /// Read data block at specific offset
    fn read_data_block_at(&self, mmap: &[u8], offset: u64) -> Result<(DataBlock, usize)> {
        if offset as usize >= mmap.len() {
            return Err(TimeSeriesError::persistence("Read offset beyond file size"));
        }

        // First, read the block header to get the size
        let block: DataBlock = bincode::deserialize(&mmap[offset as usize..])
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to deserialize data block: {}", e)))?;

        let block_size = bincode::serialized_size(&block)
            .map_err(|e| TimeSeriesError::persistence(format!("Failed to calculate block size: {}", e)))?;

        Ok((block, block_size as usize))
    }

    /// Get storage statistics
    pub fn stats(&self) -> Result<StorageStats> {
        let header = self.header.lock().unwrap();
        let file_size = *self.file_size.lock().unwrap();
        let write_offset = *self.write_offset.lock().unwrap();

        Ok(StorageStats {
            file_path: self.file_path.clone(),
            file_size,
            data_size: write_offset as usize,
            total_points: header.total_points,
            first_timestamp: header.first_timestamp,
            last_timestamp: header.last_timestamp,
            compression_enabled: header.compression_enabled,
            compression_level: header.compression_level,
            created_at: header.created_at,
            modified_at: header.modified_at,
        })
    }

    /// Flush all pending writes
    pub fn flush(&self) -> Result<()> {
        let mmap_guard = self.mmap.lock().unwrap();
        if let Some(ref mmap) = *mmap_guard {
            mmap.flush()
                .map_err(|e| TimeSeriesError::persistence(format!("Failed to flush memory map: {}", e)))?;
        }
        Ok(())
    }

    /// Close the storage and release resources
    pub fn close(&self) -> Result<()> {
        self.flush()?;
        *self.mmap.lock().unwrap() = None;
        Ok(())
    }
}

/// Data block stored in the file
#[derive(Debug, Serialize, Deserialize)]
struct DataBlock {
    /// Timestamp when block was written
    timestamp: u64,
    /// Number of data points in this block
    point_count: u32,
    /// Size of compressed data
    compressed_size: u32,
    /// Size of uncompressed data
    uncompressed_size: u32,
    /// Whether data is compressed
    is_compressed: bool,
    /// Checksum for integrity verification
    checksum: u64,
    /// The actual data (compressed or uncompressed)
    data: Vec<u8>,
}

/// Storage statistics
#[derive(Debug, Clone)]
pub struct StorageStats {
    pub file_path: PathBuf,
    pub file_size: usize,
    pub data_size: usize,
    pub total_points: u64,
    pub first_timestamp: Option<Timestamp>,
    pub last_timestamp: Option<Timestamp>,
    pub compression_enabled: bool,
    pub compression_level: i32,
    pub created_at: u64,
    pub modified_at: u64,
}

impl StorageStats {
    /// Get data utilization percentage
    pub fn utilization_percentage(&self) -> f64 {
        (self.data_size as f64 / self.file_size as f64) * 100.0
    }

    /// Get average bytes per data point
    pub fn avg_bytes_per_point(&self) -> f64 {
        if self.total_points > 0 {
            self.data_size as f64 / self.total_points as f64
        } else {
            0.0
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{TimeSeriesConfig, Value};
    use tempfile::TempDir;

    #[test]
    fn test_mmap_storage_basic() {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test.bts");
        let config = TimeSeriesConfig::default();

        let storage = MmapStorage::new(&file_path, &config).unwrap();

        // Write some data
        let data_points = vec![
            DataPoint::with_timestamp(1000, Value::Integer(1)),
            DataPoint::with_timestamp(2000, Value::Integer(2)),
            DataPoint::with_timestamp(3000, Value::Integer(3)),
        ];

        storage.append_data_points(&data_points).unwrap();
        storage.flush().unwrap();

        // Read back the data
        let read_points = storage.read_all_data_points().unwrap();
        assert_eq!(data_points, read_points);

        // Check stats
        let stats = storage.stats().unwrap();
        assert_eq!(stats.total_points, 3);
        assert_eq!(stats.first_timestamp, Some(1000));
        assert_eq!(stats.last_timestamp, Some(3000));
    }

    #[test]
    fn test_mmap_storage_persistence() {
        let temp_dir = TempDir::new().unwrap();
        let file_path = temp_dir.path().join("test_persist.bts");
        let config = TimeSeriesConfig::default();

        let data_points = vec![
            DataPoint::with_timestamp(1000, Value::String("test1".to_string())),
            DataPoint::with_timestamp(2000, Value::String("test2".to_string())),
        ];

        // Write data in first instance
        {
            let storage = MmapStorage::new(&file_path, &config).unwrap();
            storage.append_data_points(&data_points).unwrap();
            storage.flush().unwrap();
        }

        // Read data in second instance
        {
            let storage = MmapStorage::new(&file_path, &config).unwrap();
            let read_points = storage.read_all_data_points().unwrap();
            assert_eq!(data_points, read_points);
        }
    }
}