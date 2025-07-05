import os
import pytest
import warnings

from bifrost_core.features import FeatureFlags

# Helper to manage environment variables for tests
@pytest.fixture(autouse=True)
def manage_env_vars(monkeypatch):
    """
    Ensures environment variables set during a test are cleaned up.
    Also provides a way to set/unset env vars for the test duration.
    """
    original_environ = os.environ.copy()

    def _set_env_var(name, value):
        monkeypatch.setenv(name, value)

    def _del_env_var(name):
        monkeypatch.delenv(name, raising=False)

    yield _set_env_var, _del_env_var

    # Restore original environment (though monkeypatch should handle most of this)
    os.environ.clear()
    os.environ.update(original_environ)


@pytest.fixture
def ff_instance():
    """
    Provides a clean FeatureFlags instance for each test.
    """
    # The 'features' object is a global singleton. We need to reset it.
    from bifrost_core import features
    features.reset_for_testing()
    return features


class TestFeatureFlags:

    def test_env_var_enables_flag(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_TEST_FEATURE_1", "true")
        assert ff_instance.is_enabled("TEST_FEATURE_1") is True
        assert ff_instance.is_enabled("test_feature_1") is True # Case-insensitivity

    def test_env_var_disables_flag(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_TEST_FEATURE_2", "false")
        assert ff_instance.is_enabled("TEST_FEATURE_2") is False

    @pytest.mark.parametrize("true_val", ["true", "True", "1"])
    def test_env_var_true_values(self, ff_instance: FeatureFlags, manage_env_vars, true_val):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_TRUTHY", true_val)
        assert ff_instance.is_enabled("TRUTHY") is True

    @pytest.mark.parametrize("false_val", ["false", "False", "0"])
    def test_env_var_false_values(self, ff_instance: FeatureFlags, manage_env_vars, false_val):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_FALSY", false_val)
        assert ff_instance.is_enabled("FALSY") is False

    def test_invalid_env_var_value_uses_default(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        ff_instance.register_flag("INVALID_VAL_FEATURE", default_state=True, description="Test")
        set_env("BIFROST_FF_INVALID_VAL_FEATURE", "not_a_boolean")

        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always")
            # Default is True, env var is invalid, so should remain True
            assert ff_instance.is_enabled("INVALID_VAL_FEATURE") is True
            # Check if the specific warning was issued
            assert len(w) == 1
            assert issubclass(w[-1].category, RuntimeWarning)
            assert "Invalid value 'not_a_boolean' for feature flag" in str(w[-1].message)

    def test_unconfigured_flag_is_false(self, ff_instance: FeatureFlags):
        assert ff_instance.is_enabled("NON_EXISTENT_FLAG") is False

    def test_call_site_default_override_for_unconfigured_flag(self, ff_instance: FeatureFlags):
        assert ff_instance.is_enabled("NON_EXISTENT_FLAG_2", default_override=True) is True
        assert ff_instance.is_enabled("NON_EXISTENT_FLAG_3", default_override=False) is False

    def test_call_site_default_does_not_override_configured_flag(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_CONFIGURED_FLAG", "true")
        assert ff_instance.is_enabled("CONFIGURED_FLAG", default_override=False) is True

        set_env("BIFROST_FF_CONFIGURED_FLAG_2", "false")
        assert ff_instance.is_enabled("CONFIGURED_FLAG_2", default_override=True) is False

    def test_register_flag_default_true(self, ff_instance: FeatureFlags):
        ff_instance.register_flag("REGISTERED_TRUE", default_state=True, description="Desc")
        assert ff_instance.is_enabled("REGISTERED_TRUE") is True

    def test_register_flag_default_false(self, ff_instance: FeatureFlags):
        ff_instance.register_flag("REGISTERED_FALSE", default_state=False, description="Desc")
        assert ff_instance.is_enabled("REGISTERED_FALSE") is False

    def test_env_var_overrides_registered_default_true(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        ff_instance.register_flag("OVERRIDE_ME_TRUE", default_state=True, description="Initial true")
        set_env("BIFROST_FF_OVERRIDE_ME_TRUE", "false") # Env var sets to false
        assert ff_instance.is_enabled("OVERRIDE_ME_TRUE") is False

    def test_env_var_overrides_registered_default_false(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        ff_instance.register_flag("OVERRIDE_ME_FALSE", default_state=False, description="Initial false")
        set_env("BIFROST_FF_OVERRIDE_ME_FALSE", "true") # Env var sets to true
        assert ff_instance.is_enabled("OVERRIDE_ME_FALSE") is True

    def test_register_flag_after_init_and_env_var(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        # Env var sets it to true
        set_env("BIFROST_FF_REG_AFTER_ENV", "true")
        # Initialize by checking a flag
        assert ff_instance.is_enabled("ANY_FLAG_TO_INIT") is False

        # Now register with default false. Env var should still take precedence.
        ff_instance.register_flag("REG_AFTER_ENV", default_state=False, description="Test")
        assert ff_instance.is_enabled("REG_AFTER_ENV") is True

    def test_register_flag_after_init_no_env_var(self, ff_instance: FeatureFlags):
         # Initialize by checking a flag
        assert ff_instance.is_enabled("ANY_FLAG_TO_INIT_2") is False

        # Register. Since no env var, this default should apply.
        ff_instance.register_flag("REG_AFTER_NO_ENV", default_state=True, description="Test")
        assert ff_instance.is_enabled("REG_AFTER_NO_ENV") is True

    def test_get_all_flags(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars

        ff_instance.register_flag("flag1", default_state=True, description="Description 1")
        ff_instance.register_flag("flag2", default_state=False, description="Description 2")
        set_env("BIFROST_FF_FLAG2", "true") # Override flag2
        set_env("BIFROST_FF_FLAG3", "true") # Env-only flag

        # Initialize
        ff_instance.is_enabled("flag1")

        all_flags = ff_instance.get_all_flags()

        assert len(all_flags) == 3
        assert all_flags["flag1"] == (True, "Description 1")
        assert all_flags["flag2"] == (True, "Description 2") # Current state is true due to env var
        assert all_flags["flag3"] == (True, "Set via environment variable, not pre-registered.")

    def test_reset_for_testing_works(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars

        set_env("BIFROST_FF_RESET_TEST", "true")
        ff_instance.register_flag("RESET_REG_TEST", default_state=True)

        assert ff_instance.is_enabled("RESET_TEST") is True
        assert ff_instance.is_enabled("RESET_REG_TEST") is True

        ff_instance.reset_for_testing()

        # After reset, env var should be gone from internal state (though still in os.environ for this test's scope)
        # and registered flags should be gone.
        # To properly test reset's effect on env var loading, we'd need to clear the env var *then* check.
        # The main goal is that internal state is cleared.

        assert "reset_test" not in ff_instance._flags
        assert "reset_reg_test" not in ff_instance._registered_flags
        assert ff_instance.is_enabled("RESET_TEST", default_override=False) is False # False because env var not re-read yet
        assert ff_instance.is_enabled("RESET_REG_TEST") is False # False because registration is gone

        # If we set an env var again and check, it should pick up the new state
        set_env("BIFROST_FF_RESET_TEST_AFTER", "true")
        assert ff_instance.is_enabled("RESET_TEST_AFTER") is True


    def test_flag_name_normalization(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars

        set_env("BIFROST_FF_MY_UPPER_FLAG", "true")
        ff_instance.register_flag("My_Mixed_Case_Flag", default_state=False)
        set_env("BIFROST_FF_my_lower_env_flag", "true")
        ff_instance.register_flag("MY_ENV_OVERRIDDEN_REG_FLAG", default_state=False)
        set_env("BIFROST_FF_MY_ENV_OVERRIDDEN_REG_FLAG", "true")


        assert ff_instance.is_enabled("my_upper_flag") is True
        assert ff_instance.is_enabled("MY_UPPER_FLAG") is True

        assert ff_instance.is_enabled("my_mixed_case_flag") is False # Default is False, no env var
        assert ff_instance.is_enabled("MY_MIXED_CASE_FLAG") is False

        assert ff_instance.is_enabled("my_lower_env_flag") is True

        assert ff_instance.is_enabled("my_env_overridden_reg_flag") is True # Env var overrides registration

    def test_invalid_env_var_value_for_unregistered_flag(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_INVALID_UNREG", "not_bool")

        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always")
            # Should be false as it's not registered and env var is invalid
            assert ff_instance.is_enabled("INVALID_UNREG") is False
            assert len(w) == 1
            assert "Invalid value 'not_bool' for feature flag" in str(w[-1].message)

    def test_reregister_flag_before_init(self, ff_instance: FeatureFlags):
        ff_instance.register_flag("rereg_test", default_state=False, description="Initial")
        ff_instance.register_flag("rereg_test", default_state=True, description="Updated") # Reregister before init

        # Initialization happens on first call to is_enabled
        assert ff_instance.is_enabled("rereg_test") is True

        flags_info = ff_instance.get_all_flags()
        assert flags_info["rereg_test"] == (True, "Updated")

    def test_reregister_flag_after_init_no_env(self, ff_instance: FeatureFlags):
        ff_instance.register_flag("rereg_after_init", default_state=False, description="Initial")
        # Initialize
        assert ff_instance.is_enabled("rereg_after_init") is False

        # Attempt to re-register after init. Current design: this won't change the live default if no env var.
        # The original registration's default is sticky unless an env var overrides it.
        # Or, if the flag was NOT set by env, then a new registration *could* apply its default.
        # The current implementation of register_flag says:
        # "If already initialized and this flag wasn't set by an env var, apply its new default."

        ff_instance.register_flag("rereg_after_init", default_state=True, description="Updated")
        assert ff_instance.is_enabled("rereg_after_init") is True # It SHOULD update as it wasn't set by env

        flags_info = ff_instance.get_all_flags()
        assert flags_info["rereg_after_init"] == (True, "Updated")

    def test_reregister_flag_after_init_with_env(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_REREG_AFTER_INIT_ENV", "true") # Env var sets to true

        ff_instance.register_flag("rereg_after_init_env", default_state=False, description="Initial")
        # Initialize
        assert ff_instance.is_enabled("rereg_after_init_env") is True # Env var takes precedence

        # Attempt to re-register. Env var should still hold.
        ff_instance.register_flag("rereg_after_init_env", default_state=False, description="Updated but ignored")
        assert ff_instance.is_enabled("rereg_after_init_env") is True

        flags_info = ff_instance.get_all_flags()
        # Description should be from the latest registration, value from env var.
        assert flags_info["rereg_after_init_env"] == (True, "Updated but ignored")

    def test_empty_env_var_value(self, ff_instance: FeatureFlags, manage_env_vars):
        set_env, _ = manage_env_vars
        set_env("BIFROST_FF_EMPTY_VAL", "")
        ff_instance.register_flag("EMPTY_VAL", default_state=True)

        with warnings.catch_warnings(record=True) as w:
            warnings.simplefilter("always")
            # Invalid value, should use registered default
            assert ff_instance.is_enabled("EMPTY_VAL") is True
            assert len(w) == 1
            assert "Invalid value '' for feature flag" in str(w[-1].message)
