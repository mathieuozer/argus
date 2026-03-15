"""Argus SDK decorators."""

import functools
from typing import Callable, Optional

from argus.client import ArgusClient


# Module-level default client
_default_client: Optional[ArgusClient] = None


def init(endpoint: str = "localhost:8080", tenant_id: str = "") -> None:
    """Initialize the default Argus client."""
    global _default_client
    _default_client = ArgusClient(endpoint=endpoint, tenant_id=tenant_id)


def trace(func: Optional[Callable] = None, *, name: Optional[str] = None):
    """Decorator to trace a function execution.

    Usage:
        @argus.trace
        def my_function():
            ...

        @argus.trace(name="custom_name")
        def my_function():
            ...
    """

    def decorator(fn: Callable) -> Callable:
        span_name = name or fn.__qualname__

        @functools.wraps(fn)
        def wrapper(*args, **kwargs):
            client = _default_client
            if client is None:
                return fn(*args, **kwargs)

            span = client.start_span(span_name)
            try:
                result = fn(*args, **kwargs)
                return result
            except Exception as e:
                span.set_attribute("error", str(type(e).__name__))
                raise
            finally:
                span.end()

        return wrapper

    if func is not None:
        return decorator(func)
    return decorator
