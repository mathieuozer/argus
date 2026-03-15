"""Argus SDK for Python - Agent instrumentation and telemetry."""

from argus.client import ArgusClient
from argus.decorators import trace

__version__ = "0.1.0"
__all__ = ["ArgusClient", "trace"]
