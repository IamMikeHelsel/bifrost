# bifrost-core

Core abstractions and base classes for the Bifrost industrial IoT framework.

## Installation

```bash
uv add bifrost-core
```

## Usage

```python
from bifrost_core import BaseConnection, DataPoint

# Use base classes to implement custom protocols
class MyProtocol(BaseConnection):
    async def read_tag(self, address: str) -> DataPoint:
        # Implementation here
        pass
```

## What's Included

- **BaseConnection**: Abstract base class for all protocol implementations
- **BaseProtocol**: Interface for protocol-specific operations
- **DataPoint**: Unified data model for industrial data
- **EventBus**: Event system for connection lifecycle management
- **ConnectionPool**: Connection pooling and management
- **Type System**: Common type definitions and enums
- Feature Flags: A system for enabling or disabling features at runtime.

## Feature Flags

Bifrost Core includes a lightweight feature flag system that allows for toggling features primarily through environment variables.

### Checking Feature Flags

To check if a feature is enabled:

```python
from bifrost_core import features

if features.is_enabled("my_new_feature"):
    # Execute code for the new feature
    print("My New Feature is ON!")
else:
    # Fallback or old behavior
    print("My New Feature is OFF.")

# You can also provide a call-site default:
if features.is_enabled("another_feature", default_override=True):
    print("Another Feature is ON (or defaults to ON here)!")
```

### Configuring Feature Flags

Feature flags are configured via environment variables with the prefix `BIFROST_FF_`. The flag name is case-insensitive. Accepted values for enabling a flag are `true` or `1`. Accepted values for disabling a flag are `false` or `0`.

**Examples:**

```bash
export BIFROST_FF_MY_NEW_FEATURE=true
export BIFROST_FF_EXPERIMENTAL_THING=1
export BIFROST_FF_OLD_FEATURE_TO_DISABLE=false
```

If an environment variable for a flag is not set, or if it's set to an invalid value, the flag will use its registered default value, or `False` if it was never registered.

### Registering Feature Flags (for developers of Bifrost packages)

Packages extending Bifrost can register the feature flags they use. This allows them to define a default state and a description. Registration is typically done at the module level.

```python
from bifrost_core import features

# Register a flag with its default state and description
features.register_flag(
    name="my_package_specific_feature",
    default_state=False,
    description="Enables a specific optimization in My Package."
)
```

- Environment variables always take precedence over registered defaults.
- Flag names are treated as case-insensitive (e.g., `BIFROST_FF_MYFEATURE` will affect a flag registered as `MyFeature`).
- Invalid values for feature flag environment variables will cause a `RuntimeWarning` and the flag will revert to its default or `False`.

### Inspecting All Flags

You can get a snapshot of all known flags and their current states:

```python
all_flags_status = features.get_all_flags()
for flag_name, (is_active, description) in all_flags_status.items():
    print(f"Flag: {flag_name}, Active: {is_active}, Description: {description}")
```

## Dependencies

- Python 3.13+
- Pydantic 2.5+ (data validation)
- typing-extensions 4.8+ (enhanced type hints)

## License

MIT License - see LICENSE file for details.
