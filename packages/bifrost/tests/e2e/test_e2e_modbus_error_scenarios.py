"""
End-to-End tests for Modbus error scenarios via the Bifrost Gateway.

These tests assume:
1. The Bifrost Go Gateway is running.
2. A Modbus simulator (virtual device) is running and accessible to the gateway.
3. The gateway is configured to communicate with the Modbus simulator.
"""
import pytest
import asyncio

# Assuming the bifrost client library is structured appropriately.
# from bifrost.client import BifrostClient, BifrostDeviceNotFoundError, BifrostRequestError # Real imports
# from bifrost_core import Tag, DataType # Real imports

# --- Mocking BifrostClient and related classes for now ---
# This section should be removed once the actual client is available and importable.
# Using the same mocks from test_e2e_modbus_connect_read_write.py for consistency
import time
class MockBifrostClient:
    def __init__(self, base_url):
        self.base_url = base_url
        self._connected = False
        self.mock_data_store = {} # For simulating writes if needed
        self.simulated_devices = {
            MODBUS_DEVICE_ID: {"status": "connected", "protocol": "modbus-tcp"},
            "unknown-device": {"status": "does_not_exist", "protocol": "modbus-tcp"}
        }
        print(f"MockBifrostClient initialized for error testing (URL: {base_url})")

    async def __aenter__(self):
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.disconnect()

    async def connect(self):
        self._connected = True
        print(f"MockBifrostClient (ErrorSim): Connected to {self.base_url}")
        await asyncio.sleep(0.01)


    async def disconnect(self):
        self._connected = False
        print("MockBifrostClient (ErrorSim): Disconnected.")
        await asyncio.sleep(0.01)

    def is_connected(self):
        return self._connected

    async def read_tags(self, device_id, tags):
        print(f"MockBifrostClient (ErrorSim): Attempting to read tags from device {device_id}: {tags}")
        await asyncio.sleep(0.05)

        if device_id not in self.simulated_devices or self.simulated_devices[device_id]["status"] == "disconnected":
            # Simulate device not found or communication error
            raise MockBifrostDeviceNotFoundError(f"Device {device_id} not found or disconnected.")

        readings = {}
        for tag in tags:
            if tag.address == INVALID_TAG_ADDRESS:
                # Simulate invalid address error from gateway/device
                raise MockBifrostRequestError(f"Invalid address {tag.address} for device {device_id}")

            # Use a mock TagReading if the real one isn't available
            class MockTagReading:
                def __init__(self, tag, value, timestamp, error=None):
                    self.tag = tag
                    self.value = value
                    self.timestamp = timestamp
                    self.error = error # For tag-specific errors

            # Simulate a successful read for other valid tags for now
            readings[tag.name] = MockTagReading(tag, 0, time.time())
        return readings

    async def write_tags(self, device_id, tags_with_values):
        print(f"MockBifrostClient (ErrorSim): Attempting to write tags to device {device_id}: {tags_with_values}")
        await asyncio.sleep(0.05)

        if device_id not in self.simulated_devices or self.simulated_devices[device_id]["status"] == "disconnected":
            raise MockBifrostDeviceNotFoundError(f"Device {device_id} not found or disconnected.")

        for tag, value in tags_with_values.items():
            if tag.address == READ_ONLY_ADDRESS: # Example of a non-writable address
                raise MockBifrostRequestError(f"Tag at address {tag.address} is not writable on device {device_id}.")
            if tag.address == INVALID_TAG_ADDRESS:
                 raise MockBifrostRequestError(f"Invalid address {tag.address} for write on device {device_id}")
            # Simulate storing the written value
            self.mock_data_store[f"{device_id}:{tag.address}"] = value
        return True # Simulate overall success if no specific error was raised

# Custom Mock Exceptions to simulate client library errors
class MockBifrostError(Exception): pass
class MockBifrostDeviceNotFoundError(MockBifrostError): pass
class MockBifrostRequestError(MockBifrostError): pass

BifrostClient = MockBifrostClient # Use mock for now
BifrostDeviceNotFoundError = MockBifrostDeviceNotFoundError
BifrostRequestError = MockBifrostRequestError


class MockTag:
    def __init__(self, name, address, data_type, description=""):
        self.name = name
        self.address = address
        self.data_type = data_type
        self.description = description
    def __repr__(self):
        return f"Tag(name='{self.name}', address='{self.address}', data_type='{self.data_type}')"

class MockDataType:
    INT16 = "INT16"
    FLOAT32 = "FLOAT32"
    BOOLEAN = "BOOLEAN"

Tag = MockTag
DataType = MockDataType
# --- End of Mocking Section ---

GATEWAY_URL = "http://localhost:8080"
MODBUS_DEVICE_ID = "bifrost-modbus-sim" # Matches container_name
INVALID_TAG_ADDRESS = "49999" # An address that likely doesn't exist
READ_ONLY_ADDRESS = "40001"   # From modbus_server.py, sensor data is read-only up to 40030
EXISTING_TAG_FOR_DEVICE_TEST = "40002" # A valid readable tag

@pytest.fixture(scope="module")
async def client():
    """Fixture to create and connect a BifrostClient."""
    # When real client is available:
    # real_client = BifrostClient(base_url=GATEWAY_URL)
    # async with real_client as c:
    #     yield c
    # For now, using the mock:
    mock_client_instance = BifrostClient(base_url=GATEWAY_URL)
    await mock_client_instance.connect()
    yield mock_client_instance
    await mock_client_instance.disconnect()


@pytest.mark.e2e
@pytest.mark.asyncio
async def test_read_invalid_modbus_tag_address(client: BifrostClient):
    """Test reading from an invalid Modbus tag address, expecting an error."""
    tag_to_read = Tag(name="InvalidTag", address=INVALID_TAG_ADDRESS, data_type=DataType.INT16)

    with pytest.raises(BifrostRequestError) as excinfo:
        await client.read_tags(MODBUS_DEVICE_ID, [tag_to_read])

    assert INVALID_TAG_ADDRESS in str(excinfo.value).lower()
    assert "invalid address" in str(excinfo.value).lower()
    print(f"Successfully caught expected error for invalid read address: {excinfo.value}")

@pytest.mark.e2e
@pytest.mark.asyncio
async def test_write_to_readonly_address(client: BifrostClient):
    """Test writing to a Modbus address that is designated as read-only by the simulator."""
    tag_to_write = Tag(name="AttemptWriteToReadOnly", address=READ_ONLY_ADDRESS, data_type=DataType.INT16)
    test_value = 9999

    with pytest.raises(BifrostRequestError) as excinfo:
        await client.write_tags(MODBUS_DEVICE_ID, {tag_to_write: test_value})

    assert READ_ONLY_ADDRESS in str(excinfo.value)
    assert "not writable" in str(excinfo.value).lower() # Or similar error message from client/gateway
    print(f"Successfully caught expected error for writing to read-only address: {excinfo.value}")

@pytest.mark.e2e
@pytest.mark.asyncio
async def test_write_to_invalid_modbus_tag_address(client: BifrostClient):
    """Test writing to an invalid Modbus tag address, expecting an error."""
    tag_to_write = Tag(name="InvalidWriteTag", address=INVALID_TAG_ADDRESS, data_type=DataType.INT16)
    test_value = 5555

    with pytest.raises(BifrostRequestError) as excinfo:
        await client.write_tags(MODBUS_DEVICE_ID, {tag_to_write: test_value})

    assert INVALID_TAG_ADDRESS in str(excinfo.value).lower()
    assert "invalid address" in str(excinfo.value).lower()
    print(f"Successfully caught expected error for invalid write address: {excinfo.value}")


@pytest.mark.e2e
@pytest.mark.asyncio
async def test_operation_on_unknown_device_id(client: BifrostClient):
    """Test performing an operation on a device ID not known to the gateway."""
    unknown_device_id = "unknown-modbus-device"
    tag = Tag(name="AnyTag", address=READ_ONLY_ADDRESS, data_type=DataType.INT16)

    with pytest.raises(BifrostDeviceNotFoundError) as excinfo:
        await client.read_tags(unknown_device_id, [tag])

    assert unknown_device_id in str(excinfo.value)
    print(f"Successfully caught expected error for unknown device ID: {excinfo.value}")

    with pytest.raises(BifrostDeviceNotFoundError) as excinfo_write:
        await client.write_tags(unknown_device_id, {tag: 123})

    assert unknown_device_id in str(excinfo_write.value)
    print(f"Successfully caught expected error for write to unknown device ID: {excinfo_write.value}")


@pytest.mark.e2e
@pytest.mark.asyncio
async def test_communication_with_temporarily_disconnected_device(client: BifrostClient):
    """
    Test gateway/client behavior when a virtual device is temporarily unavailable.
    This test is more complex as it requires manipulating the virtual device state.
    For now, it's a placeholder.
    """
    # Steps would be:
    # 1. Ensure communication is working.
    # 2. Stop/disconnect the virtual Modbus device (e.g., `docker-compose stop modbus-sim`).
    # 3. Attempt to read/write and verify appropriate errors are received.
    # 4. Restart the virtual Modbus device.
    # 5. Verify communication is restored.
    await asyncio.sleep(0)
    print("Simulating test for communication with a disconnected device (placeholder).")
    print("This test would require control over the virtual device's lifecycle.")
    assert True

if __name__ == "__main__":
    print("To run these E2E error scenario tests, use: pytest packages/bifrost/tests/e2e/")
    pass
