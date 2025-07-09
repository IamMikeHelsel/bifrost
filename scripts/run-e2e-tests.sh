#!/bin/bash

# Exit immediately if a command exits with a non-zero status.
set -e

# --- Configuration ---
VIRTUAL_DEVICES_COMPOSE_FILE="virtual-devices/docker-compose.yml"
GO_GATEWAY_DIR="go-gateway"
PYTHON_PACKAGES_DIR="packages/bifrost" # Assuming E2E tests will be here or subdirs
GATEWAY_BINARY_PATH="${GO_GATEWAY_DIR}/bin/bifrost-gateway" # Expected location after build
GATEWAY_LOG_FILE="bifrost-gateway-e2e.log"
TEST_RESULTS_DIR="test-results/e2e"

# --- Helper Functions ---
info() {
    echo "[INFO] $1"
}

error() {
    echo "[ERROR] $1" >&2
    exit 1
}

cleanup() {
    info "Cleaning up..."
    info "Stopping virtual devices..."
    docker-compose -f "${VIRTUAL_DEVICES_COMPOSE_FILE}" down -v --remove-orphans || info "Failed to stop virtual devices, they might not have been running."

    if [ -n "$GATEWAY_PID" ] && ps -p $GATEWAY_PID > /dev/null; then
        info "Stopping Go gateway (PID: $GATEWAY_PID)..."
        kill $GATEWAY_PID
        wait $GATEWAY_PID || info "Go gateway already stopped or could not be killed."
    fi
    info "Cleanup complete."
}

# Trap ERR and EXIT signals to ensure cleanup happens.
trap cleanup ERR EXIT

# --- Main Script ---

info "Starting E2E Test Orchestration..."

# 1. Build Components (if necessary)
info "Building Go gateway..."
(cd "${GO_GATEWAY_DIR}" && just build) || error "Failed to build Go gateway."
if [ ! -f "${GATEWAY_BINARY_PATH}" ]; then
    error "Gateway binary not found at ${GATEWAY_BINARY_PATH} after build."
fi

info "Ensuring Python test environment is set up..."
# Assuming 'just dev-setup' or similar at the root handles Python venv and deps
(just dev-setup) || error "Failed to set up Python environment."


# 2. Start Virtual Devices
info "Starting virtual devices using Docker Compose..."
docker-compose -f "${VIRTUAL_DEVICES_COMPOSE_FILE}" up -d --build --force-recreate
# Add a small delay to ensure devices are fully up
sleep 10 # Adjust as needed, or implement health checks for services

# Verify Modbus simulator is running (assuming it's named 'modbus-sim' in compose and exposes 502)
if ! docker ps | grep -q "modbus-sim"; then # TODO: Get actual service name from compose
    error "Modbus simulator container does not appear to be running."
fi
info "Virtual devices started."

# 3. Start Go Gateway
info "Starting Go gateway..."
# Assuming gateway configuration (e.g., gateway.yaml) is set up to connect to localhost:502 for Modbus
# Start gateway in background and redirect its output
nohup "${GATEWAY_BINARY_PATH}" > "${GATEWAY_LOG_FILE}" 2>&1 &
GATEWAY_PID=$!
sleep 5 # Give gateway time to start

# Check if gateway is running
if ! ps -p $GATEWAY_PID > /dev/null; then
    error "Go gateway failed to start. Check ${GATEWAY_LOG_FILE}."
fi
info "Go gateway started (PID: $GATEWAY_PID). Logs at ${GATEWAY_LOG_FILE}."

# Optional: Health check for the gateway
# curl -f http://localhost:8080/health || error "Gateway health check failed."


# 4. Run E2E Tests
info "Running Python E2E tests..."
mkdir -p "${TEST_RESULTS_DIR}"
# Placeholder for actual test command. This will be updated in a later step.
# For now, we'll just simulate a successful test run.
# (cd "${PYTHON_PACKAGES_DIR}" && uv run pytest tests/e2e --junitxml="${TEST_RESULTS_DIR}/results.xml") || TEST_FAILURE=1
info "Simulating E2E test run... (Actual tests to be added)"
sleep 2 # Simulate test execution time
TEST_FAILURE=0 # Simulate success

if [ "$TEST_FAILURE" -eq 1 ]; then
    error "E2E tests failed. See output above and ${TEST_RESULTS_DIR}/results.xml."
fi
info "E2E tests completed."


# 5. Cleanup is handled by the trap

info "E2E Test Orchestration Succeeded!"
exit 0
