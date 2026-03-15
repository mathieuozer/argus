"""Argus SDK decorators for automatic function instrumentation."""

import functools
import inspect
from typing import Any, Callable, Optional

from argus.client import ArgusClient


# Module-level default client
_default_client: Optional[ArgusClient] = None


def init(
    endpoint: str = "http://localhost:8080",
    tenant_id: str = "",
    agent_id: str = "",
    flush_interval: float = 5.0,
) -> ArgusClient:
    """Initialize the default Argus client.

    Returns the client for explicit use if needed.
    """
    global _default_client
    _default_client = ArgusClient(
        endpoint=endpoint,
        tenant_id=tenant_id,
        agent_id=agent_id,
        flush_interval=flush_interval,
    )
    return _default_client


def get_client() -> Optional[ArgusClient]:
    """Return the default client, if initialized."""
    return _default_client


def shutdown() -> None:
    """Close the default client."""
    global _default_client
    if _default_client is not None:
        _default_client.close()
        _default_client = None


def trace(func: Optional[Callable] = None, *, name: Optional[str] = None) -> Any:
    """Decorator to trace a function execution.

    Supports both sync and async functions.

    Usage:
        @argus.trace
        def my_function():
            ...

        @argus.trace(name="custom_name")
        def my_function():
            ...

        @argus.trace
        async def my_async_function():
            ...
    """

    def decorator(fn: Callable) -> Callable:
        span_name = name or fn.__qualname__

        if inspect.iscoroutinefunction(fn):
            @functools.wraps(fn)
            async def async_wrapper(*args: Any, **kwargs: Any) -> Any:
                client = _default_client
                if client is None:
                    return await fn(*args, **kwargs)

                span = client.start_span(span_name)
                try:
                    result = await fn(*args, **kwargs)
                    return result
                except Exception as e:
                    span.set_error(e)
                    raise
                finally:
                    span.end()

            return async_wrapper
        else:
            @functools.wraps(fn)
            def wrapper(*args: Any, **kwargs: Any) -> Any:
                client = _default_client
                if client is None:
                    return fn(*args, **kwargs)

                span = client.start_span(span_name)
                try:
                    result = fn(*args, **kwargs)
                    return result
                except Exception as e:
                    span.set_error(e)
                    raise
                finally:
                    span.end()

            return wrapper

    if func is not None:
        return decorator(func)
    return decorator
