#!/usr/bin/env python3
"""Debug Python Modbus reads."""

from pymodbus.client import ModbusTcpClient


def test_simple_read() -> None:
    """Test a simple read operation with debug info."""
    print("Testing simple Modbus read...")

    client = ModbusTcpClient(host="127.0.0.1", port=502)

    if not client.connect():
        print("❌ Connection failed")
        return

    print("✅ Connected to Modbus device")

    try:
        # Try reading holding register 0 (40001)
        print("Reading holding register 0...")
        result = client.read_holding_registers(0, count=1, slave=1)

        print(f"Result type: {type(result)}")
        print(f"Result: {result}")

        if hasattr(result, "isError"):
            print(f"Is error: {result.isError()}")
            if result.isError():
                print(f"Error details: {result}")
            else:
                print(f"Success! Value: {result.registers}")

        # Try without slave parameter
        print("\nReading holding register 0 without slave parameter...")
        result2 = client.read_holding_registers(0, count=1)
        print(f"Result2: {result2}")

        if hasattr(result2, "isError"):
            print(f"Is error: {result2.isError()}")
            if not result2.isError():
                print(f"Success! Value: {result2.registers}")

    except Exception as e:
        print(f"Exception occurred: {e}")
        import traceback

        traceback.print_exc()

    finally:
        client.close()
        print("✅ Connection closed")


if __name__ == "__main__":
    test_simple_read()
