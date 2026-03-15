"""Argus SDK client for Python agent instrumentation."""

import json
import threading
import time
import uuid
from typing import Any, Dict, List, Optional
from urllib.request import Request, urlopen
from urllib.error import URLError


class ArgusClient:
    """Client for the Argus platform with batched telemetry sending."""

    def __init__(
        self,
        endpoint: str = "http://localhost:8080",
        tenant_id: str = "",
        agent_id: str = "",
        batch_size: int = 100,
        flush_interval: float = 5.0,
    ):
        self.endpoint = endpoint
        self.tenant_id = tenant_id
        self.agent_id = agent_id
        self.batch_size = batch_size
        self.flush_interval = flush_interval

        self._lock = threading.Lock()
        self._spans: List[Dict[str, Any]] = []
        self._events: List[Dict[str, Any]] = []
        self._closed = False

        self._flush_timer: Optional[threading.Timer] = None
        self._start_flush_timer()

    def _start_flush_timer(self) -> None:
        if self._closed:
            return
        self._flush_timer = threading.Timer(self.flush_interval, self._auto_flush)
        self._flush_timer.daemon = True
        self._flush_timer.start()

    def _auto_flush(self) -> None:
        self.flush()
        self._start_flush_timer()

    def start_span(self, name: str, trace_id: Optional[str] = None) -> "Span":
        """Start a new traced span."""
        return Span(
            name=name,
            client=self,
            trace_id=trace_id or str(uuid.uuid4()),
        )

    def emit_event(
        self,
        event_type: str,
        payload: Optional[Dict[str, str]] = None,
    ) -> None:
        """Emit a business event."""
        with self._lock:
            self._events.append({
                "event_type": event_type,
                "payload": payload or {},
                "timestamp": time.time(),
            })
            if len(self._events) >= self.batch_size:
                self._flush_events()

    def _add_span(self, span_data: Dict[str, Any]) -> None:
        """Add a completed span to the batch."""
        with self._lock:
            self._spans.append(span_data)
            if len(self._spans) >= self.batch_size:
                self._flush_spans()

    def flush(self) -> None:
        """Flush all pending spans and events to the platform."""
        with self._lock:
            self._flush_spans()
            self._flush_events()

    def _flush_spans(self) -> None:
        """Flush pending spans (must hold lock)."""
        if not self._spans:
            return
        spans = self._spans[:]
        self._spans = []
        self._send("/api/v1/telemetry/spans", {"spans": spans})

    def _flush_events(self) -> None:
        """Flush pending events (must hold lock)."""
        if not self._events:
            return
        events = self._events[:]
        self._events = []
        self._send("/api/v1/telemetry/events", {"events": events})

    def _send(self, path: str, data: Dict[str, Any]) -> None:
        """Send data to the Argus platform."""
        data["tenant_id"] = self.tenant_id
        data["agent_id"] = self.agent_id

        body = json.dumps(data).encode("utf-8")
        req = Request(
            url=self.endpoint + path,
            data=body,
            headers={
                "Content-Type": "application/json",
                "X-Tenant-ID": self.tenant_id,
            },
            method="POST",
        )
        try:
            with urlopen(req, timeout=10) as resp:
                resp.read()
        except (URLError, OSError):
            pass  # Silently drop on network errors to not block the agent

    @property
    def pending_spans(self) -> int:
        """Return count of pending spans."""
        with self._lock:
            return len(self._spans)

    @property
    def pending_events(self) -> int:
        """Return count of pending events."""
        with self._lock:
            return len(self._events)

    def close(self) -> None:
        """Close the client and flush pending data."""
        if self._closed:
            return
        self._closed = True
        if self._flush_timer:
            self._flush_timer.cancel()
        self.flush()

    def __enter__(self) -> "ArgusClient":
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        self.close()


class Span:
    """A traced operation span with context manager support."""

    def __init__(
        self,
        name: str,
        client: ArgusClient,
        trace_id: str,
    ):
        self.name = name
        self.client = client
        self.trace_id = trace_id
        self.span_id = str(uuid.uuid4())
        self.started_at = time.time()
        self.attributes: Dict[str, str] = {}
        self.error_code: Optional[str] = None

    def set_attribute(self, key: str, value: str) -> None:
        """Set an attribute on the span."""
        self.attributes[key] = value

    def set_error(self, error: Exception) -> None:
        """Record an error on the span."""
        self.error_code = type(error).__name__
        self.attributes["error"] = str(error)

    def end(self) -> None:
        """End the span and queue it for sending."""
        duration_ms = int((time.time() - self.started_at) * 1000)
        self.client._add_span({
            "span_id": self.span_id,
            "trace_id": self.trace_id,
            "operation_name": self.name,
            "started_at": self.started_at,
            "duration_ms": duration_ms,
            "attributes": self.attributes,
            "error_code": self.error_code,
        })

    def child(self, name: str) -> "Span":
        """Create a child span with the same trace ID."""
        return Span(name=name, client=self.client, trace_id=self.trace_id)

    def __enter__(self) -> "Span":
        return self

    def __exit__(self, exc_type: Any, exc_val: Any, exc_tb: Any) -> None:
        if exc_type is not None and exc_val is not None:
            self.set_error(exc_val)
        self.end()
