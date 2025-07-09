"""Tests for test integration functionality."""

from datetime import datetime, timedelta

import pytest

try:
    from bifrost_core.test_integration import (
        DeviceTestTracker,
        TestResult,
        TestSession,
        TestStatus,
    )
    from bifrost_core.device_registry import (
        DeviceRegistry,
        RealDevice,
        VirtualDevice,
        ProtocolSupport,
        PerformanceMetrics,
        VirtualDeviceConfiguration,
    )
except ImportError as e:
    pytest.skip(f"Required modules not available: {e}", allow_module_level=True)


class TestDeviceTestTracker:
    """Tests for DeviceTestTracker class."""
    
    @pytest.fixture
    def tracker(self) -> DeviceTestTracker:
        """Create a fresh test tracker."""
        return DeviceTestTracker()
    
    @pytest.fixture
    def sample_device(self) -> RealDevice:
        """Create a sample real device."""
        return RealDevice(
            id="test_plc",
            manufacturer="Test Manufacturer",
            model="Test Model",
            protocols={
                "modbus_tcp": ProtocolSupport(
                    status="testing",
                    performance=PerformanceMetrics(throughput="500 regs/sec")
                )
            }
        )
    
    def test_create_tracker(self):
        """Test creating a test tracker."""
        tracker = DeviceTestTracker()
        assert tracker is not None
        assert tracker.device_registry is not None
        assert len(tracker.test_sessions) == 0
        assert len(tracker.test_results) == 0
    
    def test_start_end_test_session(self, tracker: DeviceTestTracker):
        """Test starting and ending test sessions."""
        # Start session
        session = tracker.start_test_session("session_1", {"test_type": "integration"})
        
        assert session.session_id == "session_1"
        assert session.start_time is not None
        assert session.end_time is None
        assert session.metadata["test_type"] == "integration"
        assert "session_1" in tracker.test_sessions
        
        # End session
        ended_session = tracker.end_test_session("session_1")
        
        assert ended_session is not None
        assert ended_session.end_time is not None
        assert ended_session.session_id == "session_1"
    
    def test_record_test_result(self, tracker: DeviceTestTracker):
        """Test recording test results."""
        session = tracker.start_test_session("session_1")
        
        start_time = datetime.now()
        end_time = start_time + timedelta(seconds=5)
        
        result = tracker.record_test_result(
            session_id="session_1",
            test_name="modbus_read_test",
            device_id="test_plc",
            protocol="modbus_tcp",
            status=TestStatus.PASSED,
            start_time=start_time,
            end_time=end_time,
            performance_metrics={"throughput": 1000, "latency": 2.5}
        )
        
        assert result.test_name == "modbus_read_test"
        assert result.device_id == "test_plc"
        assert result.protocol == "modbus_tcp"
        assert result.status == TestStatus.PASSED
        assert result.duration == 5.0
        assert result.performance_metrics["throughput"] == 1000
        
        # Check it's in global results
        assert len(tracker.test_results) == 1
        assert tracker.test_results[0] == result
        
        # Check it's in session results
        assert len(tracker.test_sessions["session_1"].test_results) == 1
        assert tracker.test_sessions["session_1"].test_results[0] == result
    
    def test_get_device_test_history(self, tracker: DeviceTestTracker):
        """Test getting device test history."""
        session = tracker.start_test_session("session_1")
        
        # Record tests for multiple devices
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_1",
            device_id="device_1",
            protocol="modbus_tcp",
            status=TestStatus.PASSED,
            start_time=datetime.now()
        )
        
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_2",
            device_id="device_2",
            protocol="modbus_tcp",
            status=TestStatus.FAILED,
            start_time=datetime.now()
        )
        
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_3",
            device_id="device_1",
            protocol="opcua",
            status=TestStatus.PASSED,
            start_time=datetime.now()
        )
        
        # Get history for device_1
        device_1_history = tracker.get_device_test_history("device_1")
        assert len(device_1_history) == 2
        assert all(r.device_id == "device_1" for r in device_1_history)
        
        # Get history for device_2
        device_2_history = tracker.get_device_test_history("device_2")
        assert len(device_2_history) == 1
        assert device_2_history[0].device_id == "device_2"
    
    def test_get_protocol_test_history(self, tracker: DeviceTestTracker):
        """Test getting protocol test history."""
        session = tracker.start_test_session("session_1")
        
        # Record tests for multiple protocols
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_1",
            device_id="device_1",
            protocol="modbus_tcp",
            status=TestStatus.PASSED,
            start_time=datetime.now()
        )
        
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_2",
            device_id="device_2",
            protocol="opcua",
            status=TestStatus.FAILED,
            start_time=datetime.now()
        )
        
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_3",
            device_id="device_3",
            protocol="modbus_tcp",
            status=TestStatus.PASSED,
            start_time=datetime.now()
        )
        
        # Get history for modbus_tcp
        modbus_history = tracker.get_protocol_test_history("modbus_tcp")
        assert len(modbus_history) == 2
        assert all(r.protocol == "modbus_tcp" for r in modbus_history)
        
        # Get history for opcua
        opcua_history = tracker.get_protocol_test_history("opcua")
        assert len(opcua_history) == 1
        assert opcua_history[0].protocol == "opcua"
    
    def test_update_real_device_from_tests(self, tracker: DeviceTestTracker, sample_device: RealDevice):
        """Test updating real device based on test results."""
        # Register the device
        tracker.device_registry.register_real_device(sample_device)
        
        # Record successful tests
        session = tracker.start_test_session("session_1")
        base_time = datetime.now()
        
        for i in range(4):
            tracker.record_test_result(
                session_id="session_1",
                test_name=f"test_{i}",
                device_id="test_plc",
                protocol="modbus_tcp",
                status=TestStatus.PASSED,
                start_time=base_time + timedelta(minutes=i),
                end_time=base_time + timedelta(minutes=i, seconds=30)
            )
        
        # Record one failed test
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_failed",
            device_id="test_plc",
            protocol="modbus_tcp",
            status=TestStatus.FAILED,
            start_time=base_time + timedelta(minutes=5),
            end_time=base_time + timedelta(minutes=5, seconds=30),
            error_message="Connection timeout"
        )
        
        # Update device from test results
        updated = tracker.update_device_from_test_results("test_plc")
        assert updated is True
        
        # Check device was updated
        updated_device = tracker.device_registry.get_real_device("test_plc")
        assert updated_device is not None
        assert updated_device.protocols["modbus_tcp"].status == "validated"  # 80% success rate
        assert updated_device.test_date is not None
        assert "Connection timeout" in updated_device.test_notes
    
    def test_update_virtual_device_from_tests(self, tracker: DeviceTestTracker):
        """Test updating virtual device based on test results."""
        # Create and register virtual device
        virtual_device = VirtualDevice(
            id="sim_device",
            type="simulator",
            protocol="modbus_tcp",
            configuration=VirtualDeviceConfiguration(
                registers=1000,
                performance=PerformanceMetrics()
            )
        )
        tracker.device_registry.register_virtual_device(virtual_device)
        
        # Record tests with performance data
        session = tracker.start_test_session("session_1")
        base_time = datetime.now()
        
        for i in range(3):
            tracker.record_test_result(
                session_id="session_1",
                test_name=f"perf_test_{i}",
                device_id="sim_device",
                protocol="modbus_tcp",
                status=TestStatus.PASSED,
                start_time=base_time + timedelta(minutes=i),
                end_time=base_time + timedelta(minutes=i, seconds=30),
                performance_metrics={
                    "throughput": 1000 + i * 100,  # 1000, 1100, 1200
                    "latency": 1.0 + i * 0.5       # 1.0, 1.5, 2.0
                }
            )
        
        # Update device from test results
        updated = tracker.update_device_from_test_results("sim_device")
        assert updated is True
        
        # Check device was updated
        updated_device = tracker.device_registry.get_virtual_device("sim_device")
        assert updated_device is not None
        assert updated_device.configuration.performance.throughput == "1100 ops/sec"  # Average
        assert updated_device.configuration.performance.latency == "1.5ms"  # Average
    
    def test_generate_test_report(self, tracker: DeviceTestTracker):
        """Test generating test reports."""
        session = tracker.start_test_session("session_1")
        base_time = datetime.now()
        
        # Record mixed test results
        test_cases = [
            ("device_1", "modbus_tcp", TestStatus.PASSED),
            ("device_1", "modbus_tcp", TestStatus.PASSED),
            ("device_1", "modbus_tcp", TestStatus.FAILED),
            ("device_2", "opcua", TestStatus.PASSED),
            ("device_2", "opcua", TestStatus.ERROR),
            ("device_3", "ethernet_ip", TestStatus.SKIPPED),
        ]
        
        for i, (device_id, protocol, status) in enumerate(test_cases):
            tracker.record_test_result(
                session_id="session_1",
                test_name=f"test_{i}",
                device_id=device_id,
                protocol=protocol,
                status=status,
                start_time=base_time + timedelta(minutes=i)
            )
        
        # Generate overall report
        report = tracker.generate_test_report()
        
        assert report["total_tests"] == 6
        assert report["passed_tests"] == 3
        assert report["failed_tests"] == 1
        assert report["error_tests"] == 1
        assert report["skipped_tests"] == 1
        assert report["success_rate"] == 50.0  # 3/6 * 100
        assert report["tested_devices"] == 3
        assert report["tested_protocols"] == 3
        
        # Generate protocol-specific report
        modbus_report = tracker.generate_test_report("modbus_tcp")
        assert modbus_report["total_tests"] == 3
        assert modbus_report["passed_tests"] == 2
        assert modbus_report["success_rate"] == pytest.approx(66.67, rel=1e-2)
    
    def test_get_device_test_status(self, tracker: DeviceTestTracker):
        """Test getting device test status."""
        session = tracker.start_test_session("session_1")
        base_time = datetime.now()
        
        # Record tests for device
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_1",
            device_id="device_1",
            protocol="modbus_tcp",
            status=TestStatus.PASSED,
            start_time=base_time
        )
        
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_2",
            device_id="device_1",
            protocol="modbus_tcp",
            status=TestStatus.PASSED,
            start_time=base_time + timedelta(minutes=1)
        )
        
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_3",
            device_id="device_1",
            protocol="opcua",
            status=TestStatus.FAILED,
            start_time=base_time + timedelta(minutes=2)
        )
        
        # Get device status
        status = tracker.get_device_test_status("device_1")
        
        assert status["device_id"] == "device_1"
        assert status["status"] == TestStatus.FAILED  # Latest test
        assert status["total_tests"] == 3
        assert status["passed_tests"] == 2
        assert status["success_rate"] == pytest.approx(66.67, rel=1e-2)
        
        # Check protocol status
        assert "modbus_tcp" in status["protocol_status"]
        assert "opcua" in status["protocol_status"]
        assert status["protocol_status"]["modbus_tcp"]["success_rate"] == 100.0
        assert status["protocol_status"]["opcua"]["success_rate"] == 0.0
    
    def test_export_test_results(self, tracker: DeviceTestTracker):
        """Test exporting test results."""
        session = tracker.start_test_session("session_1")
        
        tracker.record_test_result(
            session_id="session_1",
            test_name="test_1",
            device_id="device_1",
            protocol="modbus_tcp",
            status=TestStatus.PASSED,
            start_time=datetime.now()
        )
        
        # Export as JSON
        json_export = tracker.export_test_results("json")
        assert json_export is not None
        assert "test_sessions" in json_export
        assert "test_results" in json_export
        assert "export_timestamp" in json_export
        
        # Export as YAML
        yaml_export = tracker.export_test_results("yaml")
        assert yaml_export is not None
        assert "test_sessions:" in yaml_export
        assert "test_results:" in yaml_export