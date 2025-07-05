"""Tests for bifrost-core feature system."""

import pytest
from bifrost_core.features import Feature, FeatureRegistry, HasFeatures
from typing import Any, Set, Protocol, runtime_checkable


@runtime_checkable
class MockFeatureProvider(Protocol):
    """A mock class that provides features for testing."""

    @property
    def features(self) -> Set[Feature]:
        ...

    def do_something(self) -> str:
        ...


class ConcreteFeatureProvider:
    def __init__(self, name: str, provided_features: Set[Feature]):
        self._name = name
        self._features = provided_features

    @property
    def features(self) -> Set[Feature]:
        return self._features

    def do_something(self) -> str:
        return f"Doing something from {self._name}"


class TestFeatureRegistry:
    """Tests for the FeatureRegistry class."""

    @pytest.fixture
    def registry(self) -> FeatureRegistry:
        return FeatureRegistry()

    def test_register_and_discover_single_feature(self, registry: FeatureRegistry):
        provider = ConcreteFeatureProvider("ProviderA", {"feature_x"})
        registry.register(provider)

        discovered = registry.discover("feature_x")
        assert len(discovered) == 1
        assert discovered[0] is provider

        assert registry.first("feature_x") is provider

    def test_register_and_discover_multiple_features(self, registry: FeatureRegistry):
        provider1 = ConcreteFeatureProvider("Provider1", {"feature_a", "feature_b"})
        provider2 = ConcreteFeatureProvider("Provider2", {"feature_b", "feature_c"})

        registry.register(provider1)
        registry.register(provider2)

        discovered_a = registry.discover("feature_a")
        assert len(discovered_a) == 1
        assert discovered_a[0] is provider1

        discovered_b = registry.discover("feature_b")
        assert len(discovered_b) == 2
        assert provider1 in discovered_b
        assert provider2 in discovered_b

        discovered_c = registry.discover("feature_c")
        assert len(discovered_c) == 1
        assert discovered_c[0] is provider2

    def test_discover_non_existent_feature(self, registry: FeatureRegistry):
        discovered = registry.discover("non_existent_feature")
        assert len(discovered) == 0
        assert registry.first("non_existent_feature") is None

    def test_register_non_feature_provider(self, registry: FeatureRegistry):
        class NonProvider:
            pass

        non_provider = NonProvider()
        registry.register(non_provider)

        # Should not raise an error, and should not register any features
        assert registry.discover("any_feature") == []

    def test_first_returns_none_if_no_providers(self, registry: FeatureRegistry):
        assert registry.first("some_feature") is None

    def test_first_returns_first_registered(self, registry: FeatureRegistry):
        provider1 = ConcreteFeatureProvider("Provider1", {"common_feature"})
        provider2 = ConcreteFeatureProvider("Provider2", {"common_feature"})

        registry.register(provider1)
        registry.register(provider2)

        # Order of registration matters for 'first'
        assert registry.first("common_feature") is provider1

    def test_has_features_protocol(self):
        provider = ConcreteFeatureProvider("Test", {"feat"})
        assert isinstance(provider, HasFeatures)

        class NotAFeatureProvider:
            pass

        not_provider = NotAFeatureProvider()
        assert not isinstance(not_provider, HasFeatures)