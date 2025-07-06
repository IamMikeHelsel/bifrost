import asyncio

from bifrost.modbus import ModbusConnection, ModbusDevice
from bifrost_core import DataType, Tag


async def main():
    # Replace with your Modbus device's IP address and port
    modbus_host = "192.168.1.100"  # Example IP, change this!
    modbus_port = 502

    connection = ModbusConnection(host=modbus_host, port=modbus_port)
    device = ModbusDevice(connection)

    try:
        async with connection:
            print(f"Connected to Modbus device at {modbus_host}:{modbus_port}")

            # Example 1: Read a single holding register
            tag_single = Tag(name="MotorSpeed", address="40001", data_type=DataType.INT16)
            readings_single = await device.read([tag_single])
            if tag_single in readings_single:
                print(f"Read {tag_single.name}: {readings_single[tag_single].value}")
            else:
                print(f"Failed to read {tag_single.name}")

            # Example 2: Write a single holding register
            write_value = 1234
            print(f"Attempting to write {write_value} to {tag_single.name}")
            await device.write({tag_single: write_value})
            print("Write operation attempted.")

            # Verify the write by reading again
            readings_after_write = await device.read([tag_single])
            if tag_single in readings_after_write:
                print(f"Verified {tag_single.name} after write: {readings_after_write[tag_single].value}")

            # Example 3: Read multiple holding registers
            # Assuming registers 40002 to 40006 (5 registers) contain INT16 values
            tag_multiple = Tag(name="SensorData", address="40002:5", data_type=DataType.INT16)
            readings_multiple = await device.read([tag_multiple])
            if tag_multiple in readings_multiple:
                print(f"Read {tag_multiple.name}: {readings_multiple[tag_multiple].value}")
            else:
                print(f"Failed to read {tag_multiple.name}")

            # Example 4: Write multiple holding registers
            write_values = [10, 20, 30, 40, 50]
            tag_write_multiple = Tag(name="ControlValues", address="40007:5", data_type=DataType.INT16)
            print(f"Attempting to write {write_values} to {tag_write_multiple.name}")
            await device.write({tag_write_multiple: write_values})
            print("Multiple write operation attempted.")

            # Verify the multiple write by reading again
            readings_after_multi_write = await device.read([tag_write_multiple])
            if tag_write_multiple in readings_after_multi_write:
                print(f"Verified {tag_write_multiple.name} after multi-write: {readings_after_multi_write[tag_write_multiple].value}")

    except Exception as e:
        print(f"An error occurred: {e}")
    finally:
        print("Disconnected from Modbus device.")


if __name__ == "__main__":
    asyncio.run(main())
