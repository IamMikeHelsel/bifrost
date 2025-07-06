"""Base stream processing pipeline for edge analytics."""

from abc import ABC, abstractmethod
from typing import Any, AsyncIterator, Callable, List, Optional, TypeVar
import asyncio
from collections import deque
from datetime import datetime, timedelta
from .data import DataPoint


T = TypeVar('T')


class Pipeline(ABC):
    """Base class for stream processing pipelines."""
    
    def __init__(self, buffer_size: int = 1000) -> None:
        self.buffer_size = buffer_size
        self._buffer: deque = deque(maxlen=buffer_size)
        self._processors: List[Callable] = []
        self._running = False
        self._lock = asyncio.Lock()
    
    @abstractmethod
    async def process(self, data: DataPoint) -> Optional[DataPoint]:
        """Process a single data point through the pipeline."""
        pass
    
    async def add_processor(self, processor: Callable[[DataPoint], DataPoint]) -> None:
        """Add a processing function to the pipeline."""
        async with self._lock:
            self._processors.append(processor)
    
    async def push(self, data: DataPoint) -> None:
        """Push data into the pipeline."""
        async with self._lock:
            self._buffer.append(data)
    
    async def pull(self) -> Optional[DataPoint]:
        """Pull processed data from the pipeline."""
        async with self._lock:
            if self._buffer:
                return self._buffer.popleft()
            return None
    
    async def stream(self) -> AsyncIterator[DataPoint]:
        """Stream data through the pipeline."""
        self._running = True
        while self._running:
            data = await self.pull()
            if data:
                processed = await self.process(data)
                if processed:
                    yield processed
            else:
                await asyncio.sleep(0.01)  # Small delay to prevent busy-waiting
    
    async def stop(self) -> None:
        """Stop the pipeline stream."""
        self._running = False


class WindowedPipeline(Pipeline):
    """Pipeline with time-windowed processing capabilities."""
    
    def __init__(self, window_size: timedelta, buffer_size: int = 1000) -> None:
        super().__init__(buffer_size)
        self.window_size = window_size
        self._window: List[DataPoint] = []
        self._window_start: Optional[datetime] = None
    
    async def process(self, data: DataPoint) -> Optional[DataPoint]:
        """Process data with windowing."""
        # Initialize window start time
        if self._window_start is None:
            self._window_start = data.timestamp
        
        # Check if we need to close the window
        if data.timestamp - self._window_start >= self.window_size:
            # Process window
            result = await self._process_window(self._window)
            # Reset window
            self._window = [data]
            self._window_start = data.timestamp
            return result
        else:
            # Add to current window
            self._window.append(data)
            return None
    
    @abstractmethod
    async def _process_window(self, window: List[DataPoint]) -> Optional[DataPoint]:
        """Process a complete window of data."""
        pass


class FilterPipeline(Pipeline):
    """Pipeline with filtering capabilities."""
    
    def __init__(self, filter_func: Callable[[DataPoint], bool], buffer_size: int = 1000) -> None:
        super().__init__(buffer_size)
        self.filter_func = filter_func
    
    async def process(self, data: DataPoint) -> Optional[DataPoint]:
        """Process data through filter."""
        if self.filter_func(data):
            # Apply all processors
            result = data
            for processor in self._processors:
                result = processor(result)
            return result
        return None


class AggregationPipeline(WindowedPipeline):
    """Pipeline for aggregating data over time windows."""
    
    def __init__(self, window_size: timedelta, aggregation_type: str = "mean", 
                 buffer_size: int = 1000) -> None:
        super().__init__(window_size, buffer_size)
        self.aggregation_type = aggregation_type
    
    async def _process_window(self, window: List[DataPoint]) -> Optional[DataPoint]:
        """Aggregate window data."""
        if not window:
            return None
        
        # Get numeric values
        values = [dp.value for dp in window if isinstance(dp.value, (int, float))]
        if not values:
            return None
        
        # Calculate aggregation
        if self.aggregation_type == "mean":
            result = sum(values) / len(values)
        elif self.aggregation_type == "sum":
            result = sum(values)
        elif self.aggregation_type == "min":
            result = min(values)
        elif self.aggregation_type == "max":
            result = max(values)
        elif self.aggregation_type == "count":
            result = len(values)
        else:
            raise ValueError(f"Unknown aggregation type: {self.aggregation_type}")
        
        # Create aggregated data point
        first_dp = window[0]
        return DataPoint(
            name=f"{first_dp.name}_{self.aggregation_type}",
            address=first_dp.address,
            value=result,
            timestamp=datetime.utcnow(),
            data_type=first_dp.data_type,
            quality=first_dp.quality,
            protocol=first_dp.protocol,
            source_device=first_dp.source_device,
            unit=first_dp.unit,
            description=f"{self.aggregation_type} of {len(window)} values",
            metadata={
                "aggregation": self.aggregation_type,
                "window_size": str(self.window_size),
                "sample_count": len(window)
            }
        )