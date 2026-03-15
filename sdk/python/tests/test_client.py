"""Tests for the Argus Python SDK client."""

import json
import time
import unittest
from http.server import HTTPServer, BaseHTTPRequestHandler
from threading import Thread
from unittest.mock import patch

from argus.client import ArgusClient, Span


class TestArgusClient(unittest.TestCase):
    """Tests for ArgusClient."""

    def test_default_init(self):
        client = ArgusClient()
        self.assertEqual(client.endpoint, "http://localhost:8080")
        self.assertEqual(client.tenant_id, "")
        self.assertEqual(client.agent_id, "")
        self.assertEqual(client.batch_size, 100)
        client.close()

    def test_custom_init(self):
        client = ArgusClient(
            endpoint="http://custom:9090",
            tenant_id="t1",
            agent_id="a1",
            batch_size=50,
            flush_interval=10.0,
        )
        self.assertEqual(client.endpoint, "http://custom:9090")
        self.assertEqual(client.tenant_id, "t1")
        self.assertEqual(client.agent_id, "a1")
        self.assertEqual(client.batch_size, 50)
        client.close()

    def test_start_span(self):
        client = ArgusClient()
        span = client.start_span("test-op")
        self.assertEqual(span.name, "test-op")
        self.assertIsNotNone(span.span_id)
        self.assertIsNotNone(span.trace_id)
        self.assertIsInstance(span.started_at, float)
        client.close()

    def test_emit_event(self):
        client = ArgusClient(flush_interval=999)
        client.emit_event("task.done", {"task_id": "t1"})
        self.assertEqual(client.pending_events, 1)
        client.close()

    def test_pending_spans(self):
        client = ArgusClient(flush_interval=999)
        span = client.start_span("op")
        span.end()
        self.assertEqual(client.pending_spans, 1)
        client.close()

    def test_context_manager(self):
        with ArgusClient(flush_interval=999) as client:
            span = client.start_span("op")
            span.end()
            self.assertEqual(client.pending_spans, 1)
        # After exiting, client is closed
        self.assertTrue(client._closed)

    def test_double_close(self):
        client = ArgusClient(flush_interval=999)
        client.close()
        client.close()  # Should not raise


class TestSpan(unittest.TestCase):
    """Tests for Span."""

    def setUp(self):
        self.client = ArgusClient(flush_interval=999)

    def tearDown(self):
        self.client.close()

    def test_set_attribute(self):
        span = self.client.start_span("op")
        span.set_attribute("key", "value")
        self.assertEqual(span.attributes["key"], "value")

    def test_set_error(self):
        span = self.client.start_span("op")
        span.set_error(ValueError("bad value"))
        self.assertEqual(span.error_code, "ValueError")
        self.assertEqual(span.attributes["error"], "bad value")

    def test_end_queues_span(self):
        span = self.client.start_span("op")
        span.set_attribute("k", "v")
        time.sleep(0.001)
        span.end()
        self.assertEqual(self.client.pending_spans, 1)

    def test_child_span(self):
        parent = self.client.start_span("parent")
        child = parent.child("child")
        self.assertEqual(child.trace_id, parent.trace_id)
        self.assertNotEqual(child.span_id, parent.span_id)

    def test_context_manager(self):
        with self.client.start_span("op") as span:
            span.set_attribute("k", "v")
        self.assertEqual(self.client.pending_spans, 1)

    def test_context_manager_with_error(self):
        try:
            with self.client.start_span("op") as span:
                raise RuntimeError("boom")
        except RuntimeError:
            pass
        self.assertEqual(self.client.pending_spans, 1)


class _MockHandler(BaseHTTPRequestHandler):
    """Mock HTTP handler that records requests."""

    requests = []

    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        _MockHandler.requests.append({
            "path": self.path,
            "body": json.loads(body) if body else {},
            "headers": dict(self.headers),
        })
        self.send_response(200)
        self.end_headers()

    def log_message(self, format, *args):
        pass  # Suppress output


class TestFlush(unittest.TestCase):
    """Tests for flush behavior with mock server."""

    @classmethod
    def setUpClass(cls):
        _MockHandler.requests = []
        cls.server = HTTPServer(("127.0.0.1", 0), _MockHandler)
        cls.port = cls.server.server_address[1]
        cls.thread = Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown()

    def setUp(self):
        _MockHandler.requests = []

    def test_flush_sends_spans(self):
        client = ArgusClient(
            endpoint=f"http://127.0.0.1:{self.port}",
            tenant_id="t1",
            agent_id="a1",
            flush_interval=999,
        )
        span = client.start_span("op1")
        span.end()
        client.flush()
        client.close()

        span_reqs = [r for r in _MockHandler.requests if r["path"] == "/api/v1/telemetry/spans"]
        self.assertEqual(len(span_reqs), 1)
        self.assertEqual(span_reqs[0]["body"]["tenant_id"], "t1")
        self.assertEqual(span_reqs[0]["body"]["agent_id"], "a1")
        self.assertEqual(len(span_reqs[0]["body"]["spans"]), 1)

    def test_flush_sends_events(self):
        client = ArgusClient(
            endpoint=f"http://127.0.0.1:{self.port}",
            tenant_id="t1",
            flush_interval=999,
        )
        client.emit_event("task.done", {"id": "1"})
        client.flush()
        client.close()

        event_reqs = [r for r in _MockHandler.requests if r["path"] == "/api/v1/telemetry/events"]
        self.assertEqual(len(event_reqs), 1)
        self.assertEqual(len(event_reqs[0]["body"]["events"]), 1)

    def test_flush_empty_noop(self):
        client = ArgusClient(
            endpoint=f"http://127.0.0.1:{self.port}",
            flush_interval=999,
        )
        client.flush()
        client.close()
        self.assertEqual(len(_MockHandler.requests), 0)

    def test_close_flushes(self):
        client = ArgusClient(
            endpoint=f"http://127.0.0.1:{self.port}",
            flush_interval=999,
        )
        span = client.start_span("op")
        span.end()
        client.close()

        span_reqs = [r for r in _MockHandler.requests if r["path"] == "/api/v1/telemetry/spans"]
        self.assertEqual(len(span_reqs), 1)


if __name__ == "__main__":
    unittest.main()
