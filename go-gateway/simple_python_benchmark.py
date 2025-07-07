#!/usr/bin/env python3
"""Simple Python Modbus Performance Benchmark

A simplified benchmark to compare with Go gateway performance
using the modern pymodbus 3.x API.
"""

import statistics
import time
from concurrent.futures import ThreadPoolExecutor

from pymodbus.client import ModbusTcpClient


def test_sequential_reads(host="127.0.0.1", port=502, iterations=1000):
    """Test sequential read performance."""
    print(f"Python Sequential Read Test ({iterations} iterations)...")

    client = ModbusTcpClient(host=host, port=port)

    if not client.connect():
        print("   ❌ Connection failed")
        return None

    latencies = []
    success_count = 0

    start_time = time.time()

    for i in range(iterations):
        read_start = time.time()
        try:
            # Read register 0 (equivalent to 40001)
            result = client.read_holding_registers(0, count=1, slave=1)
            read_end = time.time()

            if not result.isError():
                success_count += 1
                latencies.append(
                    (read_end - read_start) * 1000000
                )  # microseconds

        except Exception:
            pass

    end_time = time.time()
    total_time = end_time - start_time

    client.close()

    if success_count > 0:
        reads_per_second = success_count / total_time
        avg_latency_us = statistics.mean(latencies)

        print(f"   ✅ Success: {success_count}/{iterations}")
        print(f"   ✅ Performance: {reads_per_second:.0f} reads/second")
        print(f"   ✅ Average latency: {avg_latency_us:.1f}µs")
        print(f"   ✅ Total time: {total_time:.3f}s")

        return {
            "reads_per_second": reads_per_second,
            "avg_latency_us": avg_latency_us,
            "success_count": success_count,
            "total_time": total_time,
        }

    print("   ❌ All reads failed")
    return None


def worker_thread(thread_id, host, port, reads_per_thread):
    """Worker function for concurrent test."""
    client = ModbusTcpClient(host=host, port=port)

    if not client.connect():
        return 0, []

    latencies = []
    success_count = 0

    try:
        for i in range(reads_per_thread):
            read_start = time.time()
            result = client.read_holding_registers(0, count=1, slave=1)
            read_end = time.time()

            if not result.isError():
                success_count += 1
                latencies.append(
                    (read_end - read_start) * 1000000
                )  # microseconds

    except Exception:
        pass
    finally:
        client.close()

    return success_count, latencies


def test_concurrent_reads(
    host="127.0.0.1", port=502, num_threads=10, reads_per_thread=100
):
    """Test concurrent read performance."""
    print(
        f"Python Concurrent Read Test ({num_threads} threads, {reads_per_thread} reads each)..."
    )

    total_operations = num_threads * reads_per_thread

    start_time = time.time()

    with ThreadPoolExecutor(max_workers=num_threads) as executor:
        futures = [
            executor.submit(worker_thread, i, host, port, reads_per_thread)
            for i in range(num_threads)
        ]

        total_success = 0
        all_latencies = []

        for future in futures:
            try:
                success_count, latencies = future.result()
                total_success += success_count
                all_latencies.extend(latencies)
            except Exception:
                pass

    end_time = time.time()
    total_time = end_time - start_time

    if total_success > 0:
        ops_per_second = total_success / total_time
        avg_latency_us = statistics.mean(all_latencies)

        print(f"   ✅ Success: {total_success}/{total_operations}")
        print(f"   ✅ Performance: {ops_per_second:.0f} ops/second")
        print(f"   ✅ Average latency: {avg_latency_us:.1f}µs")
        print(f"   ✅ Total time: {total_time:.3f}s")

        return {
            "ops_per_second": ops_per_second,
            "avg_latency_us": avg_latency_us,
            "success_count": total_success,
            "total_time": total_time,
        }

    print("   ❌ All operations failed")
    return None


def test_write_operations(host="127.0.0.1", port=502, iterations=100):
    """Test write operation performance."""
    print(f"Python Write Test ({iterations} iterations)...")

    client = ModbusTcpClient(host=host, port=port)

    if not client.connect():
        print("   ❌ Connection failed")
        return None

    write_latencies = []
    success_count = 0

    start_time = time.time()

    for i in range(iterations):
        test_value = 1000 + i

        write_start = time.time()
        try:
            # Write to register 49 (equivalent to 40050)
            result = client.write_register(49, test_value, slave=1)
            write_end = time.time()

            if not result.isError():
                success_count += 1
                write_latencies.append(
                    (write_end - write_start) * 1000000
                )  # microseconds

        except Exception:
            pass

    end_time = time.time()
    total_time = end_time - start_time

    client.close()

    if success_count > 0:
        writes_per_second = success_count / total_time
        avg_latency_us = statistics.mean(write_latencies)

        print(f"   ✅ Success: {success_count}/{iterations}")
        print(f"   ✅ Performance: {writes_per_second:.0f} writes/second")
        print(f"   ✅ Average latency: {avg_latency_us:.1f}µs")
        print(f"   ✅ Total time: {total_time:.3f}s")

        return {
            "writes_per_second": writes_per_second,
            "avg_latency_us": avg_latency_us,
            "success_count": success_count,
            "total_time": total_time,
        }

    print("   ❌ All writes failed")
    return None


def main():
    """Run Python benchmarks and compare with Go results."""
    print("Python Modbus Performance Benchmark")
    print("=" * 50)

    # Test connectivity first
    print("\nTesting connectivity...")
    client = ModbusTcpClient(host="127.0.0.1", port=502)
    if not client.connect():
        print("❌ Cannot connect to Modbus simulator")
        print("Make sure the Python Modbus simulator is running on port 502")
        return 1

    print("✅ Connection successful")
    client.close()

    # Run benchmarks
    print("\nRunning benchmarks...")

    # Sequential reads
    sequential_result = test_sequential_reads(iterations=1000)

    # Concurrent reads
    concurrent_result = test_concurrent_reads(
        num_threads=10, reads_per_thread=100
    )

    # Write operations
    write_result = test_write_operations(iterations=100)

    # Compare with Go results
    print("\n" + "=" * 60)
    print("PERFORMANCE COMPARISON: Python vs Go Gateway")
    print("=" * 60)

    # Go results from our previous test
    go_sequential_rps = 19500
    go_concurrent_rps = 24079
    go_avg_latency_us = 51.282

    print(f"\n{'Metric':<30} {'Python':<15} {'Go':<15} {'Improvement':<15}")
    print("-" * 75)

    if sequential_result:
        python_rps = sequential_result["reads_per_second"]
        python_latency = sequential_result["avg_latency_us"]

        rps_improvement = go_sequential_rps / python_rps
        latency_improvement = python_latency / go_avg_latency_us

        print(
            f"{'Sequential Reads (ops/s)':<30} {python_rps:<15.0f} {go_sequential_rps:<15.0f} {rps_improvement:<15.1f}x"
        )
        print(
            f"{'Average Latency (µs)':<30} {python_latency:<15.1f} {go_avg_latency_us:<15.1f} {latency_improvement:<15.1f}x"
        )

    if concurrent_result:
        python_ops = concurrent_result["ops_per_second"]
        improvement = go_concurrent_rps / python_ops
        print(
            f"{'Concurrent Ops (ops/s)':<30} {python_ops:<15.0f} {go_concurrent_rps:<15.0f} {improvement:<15.1f}x"
        )

    if write_result:
        python_writes = write_result["writes_per_second"]
        print(
            f"{'Write Ops (ops/s)':<30} {python_writes:<15.0f} {'N/A':<15} {'N/A':<15}"
        )

    print("\n🎯 Benchmark completed!")

    return 0


if __name__ == "__main__":
    exit(main())
