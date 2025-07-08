//! Query engine for time-series data with aggregation support

use crate::error::{Result, TimeSeriesError};
use crate::index::CombinedIndex;
use crate::types::{AggregationResult, AggregationType, DataPoint, Timestamp, Value};
use std::collections::HashMap;

/// Query builder for constructing time-series queries
#[derive(Debug, Clone)]
pub struct QueryBuilder {
    /// Start timestamp for the query
    start_time: Option<Timestamp>,
    /// End timestamp for the query
    end_time: Option<Timestamp>,
    /// Tags to filter by
    tags: HashMap<String, String>,
    /// Whether to use AND or OR for tag filtering
    tag_and_operation: bool,
    /// Limit on number of results
    limit: Option<usize>,
    /// Aggregation type
    aggregation: Option<AggregationType>,
    /// Group by interval in nanoseconds
    group_by_interval: Option<i64>,
}

impl QueryBuilder {
    /// Create a new query builder
    pub fn new() -> Self {
        Self {
            start_time: None,
            end_time: None,
            tags: HashMap::new(),
            tag_and_operation: true,
            limit: None,
            aggregation: None,
            group_by_interval: None,
        }
    }

    /// Set the start time for the query
    pub fn start_time(mut self, timestamp: Timestamp) -> Self {
        self.start_time = Some(timestamp);
        self
    }

    /// Set the end time for the query
    pub fn end_time(mut self, timestamp: Timestamp) -> Self {
        self.end_time = Some(timestamp);
        self
    }

    /// Set both start and end times
    pub fn time_range(mut self, start: Timestamp, end: Timestamp) -> Self {
        self.start_time = Some(start);
        self.end_time = Some(end);
        self
    }

    /// Add a tag filter
    pub fn tag(mut self, key: String, value: String) -> Self {
        self.tags.insert(key, value);
        self
    }

    /// Add multiple tag filters
    pub fn tags(mut self, tags: HashMap<String, String>) -> Self {
        self.tags.extend(tags);
        self
    }

    /// Set tag operation to AND (all tags must match)
    pub fn tags_and(mut self) -> Self {
        self.tag_and_operation = true;
        self
    }

    /// Set tag operation to OR (any tag can match)
    pub fn tags_or(mut self) -> Self {
        self.tag_and_operation = false;
        self
    }

    /// Set result limit
    pub fn limit(mut self, count: usize) -> Self {
        self.limit = Some(count);
        self
    }

    /// Set aggregation type
    pub fn aggregate(mut self, aggregation: AggregationType) -> Self {
        self.aggregation = Some(aggregation);
        self
    }

    /// Set group by time interval
    pub fn group_by_interval(mut self, interval_nanos: i64) -> Self {
        self.group_by_interval = Some(interval_nanos);
        self
    }

    /// Execute the query against an index
    pub fn execute(&self, index: &CombinedIndex) -> Result<QueryResult> {
        // Build the data point list based on filters
        let data_points = self.execute_filters(index)?;

        // Apply aggregation if specified
        if let Some(agg_type) = self.aggregation {
            if let Some(interval) = self.group_by_interval {
                self.execute_grouped_aggregation(&data_points, agg_type, interval)
            } else {
                self.execute_simple_aggregation(&data_points, agg_type)
            }
        } else {
            // Return raw data points
            let mut result_points: Vec<DataPoint> = data_points.iter().map(|&dp| dp.clone()).collect();
            
            // Apply limit if specified
            if let Some(limit) = self.limit {
                result_points.truncate(limit);
            }

            Ok(QueryResult::DataPoints(result_points))
        }
    }

    /// Execute filters to get matching data points
    fn execute_filters<'a>(&self, index: &'a CombinedIndex) -> Result<Vec<&'a DataPoint>> {
        match (self.start_time, self.end_time) {
            (Some(start), Some(end)) => {
                if self.tags.is_empty() {
                    Ok(index.query_time_range(start, end))
                } else {
                    Ok(index.query_combined(start, end, &self.tags, self.tag_and_operation))
                }
            }
            (Some(start), None) => {
                let max_time = Timestamp::MAX;
                if self.tags.is_empty() {
                    Ok(index.query_time_range(start, max_time))
                } else {
                    Ok(index.query_combined(start, max_time, &self.tags, self.tag_and_operation))
                }
            }
            (None, Some(end)) => {
                let min_time = Timestamp::MIN;
                if self.tags.is_empty() {
                    Ok(index.query_time_range(min_time, end))
                } else {
                    Ok(index.query_combined(min_time, end, &self.tags, self.tag_and_operation))
                }
            }
            (None, None) => {
                if self.tags.is_empty() {
                    // No filters, return all data or latest if limit is set
                    if let Some(limit) = self.limit {
                        Ok(index.get_latest(limit))
                    } else {
                        // This could be expensive for large datasets
                        Ok(index.query_time_range(Timestamp::MIN, Timestamp::MAX))
                    }
                } else {
                    Ok(index.query_tags(&self.tags, self.tag_and_operation))
                }
            }
        }
    }

    /// Execute simple aggregation (single result)
    fn execute_simple_aggregation(
        &self,
        data_points: &[&DataPoint],
        agg_type: AggregationType,
    ) -> Result<QueryResult> {
        if data_points.is_empty() {
            return Ok(QueryResult::Aggregations(vec![AggregationResult {
                aggregation: agg_type,
                value: None,
                count: 0,
                start_timestamp: self.start_time.unwrap_or(0),
                end_timestamp: self.end_time.unwrap_or(0),
            }]));
        }

        let start_timestamp = self.start_time.unwrap_or_else(|| {
            data_points.iter().map(|dp| dp.timestamp).min().unwrap_or(0)
        });
        let end_timestamp = self.end_time.unwrap_or_else(|| {
            data_points.iter().map(|dp| dp.timestamp).max().unwrap_or(0)
        });

        let aggregated_value = self.calculate_aggregation(data_points, agg_type)?;

        Ok(QueryResult::Aggregations(vec![AggregationResult {
            aggregation: agg_type,
            value: aggregated_value,
            count: data_points.len(),
            start_timestamp,
            end_timestamp,
        }]))
    }

    /// Execute grouped aggregation (multiple results by time intervals)
    fn execute_grouped_aggregation(
        &self,
        data_points: &[&DataPoint],
        agg_type: AggregationType,
        interval_nanos: i64,
    ) -> Result<QueryResult> {
        if data_points.is_empty() {
            return Ok(QueryResult::Aggregations(vec![]));
        }

        // Group data points by time intervals
        let mut groups: HashMap<i64, Vec<&DataPoint>> = HashMap::new();

        for &data_point in data_points {
            let bucket = data_point.timestamp / interval_nanos;
            groups.entry(bucket).or_insert_with(Vec::new).push(data_point);
        }

        // Calculate aggregation for each group
        let mut results = Vec::new();
        for (&bucket, group_points) in &groups {
            let start_timestamp = bucket * interval_nanos;
            let end_timestamp = (bucket + 1) * interval_nanos - 1;

            let aggregated_value = self.calculate_aggregation(group_points, agg_type)?;

            results.push(AggregationResult {
                aggregation: agg_type,
                value: aggregated_value,
                count: group_points.len(),
                start_timestamp,
                end_timestamp,
            });
        }

        // Sort results by start timestamp
        results.sort_by_key(|r| r.start_timestamp);

        Ok(QueryResult::Aggregations(results))
    }

    /// Calculate aggregation for a group of data points
    fn calculate_aggregation(
        &self,
        data_points: &[&DataPoint],
        agg_type: AggregationType,
    ) -> Result<Option<Value>> {
        if data_points.is_empty() {
            return Ok(None);
        }

        match agg_type {
            AggregationType::Count => Ok(Some(Value::Integer(data_points.len() as i64))),
            
            AggregationType::First => Ok(Some(data_points[0].value.clone())),
            
            AggregationType::Last => Ok(Some(data_points[data_points.len() - 1].value.clone())),
            
            AggregationType::Min => {
                let min_value = data_points
                    .iter()
                    .filter_map(|dp| self.extract_numeric_value(&dp.value))
                    .min_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
                
                Ok(min_value.map(Value::Float))
            }
            
            AggregationType::Max => {
                let max_value = data_points
                    .iter()
                    .filter_map(|dp| self.extract_numeric_value(&dp.value))
                    .max_by(|a, b| a.partial_cmp(b).unwrap_or(std::cmp::Ordering::Equal));
                
                Ok(max_value.map(Value::Float))
            }
            
            AggregationType::Sum => {
                let sum: f64 = data_points
                    .iter()
                    .filter_map(|dp| self.extract_numeric_value(&dp.value))
                    .sum();
                
                Ok(Some(Value::Float(sum)))
            }
            
            AggregationType::Average => {
                let numeric_values: Vec<f64> = data_points
                    .iter()
                    .filter_map(|dp| self.extract_numeric_value(&dp.value))
                    .collect();
                
                if numeric_values.is_empty() {
                    Ok(None)
                } else {
                    let avg = numeric_values.iter().sum::<f64>() / numeric_values.len() as f64;
                    Ok(Some(Value::Float(avg)))
                }
            }
        }
    }

    /// Extract numeric value from a Value enum
    fn extract_numeric_value(&self, value: &Value) -> Option<f64> {
        match value {
            Value::Integer(i) => Some(*i as f64),
            Value::Float(f) => Some(*f),
            Value::Boolean(b) => Some(if *b { 1.0 } else { 0.0 }),
            _ => None,
        }
    }
}

impl Default for QueryBuilder {
    fn default() -> Self {
        Self::new()
    }
}

/// Query result containing either raw data points or aggregated results
#[derive(Debug, Clone)]
pub enum QueryResult {
    /// Raw data points
    DataPoints(Vec<DataPoint>),
    /// Aggregated results
    Aggregations(Vec<AggregationResult>),
}

impl QueryResult {
    /// Get the number of results
    pub fn len(&self) -> usize {
        match self {
            QueryResult::DataPoints(points) => points.len(),
            QueryResult::Aggregations(aggs) => aggs.len(),
        }
    }

    /// Check if result is empty
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }

    /// Convert to data points if possible
    pub fn to_data_points(self) -> Option<Vec<DataPoint>> {
        match self {
            QueryResult::DataPoints(points) => Some(points),
            QueryResult::Aggregations(_) => None,
        }
    }

    /// Convert to aggregations if possible
    pub fn to_aggregations(self) -> Option<Vec<AggregationResult>> {
        match self {
            QueryResult::DataPoints(_) => None,
            QueryResult::Aggregations(aggs) => Some(aggs),
        }
    }
}

/// Query engine for executing complex queries
#[derive(Debug)]
pub struct QueryEngine {
    /// Combined index for efficient queries
    index: CombinedIndex,
}

impl QueryEngine {
    /// Create a new query engine
    pub fn new() -> Self {
        Self {
            index: CombinedIndex::new(),
        }
    }

    /// Create query engine with existing index
    pub fn with_index(index: CombinedIndex) -> Self {
        Self { index }
    }

    /// Add data points to the engine
    pub fn add_data_points(&mut self, data_points: Vec<DataPoint>) {
        self.index.add_points(data_points);
    }

    /// Add a single data point to the engine
    pub fn add_data_point(&mut self, data_point: DataPoint) {
        self.index.add_point(data_point);
    }

    /// Execute a query
    pub fn execute_query(&self, query: &QueryBuilder) -> Result<QueryResult> {
        query.execute(&self.index)
    }

    /// Create a new query builder
    pub fn query(&self) -> QueryBuilder {
        QueryBuilder::new()
    }

    /// Get the latest N data points
    pub fn get_latest(&self, count: usize) -> Vec<DataPoint> {
        self.index.get_latest(count).into_iter().cloned().collect()
    }

    /// Get all data points in a time range
    pub fn get_time_range(&self, start: Timestamp, end: Timestamp) -> Vec<DataPoint> {
        self.index.query_time_range(start, end).into_iter().cloned().collect()
    }

    /// Get engine statistics
    pub fn stats(&self) -> QueryEngineStats {
        let index_stats = self.index.stats();
        QueryEngineStats {
            total_data_points: index_stats.total_data_points,
            unique_timestamps: index_stats.time_stats.unique_timestamps,
            unique_tag_keys: index_stats.unique_tag_keys,
            memory_usage: index_stats.memory_usage,
            min_timestamp: index_stats.time_stats.min_timestamp,
            max_timestamp: index_stats.time_stats.max_timestamp,
        }
    }

    /// Clear all data from the engine
    pub fn clear(&mut self) {
        self.index.clear();
    }
}

impl Default for QueryEngine {
    fn default() -> Self {
        Self::new()
    }
}

/// Query engine statistics
#[derive(Debug, Clone)]
pub struct QueryEngineStats {
    pub total_data_points: usize,
    pub unique_timestamps: usize,
    pub unique_tag_keys: usize,
    pub memory_usage: usize,
    pub min_timestamp: Option<Timestamp>,
    pub max_timestamp: Option<Timestamp>,
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::{DataPoint, Value};

    fn create_test_data() -> Vec<DataPoint> {
        let mut data = Vec::new();
        
        // Create test data with different timestamps and values
        for i in 0..10 {
            let mut tags = HashMap::new();
            tags.insert("device".to_string(), format!("sensor{}", i % 3));
            tags.insert("location".to_string(), if i < 5 { "room1".to_string() } else { "room2".to_string() });
            
            data.push(DataPoint::with_tags(
                i * 1000,
                Value::Float(i as f64 * 10.0),
                tags,
            ));
        }
        
        data
    }

    #[test]
    fn test_query_builder_basic() {
        let mut engine = QueryEngine::new();
        let test_data = create_test_data();
        engine.add_data_points(test_data);

        // Test time range query
        let result = engine
            .query()
            .time_range(2000, 6000)
            .execute(&engine.index)
            .unwrap();

        if let QueryResult::DataPoints(points) = result {
            assert_eq!(points.len(), 5); // timestamps 2000, 3000, 4000, 5000, 6000
        } else {
            panic!("Expected DataPoints result");
        }
    }

    #[test]
    fn test_query_with_tags() {
        let mut engine = QueryEngine::new();
        let test_data = create_test_data();
        engine.add_data_points(test_data);

        // Test tag query
        let mut tags = HashMap::new();
        tags.insert("device".to_string(), "sensor1".to_string());

        let result = engine
            .query()
            .tags(tags)
            .execute(&engine.index)
            .unwrap();

        if let QueryResult::DataPoints(points) = result {
            // Should get points at indices 1, 4, 7 (sensor1)
            assert_eq!(points.len(), 3);
            assert!(points.iter().all(|p| {
                p.tags.as_ref().unwrap().get("device").unwrap() == "sensor1"
            }));
        } else {
            panic!("Expected DataPoints result");
        }
    }

    #[test]
    fn test_aggregation_queries() {
        let mut engine = QueryEngine::new();
        let test_data = create_test_data();
        engine.add_data_points(test_data);

        // Test count aggregation
        let result = engine
            .query()
            .time_range(0, 5000)
            .aggregate(AggregationType::Count)
            .execute(&engine.index)
            .unwrap();

        if let QueryResult::Aggregations(aggs) = result {
            assert_eq!(aggs.len(), 1);
            assert_eq!(aggs[0].count, 6);
            if let Some(Value::Integer(count)) = &aggs[0].value {
                assert_eq!(*count, 6);
            } else {
                panic!("Expected integer count value");
            }
        } else {
            panic!("Expected Aggregations result");
        }

        // Test average aggregation
        let result = engine
            .query()
            .time_range(0, 4000)
            .aggregate(AggregationType::Average)
            .execute(&engine.index)
            .unwrap();

        if let QueryResult::Aggregations(aggs) = result {
            assert_eq!(aggs.len(), 1);
            if let Some(Value::Float(avg)) = &aggs[0].value {
                assert_eq!(*avg, 20.0); // (0 + 10 + 20 + 30 + 40) / 5
            } else {
                panic!("Expected float average value");
            }
        } else {
            panic!("Expected Aggregations result");
        }
    }

    #[test]
    fn test_grouped_aggregation() {
        let mut engine = QueryEngine::new();
        let test_data = create_test_data();
        engine.add_data_points(test_data);

        // Test grouped aggregation with 2-second intervals
        let result = engine
            .query()
            .time_range(0, 8000)
            .aggregate(AggregationType::Count)
            .group_by_interval(2000) // 2 seconds in nanoseconds would be 2_000_000_000, but we're using simplified timestamps
            .execute(&engine.index)
            .unwrap();

        if let QueryResult::Aggregations(aggs) = result {
            // Should have multiple groups
            assert!(aggs.len() > 1);
            // Each group should have a count
            assert!(aggs.iter().all(|agg| agg.count > 0));
        } else {
            panic!("Expected Aggregations result");
        }
    }

    #[test]
    fn test_combined_query() {
        let mut engine = QueryEngine::new();
        let test_data = create_test_data();
        engine.add_data_points(test_data);

        // Test combined time and tag query
        let mut tags = HashMap::new();
        tags.insert("location".to_string(), "room1".to_string());

        let result = engine
            .query()
            .time_range(0, 4000)
            .tags(tags)
            .aggregate(AggregationType::Max)
            .execute(&engine.index)
            .unwrap();

        if let QueryResult::Aggregations(aggs) = result {
            assert_eq!(aggs.len(), 1);
            if let Some(Value::Float(max_val)) = &aggs[0].value {
                assert_eq!(*max_val, 40.0); // Max value in room1 within time range
            } else {
                panic!("Expected float max value");
            }
        } else {
            panic!("Expected Aggregations result");
        }
    }
}