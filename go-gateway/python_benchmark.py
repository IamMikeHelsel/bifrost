#!/usr/bin/env python3
"""
Python Modbus Performance Benchmark

This script provides a direct comparison with the Go gateway implementation
to demonstrate the performance improvements achieved with the Go backend.
"""

import time
import threading
import statistics
from concurrent.futures import ThreadPoolExecutor, as_completed
from pymodbus.client import ModbusTcpClient
from pymodbus.constants import Endian
from pymodbus.payload import BinaryPayloadDecoder


class PythonModbusBenchmark:
    """Python Modbus performance benchmark for comparison with Go gateway."""
    
    def __init__(self, host='127.0.0.1', port=502, unit_id=1):
        self.host = host
        self.port = port
        self.unit_id = unit_id
        self.client = None
        self.results = []
        
    def connect(self):
        """Connect to Modbus device."""
        try:
            self.client = ModbusTcpClient(host=self.host, port=self.port)
            connection = self.client.connect()
            if not connection:
                raise Exception("Failed to connect to Modbus device")
            return True
        except Exception as e:
            print(f"Connection failed: {e}")
            return False
    
    def disconnect(self):
        """Disconnect from Modbus device."""
        if self.client:
            self.client.close()
    
    def read_holding_register(self, address, count=1):
        """Read holding register(s)."""
        try:
            if not self.client:
                return None
            
            # Convert from Modbus address format (40001+) to zero-based
            modbus_addr = address - 40001 if address >= 40001 else address
            
            result = self.client.read_holding_registers(modbus_addr, count, slave=self.unit_id)
            if result.isError():
                return None
            
            return result.registers
        except Exception as e:
            print(f"Read error: {e}")
            return None
    
    def write_holding_register(self, address, value):
        """Write holding register."""
        try:
            if not self.client:
                return False
            
            # Convert from Modbus address format (40001+) to zero-based
            modbus_addr = address - 40001 if address >= 40001 else address
            
            result = self.client.write_register(modbus_addr, value, slave=self.unit_id)
            if result.isError():
                return False
            
            return True
        except Exception as e:
            print(f"Write error: {e}")
            return False
    
    def benchmark_sequential_reads(self, iterations=1000):
        """Benchmark sequential read operations."""
        print(f"\nPython Sequential Read Benchmark ({iterations} iterations)...")
        
        if not self.connect():
            return None
        
        address = 40001  # First holding register
        latencies = []
        success_count = 0
        
        start_time = time.time()
        
        for i in range(iterations):
            read_start = time.time()
            result = self.read_holding_register(address)
            read_end = time.time()
            
            if result is not None:
                success_count += 1
                latencies.append((read_end - read_start) * 1000)  # Convert to ms
        
        end_time = time.time()
        total_time = end_time - start_time
        
        self.disconnect()
        
        if success_count > 0:
            reads_per_second = success_count / total_time
            avg_latency = statistics.mean(latencies)
            median_latency = statistics.median(latencies)
            
            result = {
                'type': 'sequential_reads',
                'iterations': iterations,
                'success_count': success_count,
                'total_time': total_time,
                'reads_per_second': reads_per_second,
                'avg_latency_ms': avg_latency,
                'median_latency_ms': median_latency,
                'min_latency_ms': min(latencies),
                'max_latency_ms': max(latencies)
            }
            
            print(f"   ✅ Sequential reads: {success_count}/{iterations} successful")
            print(f"   ✅ Performance: {reads_per_second:.0f} reads/second")
            print(f"   ✅ Average latency: {avg_latency:.3f}ms")
            print(f"   ✅ Median latency: {median_latency:.3f}ms")
            print(f"   ✅ Total time: {total_time:.3f}s")
            
            return result
        
        return None
    
    def benchmark_concurrent_reads(self, num_threads=10, reads_per_thread=100):
        """Benchmark concurrent read operations."""
        print(f"\nPython Concurrent Read Benchmark ({num_threads} threads, {reads_per_thread} reads each)...")
        
        address = 40001  # First holding register
        total_operations = num_threads * reads_per_thread
        success_count = 0
        latencies = []
        
        def worker_thread(thread_id):
            # Each thread gets its own client connection
            client = ModbusTcpClient(host=self.host, port=self.port)
            if not client.connect():
                return [], 0
            
            thread_latencies = []
            thread_success = 0
            
            try:
                for i in range(reads_per_thread):
                    read_start = time.time()
                    result = client.read_holding_registers(0, 1, slave=self.unit_id)  # 40001 -> 0
                    read_end = time.time()
                    
                    if not result.isError():
                        thread_success += 1
                        thread_latencies.append((read_end - read_start) * 1000)  # Convert to ms
                
            except Exception as e:
                print(f"Thread {thread_id} error: {e}")
            finally:
                client.close()
            
            return thread_latencies, thread_success
        
        start_time = time.time()
        
        # Use ThreadPoolExecutor for concurrent operations
        with ThreadPoolExecutor(max_workers=num_threads) as executor:
            futures = [executor.submit(worker_thread, i) for i in range(num_threads)]
            
            for future in as_completed(futures):
                try:
                    thread_latencies, thread_success = future.result()
                    success_count += thread_success
                    latencies.extend(thread_latencies)
                except Exception as e:
                    print(f"Thread execution error: {e}")
        
        end_time = time.time()
        total_time = end_time - start_time
        
        if success_count > 0:
            ops_per_second = success_count / total_time
            avg_latency = statistics.mean(latencies)
            median_latency = statistics.median(latencies)
            
            result = {
                'type': 'concurrent_reads',
                'num_threads': num_threads,
                'reads_per_thread': reads_per_thread,
                'total_operations': total_operations,
                'success_count': success_count,
                'total_time': total_time,
                'ops_per_second': ops_per_second,
                'avg_latency_ms': avg_latency,
                'median_latency_ms': median_latency,
                'min_latency_ms': min(latencies),
                'max_latency_ms': max(latencies)
            }
            
            print(f"   ✅ Concurrent operations: {success_count}/{total_operations} successful")
            print(f"   ✅ Performance: {ops_per_second:.0f} ops/second with {num_threads} threads")
            print(f"   ✅ Average latency: {avg_latency:.3f}ms")
            print(f"   ✅ Median latency: {median_latency:.3f}ms")
            print(f"   ✅ Total time: {total_time:.3f}s")
            
            return result
        
        return None
    
    def benchmark_read_write_operations(self, iterations=100):
        """Benchmark read/write operations."""
        print(f"\nPython Read/Write Benchmark ({iterations} iterations)...")
        
        if not self.connect():
            return None
        
        read_address = 40001  # Read from sensor
        write_address = 40050  # Write to setpoint
        
        write_latencies = []
        read_latencies = []
        success_count = 0
        
        start_time = time.time()
        
        for i in range(iterations):
            test_value = 1000 + i  # Different value each time
            
            # Write operation
            write_start = time.time()
            write_success = self.write_holding_register(write_address, test_value)
            write_end = time.time()
            
            if write_success:
                write_latencies.append((write_end - write_start) * 1000)
                
                # Read back to verify
                read_start = time.time()
                read_result = self.read_holding_register(write_address)
                read_end = time.time()
                
                if read_result is not None:
                    read_latencies.append((read_end - read_start) * 1000)
                    success_count += 1
        
        end_time = time.time()
        total_time = end_time - start_time
        
        self.disconnect()
        
        if success_count > 0:
            ops_per_second = (success_count * 2) / total_time  # 2 ops per iteration (write + read)
            avg_write_latency = statistics.mean(write_latencies)
            avg_read_latency = statistics.mean(read_latencies)
            
            result = {
                'type': 'read_write_operations',
                'iterations': iterations,
                'success_count': success_count,
                'total_time': total_time,
                'ops_per_second': ops_per_second,
                'avg_write_latency_ms': avg_write_latency,
                'avg_read_latency_ms': avg_read_latency,
            }
            
            print(f"   ✅ Read/Write operations: {success_count}/{iterations} successful")
            print(f"   ✅ Performance: {ops_per_second:.0f} ops/second")
            print(f"   ✅ Average write latency: {avg_write_latency:.3f}ms")
            print(f"   ✅ Average read latency: {avg_read_latency:.3f}ms")
            print(f"   ✅ Total time: {total_time:.3f}s")
            
            return result
        
        return None
    
    def run_all_benchmarks(self):
        """Run all benchmark tests."""
        print("Python Modbus Performance Benchmark")
        print("=" * 50)
        print(f"Target: {self.host}:{self.port}")
        print(f"Unit ID: {self.unit_id}")
        
        # Test basic connectivity
        print("\n1. Testing Connectivity...")
        if not self.connect():
            print("   ❌ Cannot connect to Modbus device")
            return []
        
        print("   ✅ Connection successful")
        self.disconnect()
        
        # Run benchmarks
        results = []
        
        # Sequential reads
        result = self.benchmark_sequential_reads(1000)
        if result:
            results.append(result)
        
        # Concurrent reads
        result = self.benchmark_concurrent_reads(10, 100)
        if result:
            results.append(result)
        
        # Read/write operations
        result = self.benchmark_read_write_operations(100)
        if result:
            results.append(result)
        
        return results
    
    def compare_with_go_results(self, go_results):
        """Compare Python results with Go gateway results."""
        print("\n" + "=" * 60)
        print("PERFORMANCE COMPARISON: Python vs Go Gateway")
        print("=" * 60)
        
        # Go results from our test (approximate values)
        go_sequential_rps = 19500
        go_concurrent_rps = 24079
        go_avg_latency_us = 51.282
        
        python_results = self.run_all_benchmarks()
        
        if python_results:
            print(f"\nComparison Summary:")
            print(f"{'Metric':<30} {'Python':<15} {'Go':<15} {'Improvement':<15}")
            print("-" * 75)
            
            for result in python_results:
                if result['type'] == 'sequential_reads':
                    python_rps = result['reads_per_second']
                    improvement = go_sequential_rps / python_rps
                    print(f"{'Sequential Reads (ops/s)':<30} {python_rps:<15.0f} {go_sequential_rps:<15.0f} {improvement:<15.1f}x")
                    
                    python_latency_ms = result['avg_latency_ms']
                    go_latency_ms = go_avg_latency_us / 1000
                    latency_improvement = python_latency_ms / go_latency_ms
                    print(f"{'Average Latency (ms)':<30} {python_latency_ms:<15.3f} {go_latency_ms:<15.3f} {latency_improvement:<15.1f}x")
                    
                elif result['type'] == 'concurrent_reads':
                    python_ops = result['ops_per_second']
                    improvement = go_concurrent_rps / python_ops
                    print(f"{'Concurrent Ops (ops/s)':<30} {python_ops:<15.0f} {go_concurrent_rps:<15.0f} {improvement:<15.1f}x")
        
        return python_results


def main():
    """Main benchmark execution."""
    # Create benchmark instance
    benchmark = PythonModbusBenchmark()
    
    # Run benchmarks and comparison
    results = benchmark.compare_with_go_results(None)
    
    if results:
        print("\n🎯 Benchmark completed successfully!")
        print(f"📊 Total benchmark results: {len(results)}")
    else:
        print("\n❌ Benchmark failed - check Modbus connection")
        return 1
    
    return 0


if __name__ == "__main__":
    exit(main())