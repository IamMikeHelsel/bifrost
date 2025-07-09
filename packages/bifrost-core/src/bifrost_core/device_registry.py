"""Device Registry System for tracking tested devices and their configurations."""

import json
import yaml
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional, Union

try:
    from pydantic import BaseModel, Field, field_validator, ConfigDict
except ImportError:
    # Fallback for environments without pydantic
    BaseModel = object
    Field = lambda **kwargs: None
    ConfigDict = dict
    def field_validator(*args, **kwargs):
        def decorator(func):
            return func
        return decorator

__all__ = [
    "PerformanceMetrics",
    "VirtualDeviceConfiguration", 
    "VirtualDevice",
    "ProtocolSupport",
    "RealDevice", 
    "DeviceRegistry"
]


class PerformanceMetrics(BaseModel):
    """Performance metrics for devices."""
    
    model_config = ConfigDict(extra="allow")
    
    throughput: Optional[str] = Field(None, description="Throughput measurement (e.g., '1500 regs/sec')")
    max_throughput: Optional[str] = Field(None, description="Maximum throughput")
    latency: Optional[str] = Field(None, description="Latency measurement (e.g., '0.5ms')")


class VirtualDeviceConfiguration(BaseModel):
    """Configuration for virtual device simulators."""
    
    model_config = ConfigDict(extra="allow")
    
    registers: Optional[int] = Field(None, description="Number of registers available")
    functions: Optional[List[int]] = Field(None, description="Supported function codes")
    performance: Optional[PerformanceMetrics] = Field(None, description="Performance characteristics")


class VirtualDevice(BaseModel):
    """Virtual device/simulator definition."""
    
    id: str = Field(..., description="Unique identifier for the virtual device")
    type: str = Field(..., description="Type of device (e.g., 'simulator')")
    protocol: str = Field(..., description="Protocol used (e.g., 'modbus_tcp')")
    configuration: Optional[VirtualDeviceConfiguration] = Field(None, description="Device configuration")
    test_scenarios: Optional[List[str]] = Field(None, description="Supported test scenarios")
    version: Optional[str] = Field(None, description="Version of the simulator")
    created_date: Optional[datetime] = Field(None, description="Creation date")
    
    @field_validator('protocol')
    @classmethod
    def validate_protocol(cls, v):
        """Validate protocol name."""
        allowed_protocols = {
            'modbus_tcp', 'modbus_rtu', 'opcua', 'ethernet_ip', 
            's7', 'bacnet', 'dnp3', 'iec61850'
        }
        if v.lower() not in allowed_protocols:
            # Allow custom protocols but warn
            pass
        return v.lower()


class ProtocolSupport(BaseModel):
    """Protocol support information for a device."""
    
    status: str = Field(..., description="Validation status (e.g., 'validated', 'testing', 'failed')")
    performance: Optional[PerformanceMetrics] = Field(None, description="Performance metrics")
    limitations: Optional[List[str]] = Field(None, description="Known limitations")
    configuration: Optional[Dict[str, Any]] = Field(None, description="Protocol-specific configuration")
    test_date: Optional[datetime] = Field(None, description="Last test date")
    
    @field_validator('status')
    @classmethod
    def validate_status(cls, v):
        """Validate status values."""
        allowed_statuses = {'validated', 'testing', 'failed', 'deprecated', 'planned'}
        if v.lower() not in allowed_statuses:
            raise ValueError(f"Status must be one of: {allowed_statuses}")
        return v.lower()


class RealDevice(BaseModel):
    """Real hardware device definition."""
    
    id: str = Field(..., description="Unique identifier for the real device")
    manufacturer: str = Field(..., description="Device manufacturer")
    model: str = Field(..., description="Device model")
    firmware: Optional[str] = Field(None, description="Firmware version")
    protocols: Dict[str, ProtocolSupport] = Field(default_factory=dict, description="Supported protocols")
    test_date: Optional[datetime] = Field(None, description="Last test date")
    test_notes: Optional[str] = Field(None, description="Test notes and observations")
    serial_number: Optional[str] = Field(None, description="Device serial number")
    location: Optional[str] = Field(None, description="Physical location or test environment")
    
    @field_validator('manufacturer')
    @classmethod
    def validate_manufacturer(cls, v):
        """Validate manufacturer name."""
        if not v or len(v.strip()) == 0:
            raise ValueError("Manufacturer cannot be empty")
        return v.strip()


class DeviceRegistry:
    """Registry for managing virtual and real devices."""
    
    def __init__(self):
        """Initialize the device registry."""
        self.virtual_devices: Dict[str, VirtualDevice] = {}
        self.real_devices: Dict[str, RealDevice] = {}
        
    def register_virtual_device(self, device: VirtualDevice) -> None:
        """Register a virtual device."""
        self.virtual_devices[device.id] = device
        
    def register_real_device(self, device: RealDevice) -> None:
        """Register a real device."""
        self.real_devices[device.id] = device
        
    def get_virtual_device(self, device_id: str) -> Optional[VirtualDevice]:
        """Get a virtual device by ID."""
        return self.virtual_devices.get(device_id)
        
    def get_real_device(self, device_id: str) -> Optional[RealDevice]:
        """Get a real device by ID."""
        return self.real_devices.get(device_id)
        
    def list_virtual_devices(self) -> List[VirtualDevice]:
        """List all virtual devices."""
        return list(self.virtual_devices.values())
        
    def list_real_devices(self) -> List[RealDevice]:
        """List all real devices.""" 
        return list(self.real_devices.values())
        
    def find_devices_by_protocol(self, protocol: str) -> Dict[str, List[Union[VirtualDevice, RealDevice]]]:
        """Find all devices supporting a specific protocol."""
        result = {'virtual': [], 'real': []}
        
        # Search virtual devices
        for device in self.virtual_devices.values():
            if device.protocol.lower() == protocol.lower():
                result['virtual'].append(device)
                
        # Search real devices
        for device in self.real_devices.values():
            if protocol.lower() in device.protocols:
                result['real'].append(device)
                
        return result
        
    def get_compatibility_report(self, protocol: str) -> Dict[str, Any]:
        """Generate a compatibility report for a protocol."""
        devices = self.find_devices_by_protocol(protocol)
        
        report = {
            'protocol': protocol,
            'virtual_devices_count': len(devices['virtual']),
            'real_devices_count': len(devices['real']),
            'virtual_devices': [],
            'real_devices': [],
            'total_tested': len(devices['virtual']) + len(devices['real'])
        }
        
        # Add virtual device details
        for device in devices['virtual']:
            report['virtual_devices'].append({
                'id': device.id,
                'type': device.type,
                'configuration': device.configuration.model_dump() if device.configuration else None
            })
            
        # Add real device details  
        for device in devices['real']:
            protocol_info = device.protocols.get(protocol.lower(), {})
            report['real_devices'].append({
                'id': device.id,
                'manufacturer': device.manufacturer,
                'model': device.model,
                'status': protocol_info.status if hasattr(protocol_info, 'status') else 'unknown',
                'limitations': protocol_info.limitations if hasattr(protocol_info, 'limitations') else []
            })
            
        return report
        
    def export_to_dict(self) -> Dict[str, Any]:
        """Export registry to dictionary format."""
        return {
            'device_registry': {
                'virtual_devices': [device.model_dump() for device in self.virtual_devices.values()],
                'real_devices': [device.model_dump() for device in self.real_devices.values()]
            }
        }
        
    def export_to_json(self, file_path: Optional[Union[str, Path]] = None) -> str:
        """Export registry to JSON format."""
        data = self.export_to_dict()
        json_str = json.dumps(data, indent=2, default=str)
        
        if file_path:
            Path(file_path).write_text(json_str)
            
        return json_str
        
    def export_to_yaml(self, file_path: Optional[Union[str, Path]] = None) -> str:
        """Export registry to YAML format."""
        data = self.export_to_dict()
        yaml_str = yaml.dump(data, default_flow_style=False, sort_keys=False)
        
        if file_path:
            Path(file_path).write_text(yaml_str)
            
        return yaml_str
        
    def import_from_dict(self, data: Dict[str, Any]) -> None:
        """Import registry from dictionary format."""
        registry_data = data.get('device_registry', {})
        
        # Import virtual devices
        for device_data in registry_data.get('virtual_devices', []):
            device = VirtualDevice(**device_data)
            self.register_virtual_device(device)
            
        # Import real devices
        for device_data in registry_data.get('real_devices', []):
            device = RealDevice(**device_data)
            self.register_real_device(device)
            
    def import_from_json(self, file_path: Union[str, Path]) -> None:
        """Import registry from JSON file."""
        data = json.loads(Path(file_path).read_text())
        self.import_from_dict(data)
        
    def import_from_yaml(self, file_path: Union[str, Path]) -> None:
        """Import registry from YAML file."""
        data = yaml.safe_load(Path(file_path).read_text())
        self.import_from_dict(data)
        
    def clear(self) -> None:
        """Clear all registered devices."""
        self.virtual_devices.clear()
        self.real_devices.clear()
        
    def remove_virtual_device(self, device_id: str) -> bool:
        """Remove a virtual device by ID."""
        if device_id in self.virtual_devices:
            del self.virtual_devices[device_id]
            return True
        return False
        
    def remove_real_device(self, device_id: str) -> bool:
        """Remove a real device by ID."""
        if device_id in self.real_devices:
            del self.real_devices[device_id]
            return True
        return False