"""Integration between Device Registry and Testing Framework."""

import json
from datetime import datetime
from enum import Enum
from typing import Any, Dict, List, Optional, Union

try:
    from pydantic import BaseModel, Field
except ImportError:
    BaseModel = object
    Field = lambda **kwargs: None

from bifrost_core.device_registry import DeviceRegistry, RealDevice, VirtualDevice

__all__ = [
    "TestResult",
    "TestStatus", 
    "TestSession",
    "DeviceTestTracker"
]


class TestStatus(str, Enum):
    """Status of a test execution."""
    PASSED = "passed"
    FAILED = "failed"
    SKIPPED = "skipped"
    ERROR = "error"
    RUNNING = "running"


class TestResult(BaseModel):
    """Result of a single test execution."""
    
    test_name: str = Field(..., description="Name of the test")
    device_id: str = Field(..., description="ID of device being tested")
    protocol: str = Field(..., description="Protocol being tested")
    status: TestStatus = Field(..., description="Test execution status")
    start_time: datetime = Field(..., description="Test start time")
    end_time: Optional[datetime] = Field(None, description="Test end time")
    duration: Optional[float] = Field(None, description="Test duration in seconds")
    error_message: Optional[str] = Field(None, description="Error message if failed")
    performance_metrics: Optional[Dict[str, Any]] = Field(None, description="Performance data collected")
    test_data: Optional[Dict[str, Any]] = Field(None, description="Additional test data")


class TestSession(BaseModel):
    """A testing session containing multiple test results."""
    
    session_id: str = Field(..., description="Unique session identifier")
    start_time: datetime = Field(..., description="Session start time")
    end_time: Optional[datetime] = Field(None, description="Session end time")
    test_results: List[TestResult] = Field(default_factory=list, description="Test results in this session")
    metadata: Optional[Dict[str, Any]] = Field(None, description="Session metadata")


class DeviceTestTracker:
    """Tracks test results and integrates with device registry."""
    
    def __init__(self, device_registry: Optional[DeviceRegistry] = None):
        """Initialize the test tracker."""
        self.device_registry = device_registry or DeviceRegistry()
        self.test_sessions: Dict[str, TestSession] = {}
        self.test_results: List[TestResult] = []
    
    def start_test_session(self, session_id: str, metadata: Optional[Dict[str, Any]] = None) -> TestSession:
        """Start a new test session."""
        session = TestSession(
            session_id=session_id,
            start_time=datetime.now(),
            metadata=metadata or {}
        )
        self.test_sessions[session_id] = session
        return session
    
    def end_test_session(self, session_id: str) -> Optional[TestSession]:
        """End a test session."""
        if session_id in self.test_sessions:
            session = self.test_sessions[session_id]
            session.end_time = datetime.now()
            return session
        return None
    
    def record_test_result(self, 
                          session_id: str,
                          test_name: str,
                          device_id: str,
                          protocol: str,
                          status: TestStatus,
                          start_time: datetime,
                          end_time: Optional[datetime] = None,
                          error_message: Optional[str] = None,
                          performance_metrics: Optional[Dict[str, Any]] = None,
                          test_data: Optional[Dict[str, Any]] = None) -> TestResult:
        """Record a test result."""
        
        duration = None
        if end_time and start_time:
            duration = (end_time - start_time).total_seconds()
        
        result = TestResult(
            test_name=test_name,
            device_id=device_id,
            protocol=protocol,
            status=status,
            start_time=start_time,
            end_time=end_time,
            duration=duration,
            error_message=error_message,
            performance_metrics=performance_metrics,
            test_data=test_data
        )
        
        # Add to global results
        self.test_results.append(result)
        
        # Add to session if it exists
        if session_id in self.test_sessions:
            self.test_sessions[session_id].test_results.append(result)
        
        return result
    
    def get_device_test_history(self, device_id: str) -> List[TestResult]:
        """Get test history for a specific device."""
        return [result for result in self.test_results if result.device_id == device_id]
    
    def get_protocol_test_history(self, protocol: str) -> List[TestResult]:
        """Get test history for a specific protocol."""
        return [result for result in self.test_results if result.protocol == protocol]
    
    def update_device_from_test_results(self, device_id: str) -> bool:
        """Update device registry based on test results."""
        device_results = self.get_device_test_history(device_id)
        if not device_results:
            return False
        
        # Check if it's a real or virtual device
        real_device = self.device_registry.get_real_device(device_id)
        virtual_device = self.device_registry.get_virtual_device(device_id)
        
        if real_device:
            return self._update_real_device_from_tests(real_device, device_results)
        elif virtual_device:
            return self._update_virtual_device_from_tests(virtual_device, device_results)
        
        return False
    
    def _update_real_device_from_tests(self, device: RealDevice, results: List[TestResult]) -> bool:
        """Update real device based on test results."""
        updated = False
        
        # Group results by protocol
        protocol_results = {}
        for result in results:
            if result.protocol not in protocol_results:
                protocol_results[result.protocol] = []
            protocol_results[result.protocol].append(result)
        
        # Update protocol support based on test results
        for protocol, protocol_test_results in protocol_results.items():
            # Get latest results
            latest_results = sorted(protocol_test_results, key=lambda x: x.start_time, reverse=True)[:5]
            
            # Determine status based on recent results
            passed_count = sum(1 for r in latest_results if r.status == TestStatus.PASSED)
            total_count = len(latest_results)
            
            if total_count > 0:
                success_rate = passed_count / total_count
                
                if success_rate >= 0.8:
                    status = "validated"
                elif success_rate >= 0.5:
                    status = "testing"
                else:
                    status = "failed"
                
                # Update or create protocol support
                if protocol in device.protocols:
                    device.protocols[protocol].status = status
                    # Update test date to most recent
                    if latest_results:
                        device.protocols[protocol].test_date = latest_results[0].end_time
                    updated = True
                
                # Update device test date and notes
                if latest_results:
                    device.test_date = latest_results[0].end_time
                    
                    # Add any error messages to notes
                    errors = [r.error_message for r in latest_results if r.error_message]
                    if errors:
                        error_summary = f"Recent test errors: {'; '.join(errors[:3])}"
                        if device.test_notes:
                            device.test_notes += f"\n{error_summary}"
                        else:
                            device.test_notes = error_summary
                    updated = True
        
        if updated:
            # Re-register the updated device
            self.device_registry.register_real_device(device)
        
        return updated
    
    def _update_virtual_device_from_tests(self, device: VirtualDevice, results: List[TestResult]) -> bool:
        """Update virtual device based on test results."""
        # For virtual devices, we mainly update performance metrics
        updated = False
        
        # Get performance data from recent successful tests
        successful_results = [r for r in results if r.status == TestStatus.PASSED and r.performance_metrics]
        
        if successful_results and device.configuration:
            # Calculate average performance metrics
            throughput_values = []
            latency_values = []
            
            for result in successful_results:
                if result.performance_metrics:
                    if 'throughput' in result.performance_metrics:
                        throughput_values.append(result.performance_metrics['throughput'])
                    if 'latency' in result.performance_metrics:
                        latency_values.append(result.performance_metrics['latency'])
            
            # Update performance metrics if we have data
            if throughput_values or latency_values:
                if not device.configuration.performance:
                    from bifrost_core.device_registry import PerformanceMetrics
                    device.configuration.performance = PerformanceMetrics()
                
                if throughput_values:
                    avg_throughput = sum(throughput_values) / len(throughput_values)
                    device.configuration.performance.throughput = f"{avg_throughput:.0f} ops/sec"
                
                if latency_values:
                    avg_latency = sum(latency_values) / len(latency_values)
                    device.configuration.performance.latency = f"{avg_latency:.1f}ms"
                
                updated = True
        
        if updated:
            # Re-register the updated device
            self.device_registry.register_virtual_device(device)
        
        return updated
    
    def generate_test_report(self, protocol: Optional[str] = None) -> Dict[str, Any]:
        """Generate a comprehensive test report."""
        results = self.test_results
        if protocol:
            results = [r for r in results if r.protocol == protocol]
        
        if not results:
            return {
                "total_tests": 0,
                "protocol_filter": protocol,
                "summary": "No test results found"
            }
        
        # Calculate statistics
        total_tests = len(results)
        passed_tests = sum(1 for r in results if r.status == TestStatus.PASSED)
        failed_tests = sum(1 for r in results if r.status == TestStatus.FAILED)
        error_tests = sum(1 for r in results if r.status == TestStatus.ERROR)
        skipped_tests = sum(1 for r in results if r.status == TestStatus.SKIPPED)
        
        # Calculate success rate
        success_rate = (passed_tests / total_tests * 100) if total_tests > 0 else 0
        
        # Get device coverage
        tested_devices = set(r.device_id for r in results)
        device_coverage = {}
        for device_id in tested_devices:
            device_results = [r for r in results if r.device_id == device_id]
            device_success = sum(1 for r in device_results if r.status == TestStatus.PASSED)
            device_total = len(device_results)
            device_coverage[device_id] = {
                "total_tests": device_total,
                "passed_tests": device_success,
                "success_rate": (device_success / device_total * 100) if device_total > 0 else 0
            }
        
        # Get protocol coverage
        protocol_coverage = {}
        tested_protocols = set(r.protocol for r in results)
        for test_protocol in tested_protocols:
            protocol_results = [r for r in results if r.protocol == test_protocol]
            protocol_success = sum(1 for r in protocol_results if r.status == TestStatus.PASSED)
            protocol_total = len(protocol_results)
            protocol_coverage[test_protocol] = {
                "total_tests": protocol_total,
                "passed_tests": protocol_success,
                "success_rate": (protocol_success / protocol_total * 100) if protocol_total > 0 else 0
            }
        
        return {
            "total_tests": total_tests,
            "passed_tests": passed_tests,
            "failed_tests": failed_tests,
            "error_tests": error_tests,
            "skipped_tests": skipped_tests,
            "success_rate": success_rate,
            "protocol_filter": protocol,
            "device_coverage": device_coverage,
            "protocol_coverage": protocol_coverage,
            "tested_devices": len(tested_devices),
            "tested_protocols": len(tested_protocols)
        }
    
    def export_test_results(self, format: str = "json") -> str:
        """Export test results in specified format."""
        data = {
            "test_sessions": [session.model_dump() for session in self.test_sessions.values()],
            "test_results": [result.model_dump() for result in self.test_results],
            "export_timestamp": datetime.now().isoformat()
        }
        
        if format.lower() == "json":
            return json.dumps(data, indent=2, default=str)
        elif format.lower() == "yaml":
            import yaml
            return yaml.dump(data, default_flow_style=False, sort_keys=False)
        else:
            raise ValueError("Format must be 'json' or 'yaml'")
    
    def get_device_test_status(self, device_id: str) -> Dict[str, Any]:
        """Get current test status for a device."""
        results = self.get_device_test_history(device_id)
        
        if not results:
            return {
                "device_id": device_id,
                "status": "untested",
                "total_tests": 0
            }
        
        # Get latest result
        latest_result = max(results, key=lambda x: x.start_time)
        
        # Calculate overall statistics
        total_tests = len(results)
        passed_tests = sum(1 for r in results if r.status == TestStatus.PASSED)
        success_rate = (passed_tests / total_tests * 100) if total_tests > 0 else 0
        
        # Group by protocol
        protocol_status = {}
        for result in results:
            if result.protocol not in protocol_status:
                protocol_status[result.protocol] = []
            protocol_status[result.protocol].append(result)
        
        for protocol in protocol_status:
            protocol_results = protocol_status[protocol]
            protocol_passed = sum(1 for r in protocol_results if r.status == TestStatus.PASSED)
            protocol_total = len(protocol_results)
            protocol_status[protocol] = {
                "total_tests": protocol_total,
                "passed_tests": protocol_passed,
                "success_rate": (protocol_passed / protocol_total * 100) if protocol_total > 0 else 0,
                "latest_result": max(protocol_results, key=lambda x: x.start_time).status
            }
        
        return {
            "device_id": device_id,
            "status": latest_result.status,
            "total_tests": total_tests,
            "passed_tests": passed_tests,
            "success_rate": success_rate,
            "latest_test": latest_result.start_time.isoformat(),
            "protocol_status": protocol_status
        }