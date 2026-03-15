"""Argus SDK for Python - Agent instrumentation and telemetry."""

from argus.client import ArgusClient, Span
from argus.decorators import init, trace, get_client, shutdown

__version__ = "0.1.0"
__all__ = ["ArgusClient", "Span", "init", "trace", "get_client", "shutdown"]
