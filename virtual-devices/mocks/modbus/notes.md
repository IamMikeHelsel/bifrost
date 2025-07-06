# Modbus Mocks

## Purpose
Lightweight Modbus protocol mocks for fast unit testing.

## Planned Contents
- **In-memory Modbus Server**: No network dependencies
- **Configurable Responses**: Predictable data for test scenarios
- **Error Injection**: Exception codes and communication failures
- **State Tracking**: Monitor function calls and parameters

## Key Features
- Zero network latency (in-memory)
- Configurable register maps
- Exception simulation
- Call history tracking
- Deterministic behavior

## Usage Patterns
- Unit testing Modbus client code
- Error condition testing
- Performance baseline testing
- CI/CD pipeline integration

## Test Focus
- Function code handling
- Exception processing
- Data type conversion
- Connection state management