#!/usr/bin/env python3
"""
Virtual Device Test Suite

This script tests all virtual industrial devices to ensure they are working correctly
and can be discovered through their respective protocols.

Usage:
    python test_virtual_devices.py [--host HOST] [--timeout TIMEOUT]
"""

import argparse
import asyncio
import logging
import socket
import sys
import time
from typing import Dict, List, Optional, Tuple

# Test results
TestResult = Dict[str, any]

def setup_logging(level: str = "INFO"):
    """Setup logging configuration."""
    logging.basicConfig(
        level=getattr(logging, level.upper()),
        format="%(asctime)s - %(levelname)s - %(message)s",
        datefmt="%Y-%m-%d %H:%M:%S"
    )
    return logging.getLogger(__name__)

def test_tcp_connection(host: str, port: int, timeout: float = 5.0) -> TestResult:
    """Test TCP connection to a service."""
    start_time = time.time()
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(timeout)
        result = sock.connect_ex((host, port))
        sock.close()
        
        elapsed = time.time() - start_time
        
        if result == 0:
            return {
                "success": True,
                "error": None,
                "response_time": elapsed,
                "message": f"Successfully connected to {host}:{port}"
            }
        else:
            return {
                "success": False,
                "error": f"Connection failed with code {result}",
                "response_time": elapsed,
                "message": f"Failed to connect to {host}:{port}"
            }
    except Exception as e:
        elapsed = time.time() - start_time
        return {
            "success": False,
            "error": str(e),
            "response_time": elapsed,
            "message": f"Exception connecting to {host}:{port}: {e}"
        }

def test_udp_connection(host: str, port: int, timeout: float = 5.0) -> TestResult:
    """Test UDP connection to a service."""
    start_time = time.time()
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        sock.settimeout(timeout)
        
        # Send a simple test packet
        test_data = b"TEST"
        sock.sendto(test_data, (host, port))
        
        # Try to receive (may timeout, which is ok for UDP)
        try:
            data, addr = sock.recvfrom(1024)
        except socket.timeout:
            pass  # Expected for UDP services that don't respond to arbitrary data
        
        sock.close()
        elapsed = time.time() - start_time
        
        return {
            "success": True,
            "error": None,
            "response_time": elapsed,
            "message": f"Successfully sent UDP packet to {host}:{port}"
        }
    except Exception as e:
        elapsed = time.time() - start_time
        return {
            "success": False,
            "error": str(e),
            "response_time": elapsed,
            "message": f"Exception with UDP {host}:{port}: {e}"
        }

async def test_modbus_tcp(host: str, port: int) -> TestResult:
    """Test Modbus TCP device."""
    try:
        # Try to import modbus library for more detailed testing
        try:
            from pymodbus.client import ModbusTcpClient
            
            client = ModbusTcpClient(host, port=port, timeout=3)
            connection = client.connect()
            
            if connection:
                # Try to read holding registers
                try:
                    result = client.read_holding_registers(0, 10, slave=1)
                    if not result.isError():
                        client.close()
                        return {
                            "success": True,
                            "error": None,
                            "message": f"Modbus TCP device responding on {host}:{port}",
                            "data": f"Read {len(result.registers)} registers"
                        }
                    else:
                        client.close()
                        return {
                            "success": False,
                            "error": str(result),
                            "message": f"Modbus TCP device error on {host}:{port}"
                        }
                except Exception as e:
                    client.close()
                    return {
                        "success": False,
                        "error": str(e),
                        "message": f"Modbus TCP read error on {host}:{port}"
                    }
            else:
                return {
                    "success": False,
                    "error": "Connection failed",
                    "message": f"Cannot connect to Modbus TCP device on {host}:{port}"
                }
        except ImportError:
            # Fall back to basic TCP test
            return test_tcp_connection(host, port)
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "message": f"Modbus TCP test exception: {e}"
        }

async def test_opcua(host: str, port: int) -> TestResult:
    """Test OPC UA server."""
    try:
        # Try to import OPC UA library for more detailed testing
        try:
            from asyncua import Client
            
            url = f"opc.tcp://{host}:{port}"
            client = Client(url=url)
            client.set_timeout(5.0)
            
            await client.connect()
            
            # Try to browse the root node
            root = client.get_root_node()
            children = await root.get_children()
            
            await client.disconnect()
            
            return {
                "success": True,
                "error": None,
                "message": f"OPC UA server responding on {host}:{port}",
                "data": f"Found {len(children)} root children"
            }
        except ImportError:
            # Fall back to basic TCP test
            return test_tcp_connection(host, port)
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "message": f"OPC UA test failed: {e}"
        }

async def test_s7(host: str, port: int) -> TestResult:
    """Test S7 PLC."""
    try:
        # Try to import snap7 for more detailed testing
        try:
            import snap7
            
            client = snap7.client.Client()
            client.set_connection_params(host, 0, 1)  # rack=0, slot=1
            client.connect(host, 0, 1)
            
            if client.get_connected():
                # Try to read a small area
                try:
                    data = client.read_area(snap7.types.areas.DB, 1, 0, 10)
                    client.disconnect()
                    return {
                        "success": True,
                        "error": None,
                        "message": f"S7 PLC responding on {host}:{port}",
                        "data": f"Read {len(data)} bytes from DB1"
                    }
                except Exception as e:
                    client.disconnect()
                    return {
                        "success": False,
                        "error": str(e),
                        "message": f"S7 PLC read error: {e}"
                    }
            else:
                return {
                    "success": False,
                    "error": "Connection failed",
                    "message": f"Cannot connect to S7 PLC on {host}:{port}"
                }
        except ImportError:
            # Fall back to basic TCP test
            return test_tcp_connection(host, port)
    except Exception as e:
        return {
            "success": False,
            "error": str(e),
            "message": f"S7 test exception: {e}"
        }

async def run_all_tests(host: str = "localhost", timeout: float = 10.0) -> Dict[str, TestResult]:
    """Run all virtual device tests."""
    logger = setup_logging()
    
    # Define test cases
    test_cases = [
        ("modbus-tcp-sim", "Modbus TCP Factory PLC", test_modbus_tcp, host, 502),
        ("modbus-rtu-sim", "Modbus RTU Energy Meter", test_modbus_tcp, host, 503),
        ("modbus-rtu-sim-temp", "Modbus RTU Temperature Controller", test_modbus_tcp, host, 504),
        ("opcua-sim", "OPC UA Factory Server", test_opcua, host, 4840),
        ("opcua-sim-process", "OPC UA Process Server", test_opcua, host, 4841),
        ("s7-sim", "S7 PLC", test_s7, host, 102),
        ("ethernet-ip-sim-tcp", "Ethernet/IP TCP", lambda h, p: test_tcp_connection(h, p), host, 2222),
        ("ethernet-ip-sim-udp", "Ethernet/IP UDP", lambda h, p: test_udp_connection(h, p), host, 44818),
    ]
    
    results = {}
    
    logger.info(f"Starting virtual device tests on host: {host}")
    logger.info(f"Test timeout: {timeout} seconds")
    logger.info("=" * 80)
    
    for test_id, description, test_func, test_host, test_port in test_cases:
        logger.info(f"Testing {description} ({test_host}:{test_port})...")
        
        try:
            if asyncio.iscoroutinefunction(test_func):
                result = await asyncio.wait_for(test_func(test_host, test_port), timeout=timeout)
            else:
                result = test_func(test_host, test_port)
            
            results[test_id] = result
            
            if result["success"]:
                logger.info(f"  ✅ {result['message']}")
                if "data" in result:
                    logger.info(f"     {result['data']}")
            else:
                logger.error(f"  ❌ {result['message']}")
                if result["error"]:
                    logger.error(f"     Error: {result['error']}")
        
        except asyncio.TimeoutError:
            logger.error(f"  ⏰ Test timed out after {timeout} seconds")
            results[test_id] = {
                "success": False,
                "error": "Timeout",
                "message": f"Test timed out after {timeout} seconds"
            }
        except Exception as e:
            logger.error(f"  💥 Test exception: {e}")
            results[test_id] = {
                "success": False,
                "error": str(e),
                "message": f"Test exception: {e}"
            }
    
    return results

def print_summary(results: Dict[str, TestResult]):
    """Print test summary."""
    logger = setup_logging()
    
    total_tests = len(results)
    successful_tests = sum(1 for r in results.values() if r["success"])
    failed_tests = total_tests - successful_tests
    
    logger.info("=" * 80)
    logger.info("TEST SUMMARY")
    logger.info("=" * 80)
    logger.info(f"Total Tests: {total_tests}")
    logger.info(f"Successful: {successful_tests}")
    logger.info(f"Failed: {failed_tests}")
    logger.info(f"Success Rate: {(successful_tests/total_tests)*100:.1f}%")
    
    if failed_tests > 0:
        logger.info("\nFailed Tests:")
        for test_id, result in results.items():
            if not result["success"]:
                logger.info(f"  - {test_id}: {result['message']}")
    
    logger.info("=" * 80)

async def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Virtual Device Test Suite")
    parser.add_argument("--host", default="localhost", help="Host to test (default: localhost)")
    parser.add_argument("--timeout", type=float, default=10.0, help="Test timeout in seconds (default: 10)")
    parser.add_argument("--log-level", default="INFO", choices=["DEBUG", "INFO", "WARNING", "ERROR"])
    
    args = parser.parse_args()
    
    # Setup logging
    setup_logging(args.log_level)
    
    # Run tests
    results = await run_all_tests(args.host, args.timeout)
    
    # Print summary
    print_summary(results)
    
    # Exit with appropriate code
    failed_tests = sum(1 for r in results.values() if not r["success"])
    sys.exit(1 if failed_tests > 0 else 0)

if __name__ == "__main__":
    asyncio.run(main())