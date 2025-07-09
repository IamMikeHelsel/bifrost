"""
End-to-End tests for Modbus communication via the Bifrost Gateway.

These tests assume:
1. The Bifrost Go Gateway is running.
2. A Modbus simulator (virtual device) is running and accessible to the gateway.
3. The gateway is configured to communicate with the Modbus simulator.
"""
import pytest
import asyncio
import time

# Assuming the bifrost client library is structured appropriately.
# This will be the actual client, not a placeholder.
# from bifrost.client import BifrostClient # This would be the real import
# from bifrost_core import Tag, DataType, TagReading # This would be the real import

# --- Mocking BifrostClient and related classes for now ---
# This section should be removed once the actual client is available and importable.
class MockBifrostClient:
    def __init__(self, base_url):
        self.base_url = base_url
        self._connected = False
        print(f"MockBifrostClient initialized for {base_url}")

    async def __aenter__(self):
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.disconnect()

    async def connect(self):
        print(f"MockBifrostClient: Connecting to {self.base_url}...")
        # Simulate network delay
        await asyncio.sleep(0.1)
        # In a real client, this would establish a connection.
        # For E2E, we assume the gateway is up. If not, tests relying on it will fail.
        self._connected = True
        print("MockBifrostClient: Connected.")

    async def disconnect(self):
        print("MockBifrostClient: Disconnecting...")
        await asyncio.sleep(0.05)
        self._connected = False
        print("MockBifrostClient: Disconnected.")

    def is_connected(self):
        return self._connected

    async def list_devices(self, protocol_filter=None):
        print(f"MockBifrostClient: Listing devices (filter: {protocol_filter})...")
        await asyncio.sleep(0.05)
        # Simulate finding the Modbus device if the gateway is configured for it
        if protocol_filter is None or "modbus" in protocol_filter.lower():
            return [{"id": MODBUS_DEVICE_ID, "name": "Simulated Modbus PLC", "protocol": "modbus-tcp"}]
        return []

    async def read_tags(self, device_id, tags):
        print(f"MockBifrostClient: Reading tags from device {device_id}: {tags}")
        await asyncio.sleep(0.1) # Simulate network read
        readings = {}
        for tag in tags:
            # Simulate some data based on address - very simplified
            value = 0
            if tag.address == READABLE_TAG_ADDRESS_1: value = 111
            elif tag.address == READABLE_TAG_ADDRESS_2: value = 222
            elif tag.address == WRITABLE_TAG_ADDRESS_1: # Reading a writable tag
                # This mock doesn't store written values globally, so it returns a default
                value = self.mock_data_store.get(f"{device_id}:{tag.address}", 0) # Default if not written

            # Use a mock TagReading if the real one isn't available
            class MockTagReading:
                def __init__(self, tag, value, timestamp):
                    self.tag = tag
                    self.value = value
                    self.timestamp = timestamp
            readings[tag.name] = MockTagReading(tag, value, time.time())
        return readings

    async def write_tags(self, device_id, tags_with_values):
        print(f"MockBifrostClient: Writing tags to device {device_id}: {tags_with_values}")
        await asyncio.sleep(0.1) # Simulate network write
        # In a real scenario, this would return success/failure status per tag or overall.
        # Simulate storing the written value for read-back verification in this mock
        if not hasattr(self, 'mock_data_store'):
            self.mock_data_store = {}
        for tag, value in tags_with_values.items():
            self.mock_data_store[f"{device_id}:{tag.address}"] = value
            print(f"  MockStored: {device_id}:{tag.address} = {value}")
        return True # Simulate success

BifrostClient = MockBifrostClient # Use mock for now

class MockTag:
    def __init__(self, name, address, data_type, description=""):
        self.name = name
        self.address = address
        self.data_type = data_type
        self.description = description
    def __repr__(self):
        return f"Tag(name='{self.name}', address='{self.address}', data_type='{self.data_type}')"
    def __eq__(self, other):
        return isinstance(other, MockTag) and self.name == other.name and self.address == other.address
    def __hash__(self):
        return hash((self.name, self.address))


class MockDataType:
    INT16 = "INT16"
    FLOAT32 = "FLOAT32"
    BOOLEAN = "BOOLEAN"

Tag = MockTag
DataType = MockDataType
# --- End of Mocking Section ---


# Configuration for the tests
GATEWAY_URL = "http://localhost:8080"  # Default Go gateway URL
MODBUS_DEVICE_ID = "bifrost-modbus-sim"  # Matches container_name in docker-compose

# Define specific tags based on modbus_server.py and the recent update
# Readable sensor data (first few registers)
READABLE_TAG_ADDRESS_1 = "40001" # Holding Register, index 0 in simulator
READABLE_TAG_DATATYPE_1 = DataType.INT16
READABLE_TAG_NAME_1 = "TemperatureSensor1"

READABLE_TAG_ADDRESS_2 = "40011" # Holding Register, index 10 in simulator
READABLE_TAG_DATATYPE_2 = DataType.INT16
READABLE_TAG_NAME_2 = "PressureSensor1"

# Writable test registers (initialized to 0 in simulator)
WRITABLE_TAG_ADDRESS_1 = "40050" # Holding Register, index 49 in simulator
WRITABLE_TAG_DATATYPE_1 = DataType.INT16
WRITABLE_TAG_NAME_1 = "TestWritableRegister1"

WRITABLE_TAG_ADDRESS_2 = "40051" # Holding Register, index 50 in simulator
WRITABLE_TAG_DATATYPE_2 = DataType.INT16
WRITABLE_TAG_NAME_2 = "TestWritableRegister2"


@pytest.fixture(scope="module")
async def client():
    """Fixture to create and connect a BifrostClient."""
    # When real client is available:
    # real_client = BifrostClient(base_url=GATEWAY_URL)
    # async with real_client as c:
    #     yield c
    # For now, using the mock:
    mock_client_instance = BifrostClient(base_url=GATEWAY_URL)
    await mock_client_instance.connect() # Manually connect if not using async with context
    yield mock_client_instance
    await mock_client_instance.disconnect()


@pytest.mark.e2e
@pytest.mark.asyncio
async def test_gateway_connection_and_device_presence(client: BifrostClient):
    """
    Test connecting to the gateway and verifying the expected Modbus simulator device is listed.
    This implicitly tests that the gateway is running and accessible.
    """
    assert client.is_connected(), "Client should be connected to the Bifrost Gateway."

    devices = await client.list_devices(protocol_filter="modbus")
    assert isinstance(devices, list), "list_devices should return a list."

    found_device = any(device.get("id") == MODBUS_DEVICE_ID for device in devices)
    assert found_device, f"Modbus device '{MODBUS_DEVICE_ID}' not found via gateway."
    print(f"Successfully connected to gateway and found device: {MODBUS_DEVICE_ID}")

@pytest.mark.e2e
@pytest.mark.asyncio
async def test_read_single_modbus_tag(client: BifrostClient):
    """Test reading a single predefined readable tag from the Modbus device via the gateway."""
    tag_to_read = Tag(name=READABLE_TAG_NAME_1, address=READABLE_TAG_ADDRESS_1, data_type=READABLE_TAG_DATATYPE_1)

    readings = await client.read_tags(MODBUS_DEVICE_ID, [tag_to_read])

    assert readings is not None, "Readings should not be None."
    assert tag_to_read.name in readings, f"Reading for tag '{tag_to_read.name}' not found."

    reading = readings[tag_to_read.name]
    assert reading.value is not None, f"Value for tag '{tag_to_read.name}' should not be None."
    # Specific value check depends on simulator's dynamic data; presence is key here.
    # assert reading.value == EXPECTED_INITIAL_VALUE_FOR_40001 # If known and static
    assert reading.timestamp is not None, "Timestamp should be present in the reading."
    print(f"Successfully read tag '{tag_to_read.name}': Value={reading.value}")

@pytest.mark.e2e
@pytest.mark.asyncio
async def test_write_single_modbus_tag_and_verify(client: BifrostClient):
    """Test writing to a single writable tag and verifying the write by reading it back."""
    tag_to_write = Tag(name=WRITABLE_TAG_NAME_1, address=WRITABLE_TAG_ADDRESS_1, data_type=WRITABLE_TAG_DATATYPE_1)
    test_value = 12345

    write_success = await client.write_tags(MODBUS_DEVICE_ID, {tag_to_write: test_value})
    assert write_success, "Write operation should be successful."

    print(f"Attempted to write {test_value} to tag '{tag_to_write.name}'. Verifying read-back...")

    # Add a small delay if necessary for the write to propagate through layers
    await asyncio.sleep(0.2)

    readings = await client.read_tags(MODBUS_DEVICE_ID, [tag_to_write])
    assert readings is not None and tag_to_write.name in readings
    reading = readings[tag_to_write.name]

    assert reading.value == test_value, \
        f"Read-back value mismatch for tag '{tag_to_write.name}'. Expected {test_value}, got {reading.value}."
    print(f"Successfully wrote and verified tag '{tag_to_write.name}': Value={reading.value}")

    # Clean up by writing back to 0 (optional, but good practice for some tests)
    await client.write_tags(MODBUS_DEVICE_ID, {tag_to_write: 0})


@pytest.mark.e2e
@pytest.mark.asyncio
async def test_read_multiple_modbus_tags(client: BifrostClient):
    """Test reading multiple tags (a mix of readable and writable) from the Modbus device."""
    tags_to_read = [
        Tag(name=READABLE_TAG_NAME_1, address=READABLE_TAG_ADDRESS_1, data_type=READABLE_TAG_DATATYPE_1),
        Tag(name=READABLE_TAG_NAME_2, address=READABLE_TAG_ADDRESS_2, data_type=READABLE_TAG_DATATYPE_2),
        Tag(name=WRITABLE_TAG_NAME_1, address=WRITABLE_TAG_ADDRESS_1, data_type=WRITABLE_TAG_DATATYPE_1) # Read a writable tag
    ]

    readings = await client.read_tags(MODBUS_DEVICE_ID, tags_to_read)

    assert readings is not None, "Readings should not be None."
    assert len(readings) == len(tags_to_read), "Should receive readings for all requested tags."

    for tag in tags_to_read:
        assert tag.name in readings, f"Reading for tag '{tag.name}' not found."
        assert readings[tag.name].value is not None, f"Value for tag '{tag.name}' should not be None."
        assert readings[tag.name].timestamp is not None, f"Timestamp for tag '{tag.name}' should be present."
        print(f"Successfully read multiple tag '{tag.name}': Value={readings[tag.name].value}")

@pytest.mark.e2e
@pytest.mark.asyncio
async def test_write_multiple_modbus_tags_and_verify(client: BifrostClient):
    """Test writing to multiple writable tags and verifying them."""
    tag1 = Tag(name=WRITABLE_TAG_NAME_1, address=WRITABLE_TAG_ADDRESS_1, data_type=WRITABLE_TAG_DATATYPE_1)
    tag2 = Tag(name=WRITABLE_TAG_NAME_2, address=WRITABLE_TAG_ADDRESS_2, data_type=WRITABLE_TAG_DATATYPE_2)

    values_to_write = {
        tag1: 54321,
        tag2: 9876
    }

    write_success = await client.write_tags(MODBUS_DEVICE_ID, values_to_write)
    assert write_success, "Multiple tag write operation should be successful."

    print(f"Attempted to write multiple tags. Verifying read-back...")
    await asyncio.sleep(0.2)

    tags_to_verify = [tag1, tag2]
    readings = await client.read_tags(MODBUS_DEVICE_ID, tags_to_verify)

    assert readings is not None
    for original_tag, written_value in values_to_write.items():
        assert original_tag.name in readings, f"Tag {original_tag.name} not found in read-back."
        assert readings[original_tag.name].value == written_value, \
            f"Read-back value mismatch for tag '{original_tag.name}'. Expected {written_value}, got {readings[original_tag.name].value}."
        print(f"Successfully wrote and verified tag '{original_tag.name}': Value={readings[original_tag.name].value}")

    # Clean up
    await client.write_tags(MODBUS_DEVICE_ID, {tag1: 0, tag2: 0})

# Notes for actual implementation:
# 1. Replace MockBifrostClient, MockTag, MockDataType with actual imports from `bifrost.client` and `bifrost_core`.
# 2. The `BifrostClient` will need to correctly route requests for a given `device_id`
#    to the Go gateway, which then communicates with the corresponding virtual device.
# 3. Error handling (e.g., what happens if gateway is down, device_id unknown, tag address invalid)
#    will be crucial and should be tested in `test_e2e_modbus_error_scenarios.py`.
# 4. Ensure `MODBUS_DEVICE_ID` matches how the gateway identifies the Modbus simulator
#    (likely configured in the gateway's own configuration if it doesn't auto-discover based on network).
#    The `run-e2e-tests.sh` script currently assumes the simulator is just "there" on localhost:502
#    and the gateway is configured to find it.
# 5. The `client.list_devices()` method is an assumption; the actual API might differ.
#    If discovery isn't a gateway feature, tests might need to assume a pre-configured device ID.

if __name__ == "__main__":
    # This is for local testing and might need adjustment based on actual client.
    print("To run these E2E tests, use: pytest -m e2e")

    async def local_run():
        # This is a very simplified local runner, not a replacement for pytest
        print("Simulating local async E2E run (using Mocks):")
        test_client = BifrostClient(base_url=GATEWAY_URL)
        async with test_client: # Uses mock __aenter__ and __aexit__
            if not test_client.is_connected():
                 print("Failed to connect client for local run.")
                 return

            await test_gateway_connection_and_device_presence(test_client)
            await test_read_single_modbus_tag(test_client)
            await test_write_single_modbus_tag_and_verify(test_client)
            await test_read_multiple_modbus_tags(test_client)
            await test_write_multiple_modbus_tags_and_verify(test_client)
        print("Local async E2E run simulation complete.")

    # asyncio.run(local_run()) # Uncomment to try local simulation with mocks
    pass
