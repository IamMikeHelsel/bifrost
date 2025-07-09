# End-to-End (E2E) Testing Strategy

This document outlines the End-to-End testing strategy for the Bifrost project. E2E tests are crucial for verifying that all components of the system (Go Gateway, Python client libraries, virtual devices, etc.) integrate and function correctly together.

## Overview

Our E2E tests simulate real-world scenarios where a client application (e.g., using the `bifrost` Python package) interacts with the Bifrost Go Gateway, which in turn communicates with industrial devices (simulated by our virtual device containers).

## Running E2E Tests Locally

The primary way to run the E2E test suite locally is using the orchestration script:

```bash
./scripts/run-e2e-tests.sh
```

This script handles:
1.  Building the Go Gateway.
2.  Setting up the Python test environment (via `just dev-setup`).
3.  Starting virtual devices (e.g., Modbus simulator) using Docker Compose from `virtual-devices/docker-compose.yml`.
4.  Starting the Go Gateway, configured to connect to these virtual devices.
5.  Executing the Python E2E test suite located in `packages/bifrost/tests/e2e/` using `pytest`.
6.  Collecting test results (e.g., JUnit XML).
7.  Stopping and cleaning up all started services (gateway and virtual devices).

Refer to the script itself for detailed steps and configuration.

## E2E Test Scenarios

The E2E tests are located in `packages/bifrost/tests/e2e/`. Each file typically covers a specific aspect or protocol.

### Current Modbus E2E Scenarios:

*   **`test_e2e_modbus_connect_read_write.py`**:
    *   Verifies connection to the gateway.
    *   Checks if the configured Modbus simulator device is accessible via the gateway.
    *   Tests reading single and multiple tags from the Modbus simulator.
    *   Tests writing to single and multiple writable tags on the simulator and verifies the values by reading them back.
*   **`test_e2e_modbus_error_scenarios.py`**:
    *   Tests gateway and client behavior when attempting to read from invalid Modbus addresses.
    *   Tests attempts to write to read-only Modbus addresses.
    *   Tests operations against device IDs not known to the gateway.
    *   (Placeholder) Tests communication with a temporarily disconnected Modbus device.

*(Note: The actual implementation of the Python `BifrostClient` calls within these tests might initially use mocks until the client library and gateway API are fully stabilized. The goal is to transition these to use the live client entirely.)*

## Adding New E2E Test Scenarios

To add new E2E test scenarios:

1.  **Identify the Scenario**: Clearly define what aspect of cross-component interaction you want to test.
2.  **Simulator Requirements**:
    *   If testing a new protocol, ensure a virtual device/simulator for it exists in `virtual-devices/` and is included in `virtual-devices/docker-compose.yml`.
    *   For existing protocols (like Modbus), ensure the simulator (`modbus_server.py`) exposes the necessary registers/tags with the behavior you need (e.g., specific read-only values, writable areas, error conditions). You might need to update the simulator.
3.  **Gateway Configuration**: Ensure the Go Gateway (when started by `run-e2e-tests.sh`) will be configured to communicate with the new or updated simulator settings. This might involve changes to a default gateway configuration file used for E2E tests if static configuration is employed.
4.  **Create Test File**:
    *   Add a new Python test file in `packages/bifrost/tests/e2e/`, typically named `test_e2e_<protocol>_<scenario>.py`.
    *   Use `pytest` and `asyncio` for asynchronous tests.
    *   Import the `BifrostClient` (or its equivalent) and any necessary types from `bifrost_core`.
5.  **Write Test Functions**:
    *   Define test functions (e.g., `async def test_my_new_scenario(client):`).
    *   Use the `client` fixture (or create a new one if special setup is needed) to interact with the gateway.
    *   Perform operations (reads, writes, configuration changes, etc.) that trigger the scenario.
    *   Make assertions about the results (e.g., correct data read, expected errors raised, proper system state).
6.  **Mark as E2E**: Add the `@pytest.mark.e2e` marker to your test functions or class.
7.  **Test Locally**: Run `./scripts/run-e2e-tests.sh` to ensure your new tests pass and don't break existing ones. You can also run specific files or tests using `pytest` directly if the environment is already up (e.g., `pytest packages/bifrost/tests/e2e/your_new_test_file.py`).
8.  **CI Integration**: The CI workflow (`.github/workflows/ci.yml`) is already configured to run all tests picked up by `run-e2e-tests.sh`. Ensure your tests are discovered correctly by `pytest`.

## Future Enhancements

*   E2E tests for other protocols (OPC UA, Ethernet/IP, etc.) as they are implemented.
*   Tests involving the VS Code extension interacting with the live gateway.
*   More complex scenarios: data persistence, performance under E2E load, high availability/failover.
*   Dynamic control over virtual device states (e.g., stopping/starting a specific simulator mid-test) from the `run-e2e-tests.sh` script or Python tests.

By following this structure, we can build a comprehensive suite of E2E tests that provide high confidence in the stability and correctness of the entire Bifrost system.
