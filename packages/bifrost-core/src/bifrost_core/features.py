"""A system for discovering and managing features provided by Bifrost components."""

from typing import Any, Protocol

from .typing import Feature


class HasFeatures(Protocol):
    """An interface for objects that provide features."""

    @property
    def features(self) -> set[Feature]:
        """Return a set of features provided by the object."""
        ...


class FeatureRegistry:
    """A registry for discovering and managing features."""

    def __init__(self) -> None:
        self._providers: dict[Feature, list[Any]] = {}

    def register(self, provider: Any) -> None:
        """Register a feature provider."""
        if isinstance(provider, HasFeatures):
            for feature in provider.features:
                if feature not in self._providers:
                    self._providers[feature] = []
                self._providers[feature].append(provider)

    def discover(self, feature: Feature) -> list[Any]:
        """Discover all providers for a given feature."""
        return self._providers.get(feature, [])

    def first(self, feature: Feature) -> Any | None:
        """Return the first provider for a given feature."""
        providers = self.discover(feature)
        return providers[0] if providers else None
