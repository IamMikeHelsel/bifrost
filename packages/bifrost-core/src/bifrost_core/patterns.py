"""Device pattern models for Bifrost.

This module defines the core data models for device patterns, which enable
fast device recognition and optimal configuration based on historical data
and community knowledge.
"""

from enum import Enum
from typing import Any

from pydantic import BaseModel, Field, ConfigDict

from .typing import JsonDict, Timestamp


class VersionRange(BaseModel):
    """Represents a firmware version range."""

    min_version: str | None = Field(None, description="Minimum version (inclusive)")
    max_version: str | None = Field(None, description="Maximum version (inclusive)")
    exact_version: str | None = Field(None, description="Exact version match")

    def matches(self, version: str) -> bool:
        """Check if a version matches this range."""
        if self.exact_version:
            return version == self.exact_version
        
        # Simple string comparison for now
        # TODO: Implement proper semantic version comparison
        if self.min_version and version < self.min_version:
            return False
        if self.max_version and version > self.max_version:
            return False
        return True


class ProtocolSpec(BaseModel):
    """Protocol specification for device patterns."""

    protocol: str = Field(..., description="Protocol name (e.g., 'modbus.tcp')")
    variant: str | None = Field(None, description="Protocol variant or implementation")
    version: str | None = Field(None, description="Protocol version")
    port: int | None = Field(None, description="Default port for this protocol")


class ResponsePattern(BaseModel):
    """Network response pattern for device fingerprinting."""

    request_data: bytes = Field(..., description="Request data sent to device")
    expected_response_pattern: str = Field(
        ..., description="Regex pattern for expected response"
    )
    response_length_min: int = Field(0, description="Minimum response length")
    response_length_max: int | None = Field(None, description="Maximum response length")
    confidence_weight: float = Field(1.0, description="Weight for confidence calculation")


class TimingProfile(BaseModel):
    """Timing characteristics for device communication."""

    typical_response_time_ms: float = Field(
        ..., description="Typical response time in milliseconds"
    )
    max_response_time_ms: float = Field(
        ..., description="Maximum acceptable response time"
    )
    inter_request_delay_ms: float = Field(
        0, description="Required delay between requests"
    )
    connection_timeout_ms: float = Field(
        5000, description="Connection timeout in milliseconds"
    )


class ServiceInfo(BaseModel):
    """Service discovery information."""

    service_type: str = Field(..., description="Type of service")
    service_data: JsonDict = Field(
        default_factory=dict, description="Service-specific data"
    )


class RequestTemplate(BaseModel):
    """Template for optimized device requests."""

    template_id: str = Field(..., description="Unique identifier for template")
    request_pattern: str = Field(..., description="Request pattern with placeholders")
    expected_response_pattern: str = Field(..., description="Expected response pattern")
    batch_compatible: bool = Field(False, description="Can be batched with other requests")
    priority: int = Field(1, description="Request priority (1=high, 10=low)")


class DataPointMap(BaseModel):
    """Mapping for device data points."""

    tag_name: str = Field(..., description="Tag name")
    address: str = Field(..., description="Device address")
    data_type: str = Field(..., description="Data type")
    scale_factor: float = Field(1.0, description="Scaling factor")
    unit: str | None = Field(None, description="Engineering unit")
    description: str | None = Field(None, description="Description")


class ErrorStrategy(BaseModel):
    """Error handling strategy for device communication."""

    retry_count: int = Field(3, description="Number of retries")
    retry_delay_ms: float = Field(1000, description="Delay between retries")
    fallback_strategy: str = Field("skip", description="Fallback strategy")
    error_codes_to_ignore: list[str] = Field(
        default_factory=list, description="Error codes to ignore"
    )


class BandwidthProfile(BaseModel):
    """Bandwidth requirements for device communication."""

    bytes_per_second: float = Field(..., description="Bytes per second")
    burst_capacity: float = Field(..., description="Burst capacity in bytes")
    concurrent_connections: int = Field(1, description="Concurrent connections")


class DiscoverySignature(BaseModel):
    """Device discovery signature for pattern matching."""

    network_responses: list[ResponsePattern] = Field(
        default_factory=list, description="Network response patterns"
    )
    timing_characteristics: TimingProfile | None = Field(
        None, description="Timing profile"
    )
    service_discovery_data: list[ServiceInfo] = Field(
        default_factory=list, description="Service discovery information"
    )


class CommunicationProfile(BaseModel):
    """Communication profile for optimal device interaction."""

    optimal_polling_rate: float = Field(
        1.0, description="Optimal polling rate in Hz"
    )
    request_templates: list[RequestTemplate] = Field(
        default_factory=list, description="Request templates"
    )
    data_point_mappings: list[DataPointMap] = Field(
        default_factory=list, description="Data point mappings"
    )
    error_handling_strategy: ErrorStrategy = Field(
        default_factory=ErrorStrategy, description="Error handling strategy"
    )


class HistoricalPerformance(BaseModel):
    """Historical performance metrics for device pattern."""

    avg_response_time: float = Field(..., description="Average response time in ms")
    reliability_score: float = Field(
        ..., description="Reliability score (0.0-1.0)", ge=0.0, le=1.0
    )
    bandwidth_requirements: BandwidthProfile = Field(
        ..., description="Bandwidth requirements"
    )
    last_updated: Timestamp = Field(..., description="Last update timestamp")


class PatternStatus(str, Enum):
    """Status of a device pattern."""

    ACTIVE = "active"
    DEPRECATED = "deprecated"
    EXPERIMENTAL = "experimental"
    ARCHIVED = "archived"


class DevicePattern(BaseModel):
    """Complete device pattern for recognition and optimization.
    
    This represents the core pattern data structure that enables fast device
    recognition and optimal configuration based on historical data.
    """
    
    model_config = ConfigDict(use_enum_values=True)

    # Identity
    pattern_id: str = Field(..., description="Unique pattern identifier")
    manufacturer_id: str = Field(..., description="Manufacturer identifier")
    product_family: str = Field(..., description="Product family")
    model_number: str = Field(..., description="Model number")
    firmware_version_range: VersionRange = Field(
        default_factory=VersionRange, description="Supported firmware versions"
    )
    protocol_variant: ProtocolSpec = Field(..., description="Protocol specification")

    # Discovery Patterns
    discovery_signature: DiscoverySignature = Field(
        default_factory=DiscoverySignature, description="Discovery signature"
    )

    # Communication Templates
    communication_profile: CommunicationProfile = Field(
        default_factory=CommunicationProfile, description="Communication profile"
    )

    # Performance Metrics
    historical_performance: HistoricalPerformance | None = Field(
        None, description="Historical performance data"
    )

    # Confidence Scoring
    pattern_confidence: float = Field(
        0.5, description="Pattern confidence score", ge=0.0, le=1.0
    )
    usage_count: int = Field(0, description="Number of times pattern was used")
    last_verified: Timestamp | None = Field(
        None, description="Last verification timestamp"
    )
    contributor_reputation: float = Field(
        0.5, description="Contributor reputation score", ge=0.0, le=1.0
    )

    # Metadata
    status: PatternStatus = Field(PatternStatus.ACTIVE, description="Pattern status")
    tags: list[str] = Field(default_factory=list, description="Pattern tags")
    metadata: JsonDict = Field(
        default_factory=dict, description="Additional metadata"
    )

    def calculate_match_confidence(self, device_data: JsonDict) -> float:
        """Calculate confidence score for pattern match against device data.
        
        Args:
            device_data: Device data to match against
            
        Returns:
            Confidence score between 0.0 and 1.0
        """
        confidence = 0.0
        total_weight = 0.0

        # Base confidence from pattern itself
        confidence += self.pattern_confidence * 0.3
        total_weight += 0.3

        # Manufacturer match
        if device_data.get("manufacturer") == self.manufacturer_id:
            confidence += 1.0 * 0.3
            total_weight += 0.3

        # Model match
        if device_data.get("model") == self.model_number:
            confidence += 1.0 * 0.2
            total_weight += 0.2

        # Protocol match
        if device_data.get("protocol") == self.protocol_variant.protocol:
            confidence += 1.0 * 0.2
            total_weight += 0.2

        # Normalize by total weight
        if total_weight > 0:
            confidence = confidence / total_weight

        return min(max(confidence, 0.0), 1.0)

    def is_compatible(self, device_data: JsonDict) -> bool:
        """Check if this pattern is compatible with the given device data.
        
        Args:
            device_data: Device data to check compatibility
            
        Returns:
            True if pattern is compatible with device
        """
        # Check protocol compatibility
        if device_data.get("protocol") != self.protocol_variant.protocol:
            return False

        # Check firmware version if available
        firmware_version = device_data.get("firmware_version")
        if firmware_version and not self.firmware_version_range.matches(firmware_version):
            return False

        return True


class PatternMatchResult(BaseModel):
    """Result of pattern matching operation."""

    pattern: DevicePattern = Field(..., description="Matched pattern")
    confidence: float = Field(
        ..., description="Match confidence score", ge=0.0, le=1.0
    )
    match_type: str = Field(
        ..., description="Type of match (exact, fuzzy, composite)"
    )
    match_details: JsonDict = Field(
        default_factory=dict, description="Detailed match information"
    )


class PatternDatabase(BaseModel):
    """Database of device patterns."""
    
    model_config = ConfigDict(use_enum_values=True)

    patterns: dict[str, DevicePattern] = Field(
        default_factory=dict, description="Patterns indexed by pattern_id"
    )
    version: str = Field("1.0", description="Database version")
    last_updated: Timestamp | None = Field(
        None, description="Last update timestamp"
    )
    metadata: JsonDict = Field(
        default_factory=dict, description="Database metadata"
    )

    def add_pattern(self, pattern: DevicePattern) -> None:
        """Add a pattern to the database."""
        self.patterns[pattern.pattern_id] = pattern

    def remove_pattern(self, pattern_id: str) -> bool:
        """Remove a pattern from the database."""
        if pattern_id in self.patterns:
            del self.patterns[pattern_id]
            return True
        return False

    def find_patterns(
        self,
        device_data: JsonDict,
        min_confidence: float = 0.5
    ) -> list[PatternMatchResult]:
        """Find matching patterns for device data.
        
        Args:
            device_data: Device data to match against
            min_confidence: Minimum confidence threshold
            
        Returns:
            List of pattern matches sorted by confidence (highest first)
        """
        matches = []

        for pattern in self.patterns.values():
            if not pattern.is_compatible(device_data):
                continue

            confidence = pattern.calculate_match_confidence(device_data)
            if confidence >= min_confidence:
                match_result = PatternMatchResult(
                    pattern=pattern,
                    confidence=confidence,
                    match_type="fuzzy",  # Default to fuzzy for now
                    match_details={"device_data": device_data}
                )
                matches.append(match_result)

        # Sort by confidence (highest first)
        matches.sort(key=lambda x: x.confidence, reverse=True)
        return matches

    def get_pattern(self, pattern_id: str) -> DevicePattern | None:
        """Get a pattern by ID."""
        return self.patterns.get(pattern_id)

    def count_patterns(self) -> int:
        """Get the number of patterns in the database."""
        return len(self.patterns)