//! Indexing system for efficient time-series queries

use crate::types::{DataPoint, Timestamp};
use std::collections::{BTreeMap, HashMap};
use std::ops::Bound;

/// Time-based index for efficient range queries
#[derive(Debug, Clone)]
pub struct TimeIndex {
    /// Index mapping timestamp to data point positions
    index: BTreeMap<Timestamp, Vec<usize>>,
    /// Total number of indexed points
    total_points: usize,
}

impl TimeIndex {
    /// Create a new time index
    pub fn new() -> Self {
        Self {
            index: BTreeMap::new(),
            total_points: 0,
        }
    }

    /// Add a data point to the index
    pub fn add_point(&mut self, timestamp: Timestamp, position: usize) {
        self.index.entry(timestamp).or_insert_with(Vec::new).push(position);
        self.total_points += 1;
    }

    /// Add multiple data points to the index
    pub fn add_points(&mut self, points: &[(Timestamp, usize)]) {
        for &(timestamp, position) in points {
            self.add_point(timestamp, position);
        }
    }

    /// Get positions for exact timestamp
    pub fn get_exact(&self, timestamp: Timestamp) -> Option<&Vec<usize>> {
        self.index.get(&timestamp)
    }

    /// Get positions for timestamp range
    pub fn get_range(&self, start: Timestamp, end: Timestamp) -> Vec<usize> {
        let mut positions = Vec::new();
        
        for (_, pos_vec) in self.index.range(start..=end) {
            positions.extend_from_slice(pos_vec);
        }
        
        positions.sort_unstable();
        positions
    }

    /// Get positions for timestamp range with bounds
    pub fn get_range_bounded(
        &self,
        start: Bound<Timestamp>,
        end: Bound<Timestamp>,
    ) -> Vec<usize> {
        let mut positions = Vec::new();
        
        for (_, pos_vec) in self.index.range((start, end)) {
            positions.extend_from_slice(pos_vec);
        }
        
        positions.sort_unstable();
        positions
    }

    /// Get the first N positions after a timestamp
    pub fn get_first_after(&self, timestamp: Timestamp, count: usize) -> Vec<usize> {
        let mut positions = Vec::new();
        let mut collected = 0;
        
        for (_, pos_vec) in self.index.range(timestamp..) {
            for &pos in pos_vec {
                if collected >= count {
                    return positions;
                }
                positions.push(pos);
                collected += 1;
            }
        }
        
        positions
    }

    /// Get the last N positions before a timestamp
    pub fn get_last_before(&self, timestamp: Timestamp, count: usize) -> Vec<usize> {
        let mut positions = Vec::new();
        
        for (_, pos_vec) in self.index.range(..timestamp).rev() {
            for &pos in pos_vec.iter().rev() {
                if positions.len() >= count {
                    break;
                }
                positions.push(pos);
            }
            if positions.len() >= count {
                break;
            }
        }
        
        positions.reverse();
        positions
    }

    /// Get the minimum timestamp
    pub fn min_timestamp(&self) -> Option<Timestamp> {
        self.index.keys().next().copied()
    }

    /// Get the maximum timestamp
    pub fn max_timestamp(&self) -> Option<Timestamp> {
        self.index.keys().next_back().copied()
    }

    /// Get number of unique timestamps
    pub fn unique_timestamps(&self) -> usize {
        self.index.len()
    }

    /// Get total number of indexed points
    pub fn total_points(&self) -> usize {
        self.total_points
    }

    /// Remove points from index
    pub fn remove_points(&mut self, timestamps: &[Timestamp]) {
        for &timestamp in timestamps {
            if let Some(positions) = self.index.remove(&timestamp) {
                self.total_points = self.total_points.saturating_sub(positions.len());
            }
        }
    }

    /// Clear the entire index
    pub fn clear(&mut self) {
        self.index.clear();
        self.total_points = 0;
    }

    /// Get index statistics
    pub fn stats(&self) -> IndexStats {
        let memory_usage = self.index.len() * std::mem::size_of::<(Timestamp, Vec<usize>)>()
            + self.index.values().map(|v| v.capacity() * std::mem::size_of::<usize>()).sum::<usize>();

        IndexStats {
            unique_timestamps: self.index.len(),
            total_points: self.total_points,
            memory_usage,
            min_timestamp: self.min_timestamp(),
            max_timestamp: self.max_timestamp(),
        }
    }
}

impl Default for TimeIndex {
    fn default() -> Self {
        Self::new()
    }
}

/// Tag-based index for filtering by metadata
#[derive(Debug, Clone)]
pub struct TagIndex {
    /// Index mapping tag key-value pairs to data point positions
    index: HashMap<String, HashMap<String, Vec<usize>>>,
    /// Reverse index mapping positions to their tags
    reverse_index: HashMap<usize, HashMap<String, String>>,
}

impl TagIndex {
    /// Create a new tag index
    pub fn new() -> Self {
        Self {
            index: HashMap::new(),
            reverse_index: HashMap::new(),
        }
    }

    /// Add a data point with tags to the index
    pub fn add_point(&mut self, position: usize, tags: &HashMap<String, String>) {
        for (key, value) in tags {
            self.index
                .entry(key.clone())
                .or_insert_with(HashMap::new)
                .entry(value.clone())
                .or_insert_with(Vec::new)
                .push(position);
        }
        
        self.reverse_index.insert(position, tags.clone());
    }

    /// Get positions for a specific tag key-value pair
    pub fn get_by_tag(&self, key: &str, value: &str) -> Option<&Vec<usize>> {
        self.index.get(key)?.get(value)
    }

    /// Get positions matching all provided tags (AND operation)
    pub fn get_by_tags_and(&self, tags: &HashMap<String, String>) -> Vec<usize> {
        if tags.is_empty() {
            return Vec::new();
        }

        let mut result: Option<Vec<usize>> = None;

        for (key, value) in tags {
            if let Some(positions) = self.get_by_tag(key, value) {
                match result {
                    None => result = Some(positions.clone()),
                    Some(ref mut current) => {
                        // Intersection of current result and new positions
                        let mut new_result = Vec::new();
                        let mut i = 0;
                        let mut j = 0;
                        current.sort_unstable();
                        let mut sorted_positions = positions.clone();
                        sorted_positions.sort_unstable();

                        while i < current.len() && j < sorted_positions.len() {
                            if current[i] == sorted_positions[j] {
                                new_result.push(current[i]);
                                i += 1;
                                j += 1;
                            } else if current[i] < sorted_positions[j] {
                                i += 1;
                            } else {
                                j += 1;
                            }
                        }
                        *current = new_result;
                    }
                }
            } else {
                // Tag not found, so no points match
                return Vec::new();
            }
        }

        result.unwrap_or_default()
    }

    /// Get positions matching any of the provided tags (OR operation)
    pub fn get_by_tags_or(&self, tags: &HashMap<String, String>) -> Vec<usize> {
        let mut result = Vec::new();

        for (key, value) in tags {
            if let Some(positions) = self.get_by_tag(key, value) {
                result.extend_from_slice(positions);
            }
        }

        result.sort_unstable();
        result.dedup();
        result
    }

    /// Get all unique values for a tag key
    pub fn get_tag_values(&self, key: &str) -> Vec<String> {
        self.index
            .get(key)
            .map(|values| values.keys().cloned().collect())
            .unwrap_or_default()
    }

    /// Get all tag keys
    pub fn get_tag_keys(&self) -> Vec<String> {
        self.index.keys().cloned().collect()
    }

    /// Remove points from index
    pub fn remove_points(&mut self, positions: &[usize]) {
        for &position in positions {
            if let Some(tags) = self.reverse_index.remove(&position) {
                for (key, value) in tags {
                    if let Some(key_map) = self.index.get_mut(&key) {
                        if let Some(value_vec) = key_map.get_mut(&value) {
                            value_vec.retain(|&p| p != position);
                            if value_vec.is_empty() {
                                key_map.remove(&value);
                            }
                        }
                        if key_map.is_empty() {
                            self.index.remove(&key);
                        }
                    }
                }
            }
        }
    }

    /// Get total number of indexed points
    pub fn total_points(&self) -> usize {
        self.reverse_index.len()
    }

    /// Clear the entire index
    pub fn clear(&mut self) {
        self.index.clear();
        self.reverse_index.clear();
    }
}

impl Default for TagIndex {
    fn default() -> Self {
        Self::new()
    }
}

/// Combined index supporting both time and tag queries
#[derive(Debug)]
pub struct CombinedIndex {
    /// Time-based index
    time_index: TimeIndex,
    /// Tag-based index
    tag_index: TagIndex,
    /// Data points storage for position-based access
    data_points: Vec<DataPoint>,
}

impl CombinedIndex {
    /// Create a new combined index
    pub fn new() -> Self {
        Self {
            time_index: TimeIndex::new(),
            tag_index: TagIndex::new(),
            data_points: Vec::new(),
        }
    }

    /// Add a data point to the index
    pub fn add_point(&mut self, data_point: DataPoint) {
        let position = self.data_points.len();
        
        // Add to time index
        self.time_index.add_point(data_point.timestamp, position);
        
        // Add to tag index if tags exist
        if let Some(ref tags) = data_point.tags {
            self.tag_index.add_point(position, tags);
        }
        
        // Store the data point
        self.data_points.push(data_point);
    }

    /// Add multiple data points to the index
    pub fn add_points(&mut self, data_points: Vec<DataPoint>) {
        for data_point in data_points {
            self.add_point(data_point);
        }
    }

    /// Query by time range only
    pub fn query_time_range(&self, start: Timestamp, end: Timestamp) -> Vec<&DataPoint> {
        let positions = self.time_index.get_range(start, end);
        positions.iter().map(|&pos| &self.data_points[pos]).collect()
    }

    /// Query by tags only
    pub fn query_tags(&self, tags: &HashMap<String, String>, use_and: bool) -> Vec<&DataPoint> {
        let positions = if use_and {
            self.tag_index.get_by_tags_and(tags)
        } else {
            self.tag_index.get_by_tags_or(tags)
        };
        
        positions.iter().map(|&pos| &self.data_points[pos]).collect()
    }

    /// Query by both time range and tags
    pub fn query_combined(
        &self,
        start: Timestamp,
        end: Timestamp,
        tags: &HashMap<String, String>,
        use_and: bool,
    ) -> Vec<&DataPoint> {
        let time_positions = self.time_index.get_range(start, end);
        let tag_positions = if use_and {
            self.tag_index.get_by_tags_and(tags)
        } else {
            self.tag_index.get_by_tags_or(tags)
        };

        // Intersection of time and tag positions
        let mut result_positions = Vec::new();
        let mut i = 0;
        let mut j = 0;
        let mut sorted_time = time_positions;
        let mut sorted_tags = tag_positions;
        sorted_time.sort_unstable();
        sorted_tags.sort_unstable();

        while i < sorted_time.len() && j < sorted_tags.len() {
            if sorted_time[i] == sorted_tags[j] {
                result_positions.push(sorted_time[i]);
                i += 1;
                j += 1;
            } else if sorted_time[i] < sorted_tags[j] {
                i += 1;
            } else {
                j += 1;
            }
        }

        result_positions.iter().map(|&pos| &self.data_points[pos]).collect()
    }

    /// Get the latest N data points
    pub fn get_latest(&self, count: usize) -> Vec<&DataPoint> {
        if let Some(max_timestamp) = self.time_index.max_timestamp() {
            let positions = self.time_index.get_last_before(max_timestamp + 1, count);
            positions.iter().map(|&pos| &self.data_points[pos]).collect()
        } else {
            Vec::new()
        }
    }

    /// Get total number of data points
    pub fn len(&self) -> usize {
        self.data_points.len()
    }

    /// Check if index is empty
    pub fn is_empty(&self) -> bool {
        self.data_points.is_empty()
    }

    /// Get index statistics
    pub fn stats(&self) -> CombinedIndexStats {
        CombinedIndexStats {
            time_stats: self.time_index.stats(),
            total_data_points: self.data_points.len(),
            unique_tag_keys: self.tag_index.get_tag_keys().len(),
            memory_usage: self.estimate_memory_usage(),
        }
    }

    /// Estimate total memory usage
    fn estimate_memory_usage(&self) -> usize {
        let time_index_size = self.time_index.stats().memory_usage;
        let data_points_size = self.data_points.capacity() * std::mem::size_of::<DataPoint>();
        let tag_index_size = self.tag_index.total_points() * 64; // Rough estimate
        
        time_index_size + data_points_size + tag_index_size
    }

    /// Clear all indexes and data
    pub fn clear(&mut self) {
        self.time_index.clear();
        self.tag_index.clear();
        self.data_points.clear();
    }
}

impl Default for CombinedIndex {
    fn default() -> Self {
        Self::new()
    }
}

/// Index statistics
#[derive(Debug, Clone)]
pub struct IndexStats {
    pub unique_timestamps: usize,
    pub total_points: usize,
    pub memory_usage: usize,
    pub min_timestamp: Option<Timestamp>,
    pub max_timestamp: Option<Timestamp>,
}

/// Combined index statistics
#[derive(Debug, Clone)]
pub struct CombinedIndexStats {
    pub time_stats: IndexStats,
    pub total_data_points: usize,
    pub unique_tag_keys: usize,
    pub memory_usage: usize,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{DataPoint, Value};

    #[test]
    fn test_time_index_basic() {
        let mut index = TimeIndex::new();
        
        // Add some points
        index.add_point(1000, 0);
        index.add_point(2000, 1);
        index.add_point(3000, 2);
        index.add_point(2000, 3); // Duplicate timestamp
        
        assert_eq!(index.total_points(), 4);
        assert_eq!(index.unique_timestamps(), 3);
        
        // Test exact lookup
        assert_eq!(index.get_exact(2000), Some(&vec![1, 3]));
        
        // Test range query
        let range_result = index.get_range(1500, 2500);
        assert_eq!(range_result, vec![1, 3]);
        
        // Test min/max
        assert_eq!(index.min_timestamp(), Some(1000));
        assert_eq!(index.max_timestamp(), Some(3000));
    }

    #[test]
    fn test_tag_index_basic() {
        let mut index = TagIndex::new();
        
        let mut tags1 = HashMap::new();
        tags1.insert("device".to_string(), "sensor1".to_string());
        tags1.insert("location".to_string(), "room1".to_string());
        
        let mut tags2 = HashMap::new();
        tags2.insert("device".to_string(), "sensor2".to_string());
        tags2.insert("location".to_string(), "room1".to_string());
        
        let mut tags3 = HashMap::new();
        tags3.insert("device".to_string(), "sensor1".to_string());
        tags3.insert("location".to_string(), "room2".to_string());
        
        index.add_point(0, &tags1);
        index.add_point(1, &tags2);
        index.add_point(2, &tags3);
        
        // Test single tag query
        assert_eq!(index.get_by_tag("device", "sensor1"), Some(&vec![0, 2]));
        assert_eq!(index.get_by_tag("location", "room1"), Some(&vec![0, 1]));
        
        // Test AND query
        let mut query_tags = HashMap::new();
        query_tags.insert("device".to_string(), "sensor1".to_string());
        query_tags.insert("location".to_string(), "room1".to_string());
        let and_result = index.get_by_tags_and(&query_tags);
        assert_eq!(and_result, vec![0]);
        
        // Test OR query
        query_tags.clear();
        query_tags.insert("device".to_string(), "sensor1".to_string());
        query_tags.insert("location".to_string(), "room2".to_string());
        let or_result = index.get_by_tags_or(&query_tags);
        assert!(or_result.contains(&0) && or_result.contains(&2));
    }

    #[test]
    fn test_combined_index() {
        let mut index = CombinedIndex::new();
        
        // Create test data points
        let mut tags1 = HashMap::new();
        tags1.insert("device".to_string(), "sensor1".to_string());
        
        let mut tags2 = HashMap::new();
        tags2.insert("device".to_string(), "sensor2".to_string());
        
        let dp1 = DataPoint::with_tags(1000, Value::Integer(1), tags1);
        let dp2 = DataPoint::with_tags(2000, Value::Integer(2), tags2);
        let dp3 = DataPoint::with_timestamp(3000, Value::Integer(3));
        
        index.add_point(dp1);
        index.add_point(dp2);
        index.add_point(dp3);
        
        // Test time range query
        let time_result = index.query_time_range(1500, 2500);
        assert_eq!(time_result.len(), 1);
        assert_eq!(time_result[0].timestamp, 2000);
        
        // Test tag query
        let mut query_tags = HashMap::new();
        query_tags.insert("device".to_string(), "sensor1".to_string());
        let tag_result = index.query_tags(&query_tags, true);
        assert_eq!(tag_result.len(), 1);
        assert_eq!(tag_result[0].timestamp, 1000);
        
        // Test combined query
        query_tags.insert("device".to_string(), "sensor2".to_string());
        let combined_result = index.query_combined(1500, 2500, &query_tags, false);
        assert_eq!(combined_result.len(), 1);
        assert_eq!(combined_result[0].timestamp, 2000);
    }
}