"""Argus SDK client."""

import time
from typing import Dict, Optional


class ArgusClient:
    """Client for the Argus platform."""

    def __init__(
        self,
        endpoint: str = "localhost:8080",
        tenant_id: str = "",
    ):
        self.endpoint = endpoint
        self.tenant_id = tenant_id

    def start_span(self, name: str) -> "Span":
        """Start a new traced span."""
        return Span(name=name, client=self)

    def emit_event(
        self,
        event_type: str,
        payload: Optional[Dict[str, str]] = None,
    ) -> None:
        """Emit a business event."""
        # Stub: in production, send event to Argus platform
        pass

    def close(self) -> None:
        """Close the client and flush pending data."""
        pass


class Span:
    """A traced operation span."""

    def __init__(self, name: str, client: ArgusClient):
        self.name = name
        self.client = client
        self.started_at = time.time()
        self.attributes: Dict[str, str] = {}

    def set_attribute(self, key: str, value: str) -> None:
        """Set an attribute on the span."""
        self.attributes[key] = value

    def end(self) -> None:
        """End the span and send it to the platform."""
        # Stub: in production, send span to Argus platform
        pass

    def __enter__(self) -> "Span":
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        self.end()
