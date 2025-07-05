import os
import warnings
from typing import Any

class FeatureFlags:
    """
    Manages feature flags for the Bifrost application.

    Feature flags are primarily configured via environment variables.
    Format: BIFROST_FF_<FLAG_NAME_UPPERCASE>=<true|false|1|0>
    Example: BIFROST_FF_NEW_MODBUS_OPTIMIZATION=true

    The system supports pre-registering flags with default values and descriptions.
    Environment variables will always override registered defaults.
    """

    _instance: Any = None # Using Any to satisfy mypy for singleton pattern until Python 3.13 can use Self
    _initialized: bool = False
    _flags: dict[str, bool]
    _registered_flags: dict[str, tuple[bool, str | None]] # name -> (default_state, description)
    _env_vars_loaded: set[str] # Tracks flags loaded from environment variables

    def __new__(cls) -> "FeatureFlags":
        if cls._instance is None:
            cls._instance = super().__new__(cls)
            cls._instance._flags = {}
            cls._instance._registered_flags = {}
            cls._instance._env_vars_loaded = set()
        return cls._instance

    def _initialize(self) -> None:
        if self._initialized:
            return

        # 1. Load registered defaults into the primary _flags dictionary
        for name, (default_val, _) in self._registered_flags.items():
            self._flags[name] = default_val

        # 2. Load from environment variables, overriding registered defaults
        prefix = "BIFROST_FF_"
        for env_var_name, value_str in os.environ.items():
            if env_var_name.startswith(prefix):
                flag_name_from_env = env_var_name[len(prefix):]
                # Normalize to lowercase for internal storage and matching
                normalized_flag_name = flag_name_from_env.lower()

                parsed_value: bool | None = None
                if value_str.lower() in ("true", "1"):
                    parsed_value = True
                elif value_str.lower() in ("false", "0"):
                    parsed_value = False
                else:
                    warnings.warn(
                        f"Invalid value '{value_str}' for feature flag environment variable {env_var_name}. "
                        "Expected 'true', 'false', '1', or '0'. Flag will be ignored or use default.",
                        RuntimeWarning
                    )

                if parsed_value is not None:
                    self._flags[normalized_flag_name] = parsed_value
                    self._env_vars_loaded.add(normalized_flag_name)

        self._initialized = True

    def register_flag(
        self, name: str, default_state: bool, description: str | None = None
    ) -> None:
        """
        Registers a known feature flag with its default state and description.

        This method allows different parts of Bifrost (e.g., extension packages)
        to declare the feature flags they use. It's recommended to call this
        at module import time.

        If a flag is already registered, this method will not update it unless
        the feature flag system has not been initialized yet. If already initialized,
        environment variables would have taken precedence.

        Args:
            name: The name of the feature flag (e.g., "new_modbus_optimization").
                  Will be normalized to lowercase.
            default_state: The default boolean state of the flag.
            description: An optional human-readable description of the feature.
        """
        normalized_name = name.lower()

        if normalized_name not in self._registered_flags:
            self._registered_flags[normalized_name] = (default_state, description)
            # If already initialized and this flag wasn't set by an env var, apply its new default.
            if self._initialized and normalized_name not in self._env_vars_loaded:
                self._flags[normalized_name] = default_state
        elif not self._initialized:
            # Allow re-registration (e.g. override) if not yet initialized
             self._registered_flags[normalized_name] = (default_state, description)


    def is_enabled(self, flag_name: str, default_override: bool | None = None) -> bool:
        """
        Checks if a feature flag is enabled.

        The evaluation order is:
        1. Environment variable (if set).
        2. Value from pre-registration (if registered and not overridden by env var).
        3. `default_override` parameter (if provided).
        4. Global default of `False` if the flag is unknown.

        Args:
            flag_name: The name of the feature flag. Case-insensitive.
            default_override: If provided, this value is returned if the flag
                              is not explicitly configured via environment
                              variables or registration.

        Returns:
            True if the feature is enabled, False otherwise.
        """
        if not self._initialized:
            self._initialize()

        normalized_name = flag_name.lower()

        # Check if the flag has a value (either from env or registration)
        if normalized_name in self._flags:
            return self._flags[normalized_name]

        # If not found, use default_override if provided
        if default_override is not None:
            return default_override

        # If the flag was never registered and not set by env var, and no call-site default,
        # it's considered disabled.
        # A warning could be issued here for completely unknown flags if desired.
        # For now, unknown flags silently return False.
        # warnings.warn(f"Feature flag '{normalized_name}' accessed but not registered or configured.", RuntimeWarning)
        return False

    def get_all_flags(self) -> dict[str, tuple[bool, str | None]]:
        """
        Returns a dictionary of all known flag states and their descriptions.
        This includes flags loaded from environment variables and registered flags.

        Returns:
            A dictionary where keys are flag names and values are tuples
            of (current_bool_state, description_or_None).
        """
        if not self._initialized:
            self._initialize()

        all_flags_info = {}
        # Start with registered flags to get descriptions
        for name, (default_val, desc) in self._registered_flags.items():
            current_val = self._flags.get(name, default_val) # Get current state, may differ from registered default
            all_flags_info[name] = (current_val, desc)

        # Add any flags that were only set by env vars and not pre-registered
        for name, val in self._flags.items():
            if name not in all_flags_info:
                all_flags_info[name] = (val, "Set via environment variable, not pre-registered.")

        return all_flags_info

    def reset_for_testing(self) -> None:
        """
        Resets the FeatureFlags instance to a clean state.
        Primarily intended for use in testing scenarios.
        """
        self._initialized = False
        self._flags.clear()
        self._registered_flags.clear()
        self._env_vars_loaded.clear()
        # Note: This doesn't clear os.environ, tests need to manage that.

# Global instance to be used by applications/packages
features = FeatureFlags()
