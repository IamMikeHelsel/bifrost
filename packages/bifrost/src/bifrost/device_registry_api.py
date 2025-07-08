"""Device Registry API for programmatic access."""

import json
from pathlib import Path
from typing import Any, Dict, List, Optional, Union

try:
    from fastapi import FastAPI, HTTPException
    from fastapi.responses import JSONResponse
    FASTAPI_AVAILABLE = True
except ImportError:
    FASTAPI_AVAILABLE = False

from bifrost_core.device_registry import (
    DeviceRegistry,
    RealDevice,
    VirtualDevice,
    PerformanceMetrics,
    ProtocolSupport,
    VirtualDeviceConfiguration,
)

__all__ = ["DeviceRegistryAPI", "create_device_registry_app"]


class DeviceRegistryAPI:
    """API class for Device Registry operations."""
    
    def __init__(self, registry: Optional[DeviceRegistry] = None):
        """Initialize the API with an optional registry instance."""
        self.registry = registry or DeviceRegistry()
    
    def get_all_virtual_devices(self) -> List[Dict[str, Any]]:
        """Get all virtual devices."""
        return [device.model_dump() for device in self.registry.list_virtual_devices()]
    
    def get_all_real_devices(self) -> List[Dict[str, Any]]:
        """Get all real devices."""
        return [device.model_dump() for device in self.registry.list_real_devices()]
    
    def get_virtual_device(self, device_id: str) -> Optional[Dict[str, Any]]:
        """Get a specific virtual device by ID."""
        device = self.registry.get_virtual_device(device_id)
        return device.model_dump() if device else None
    
    def get_real_device(self, device_id: str) -> Optional[Dict[str, Any]]:
        """Get a specific real device by ID."""
        device = self.registry.get_real_device(device_id)
        return device.model_dump() if device else None
    
    def create_virtual_device(self, device_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new virtual device."""
        device = VirtualDevice(**device_data)
        self.registry.register_virtual_device(device)
        return device.model_dump()
    
    def create_real_device(self, device_data: Dict[str, Any]) -> Dict[str, Any]:
        """Create a new real device."""
        device = RealDevice(**device_data)
        self.registry.register_real_device(device)
        return device.model_dump()
    
    def update_virtual_device(self, device_id: str, device_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Update an existing virtual device."""
        if device_id not in self.registry.virtual_devices:
            return None
        
        device_data['id'] = device_id  # Ensure ID matches
        device = VirtualDevice(**device_data)
        self.registry.register_virtual_device(device)
        return device.model_dump()
    
    def update_real_device(self, device_id: str, device_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Update an existing real device."""
        if device_id not in self.registry.real_devices:
            return None
        
        device_data['id'] = device_id  # Ensure ID matches
        device = RealDevice(**device_data)
        self.registry.register_real_device(device)
        return device.model_dump()
    
    def delete_virtual_device(self, device_id: str) -> bool:
        """Delete a virtual device."""
        return self.registry.remove_virtual_device(device_id)
    
    def delete_real_device(self, device_id: str) -> bool:
        """Delete a real device."""
        return self.registry.remove_real_device(device_id)
    
    def find_devices_by_protocol(self, protocol: str) -> Dict[str, List[Dict[str, Any]]]:
        """Find devices supporting a specific protocol."""
        devices = self.registry.find_devices_by_protocol(protocol)
        return {
            'virtual': [device.model_dump() for device in devices['virtual']],
            'real': [device.model_dump() for device in devices['real']]
        }
    
    def get_compatibility_report(self, protocol: str) -> Dict[str, Any]:
        """Generate compatibility report for a protocol."""
        return self.registry.get_compatibility_report(protocol)
    
    def get_supported_protocols(self) -> List[str]:
        """Get list of all supported protocols."""
        protocols = set()
        
        # From virtual devices
        for device in self.registry.list_virtual_devices():
            protocols.add(device.protocol)
        
        # From real devices
        for device in self.registry.list_real_devices():
            protocols.update(device.protocols.keys())
        
        return sorted(list(protocols))
    
    def export_registry(self, format: str = 'json') -> str:
        """Export registry in specified format."""
        if format.lower() == 'json':
            return self.registry.export_to_json()
        elif format.lower() == 'yaml':
            return self.registry.export_to_yaml()
        else:
            raise ValueError("Format must be 'json' or 'yaml'")
    
    def import_registry(self, data: str, format: str = 'json') -> None:
        """Import registry from string data."""
        if format.lower() == 'json':
            data_dict = json.loads(data)
            self.registry.import_from_dict(data_dict)
        elif format.lower() == 'yaml':
            import yaml
            data_dict = yaml.safe_load(data)
            self.registry.import_from_dict(data_dict)
        else:
            raise ValueError("Format must be 'json' or 'yaml'")
    
    def get_registry_stats(self) -> Dict[str, Any]:
        """Get registry statistics."""
        virtual_devices = self.registry.list_virtual_devices()
        real_devices = self.registry.list_real_devices()
        
        # Count devices by protocol
        protocol_counts = {}
        for device in virtual_devices:
            protocol_counts[device.protocol] = protocol_counts.get(device.protocol, 0) + 1
        
        for device in real_devices:
            for protocol in device.protocols.keys():
                protocol_counts[protocol] = protocol_counts.get(protocol, 0) + 1
        
        return {
            'total_virtual_devices': len(virtual_devices),
            'total_real_devices': len(real_devices),
            'total_devices': len(virtual_devices) + len(real_devices),
            'supported_protocols': len(protocol_counts),
            'protocol_distribution': protocol_counts
        }


def create_device_registry_app(registry: Optional[DeviceRegistry] = None) -> Optional[Any]:
    """Create FastAPI application for Device Registry if FastAPI is available."""
    if not FASTAPI_AVAILABLE:
        return None
    
    app = FastAPI(
        title="Device Registry API",
        description="API for managing virtual and real industrial devices",
        version="1.0.0"
    )
    
    api = DeviceRegistryAPI(registry)
    
    @app.get("/", summary="API Information")
    async def root():
        """Get API information."""
        return {
            "name": "Device Registry API",
            "version": "1.0.0",
            "description": "API for managing virtual and real industrial devices"
        }
    
    @app.get("/stats", summary="Registry Statistics")
    async def get_stats():
        """Get registry statistics."""
        return api.get_registry_stats()
    
    @app.get("/protocols", summary="Supported Protocols")
    async def get_protocols():
        """Get list of supported protocols."""
        return {"protocols": api.get_supported_protocols()}
    
    @app.get("/virtual-devices", summary="List Virtual Devices")
    async def list_virtual_devices():
        """List all virtual devices."""
        return {"devices": api.get_all_virtual_devices()}
    
    @app.get("/real-devices", summary="List Real Devices")
    async def list_real_devices():
        """List all real devices."""
        return {"devices": api.get_all_real_devices()}
    
    @app.get("/virtual-devices/{device_id}", summary="Get Virtual Device")
    async def get_virtual_device(device_id: str):
        """Get a specific virtual device."""
        device = api.get_virtual_device(device_id)
        if device is None:
            raise HTTPException(status_code=404, detail="Virtual device not found")
        return device
    
    @app.get("/real-devices/{device_id}", summary="Get Real Device")
    async def get_real_device(device_id: str):
        """Get a specific real device."""
        device = api.get_real_device(device_id)
        if device is None:
            raise HTTPException(status_code=404, detail="Real device not found")
        return device
    
    @app.post("/virtual-devices", summary="Create Virtual Device")
    async def create_virtual_device(device_data: dict):
        """Create a new virtual device."""
        try:
            device = api.create_virtual_device(device_data)
            return device
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @app.post("/real-devices", summary="Create Real Device")
    async def create_real_device(device_data: dict):
        """Create a new real device."""
        try:
            device = api.create_real_device(device_data)
            return device
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @app.put("/virtual-devices/{device_id}", summary="Update Virtual Device")
    async def update_virtual_device(device_id: str, device_data: dict):
        """Update an existing virtual device."""
        try:
            device = api.update_virtual_device(device_id, device_data)
            if device is None:
                raise HTTPException(status_code=404, detail="Virtual device not found")
            return device
        except HTTPException:
            raise
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @app.put("/real-devices/{device_id}", summary="Update Real Device")
    async def update_real_device(device_id: str, device_data: dict):
        """Update an existing real device."""
        try:
            device = api.update_real_device(device_id, device_data)
            if device is None:
                raise HTTPException(status_code=404, detail="Real device not found")
            return device
        except HTTPException:
            raise
        except Exception as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @app.delete("/virtual-devices/{device_id}", summary="Delete Virtual Device")
    async def delete_virtual_device(device_id: str):
        """Delete a virtual device."""
        if not api.delete_virtual_device(device_id):
            raise HTTPException(status_code=404, detail="Virtual device not found")
        return {"message": "Virtual device deleted successfully"}
    
    @app.delete("/real-devices/{device_id}", summary="Delete Real Device")
    async def delete_real_device(device_id: str):
        """Delete a real device."""
        if not api.delete_real_device(device_id):
            raise HTTPException(status_code=404, detail="Real device not found")
        return {"message": "Real device deleted successfully"}
    
    @app.get("/protocols/{protocol}/devices", summary="Find Devices by Protocol")
    async def find_devices_by_protocol(protocol: str):
        """Find devices supporting a specific protocol."""
        return api.find_devices_by_protocol(protocol)
    
    @app.get("/protocols/{protocol}/compatibility", summary="Protocol Compatibility Report")
    async def get_protocol_compatibility(protocol: str):
        """Get compatibility report for a protocol."""
        return api.get_compatibility_report(protocol)
    
    @app.get("/export", summary="Export Registry")
    async def export_registry(format: str = "json"):
        """Export registry data."""
        try:
            data = api.export_registry(format)
            return JSONResponse(
                content={"data": data, "format": format},
                headers={"Content-Type": "application/json"}
            )
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))
    
    @app.post("/import", summary="Import Registry") 
    async def import_registry(import_data: dict):
        """Import registry data."""
        try:
            data = import_data.get('data')
            format = import_data.get('format', 'json')
            if not data:
                raise HTTPException(status_code=400, detail="No data provided")
            
            api.import_registry(data, format)
            return {"message": "Registry imported successfully"}
        except ValueError as e:
            raise HTTPException(status_code=400, detail=str(e))
        except Exception as e:
            raise HTTPException(status_code=500, detail=str(e))
    
    return app


# Simple command line interface
def main():
    """Simple CLI for the Device Registry API."""
    import argparse
    
    parser = argparse.ArgumentParser(description="Device Registry API")
    parser.add_argument("--export", choices=['json', 'yaml'], help="Export registry")
    parser.add_argument("--import", dest='import_file', help="Import registry from file")
    parser.add_argument("--format", choices=['json', 'yaml'], default='json', help="Import/export format")
    parser.add_argument("--stats", action='store_true', help="Show registry statistics")
    parser.add_argument("--protocols", action='store_true', help="List supported protocols")
    
    args = parser.parse_args()
    
    api = DeviceRegistryAPI()
    
    if args.export:
        print(api.export_registry(args.export))
    elif args.import_file:
        with open(args.import_file, 'r') as f:
            data = f.read()
        api.import_registry(data, args.format)
        print(f"Registry imported from {args.import_file}")
    elif args.stats:
        stats = api.get_registry_stats()
        print(json.dumps(stats, indent=2))
    elif args.protocols:
        protocols = api.get_supported_protocols()
        print(json.dumps(protocols, indent=2))
    else:
        print("Device Registry API - use --help for options")


if __name__ == "__main__":
    main()