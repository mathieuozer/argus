"""Tests for the Argus Python SDK decorators."""

import asyncio
import unittest

import argus
from argus.client import ArgusClient
from argus.decorators import init, trace, get_client, shutdown


class TestInit(unittest.TestCase):
    """Tests for init/shutdown."""

    def tearDown(self):
        shutdown()

    def test_init_creates_client(self):
        client = init(endpoint="http://test:8080", tenant_id="t1")
        self.assertIsInstance(client, ArgusClient)
        self.assertEqual(client.endpoint, "http://test:8080")
        self.assertEqual(client.tenant_id, "t1")

    def test_get_client_before_init(self):
        self.assertIsNone(get_client())

    def test_get_client_after_init(self):
        init()
        self.assertIsNotNone(get_client())

    def test_shutdown_clears_client(self):
        init()
        shutdown()
        self.assertIsNone(get_client())

    def test_double_shutdown(self):
        init()
        shutdown()
        shutdown()  # Should not raise


class TestTrace(unittest.TestCase):
    """Tests for the @trace decorator."""

    def setUp(self):
        self.client = init(flush_interval=999)

    def tearDown(self):
        shutdown()

    def test_trace_bare(self):
        @trace
        def my_func():
            return 42

        result = my_func()
        self.assertEqual(result, 42)
        self.assertEqual(self.client.pending_spans, 1)

    def test_trace_with_name(self):
        @trace(name="custom_name")
        def my_func():
            return "hello"

        result = my_func()
        self.assertEqual(result, "hello")
        self.assertEqual(self.client.pending_spans, 1)

    def test_trace_preserves_function_name(self):
        @trace
        def my_special_func():
            """My docstring."""
            pass

        self.assertEqual(my_special_func.__name__, "my_special_func")
        self.assertEqual(my_special_func.__doc__, "My docstring.")

    def test_trace_with_exception(self):
        @trace
        def failing_func():
            raise ValueError("boom")

        with self.assertRaises(ValueError):
            failing_func()

        # Span should still be recorded
        self.assertEqual(self.client.pending_spans, 1)

    def test_trace_without_client(self):
        shutdown()

        @trace
        def my_func():
            return 99

        result = my_func()
        self.assertEqual(result, 99)

    def test_trace_with_args(self):
        @trace
        def add(a, b):
            return a + b

        self.assertEqual(add(3, 4), 7)
        self.assertEqual(self.client.pending_spans, 1)

    def test_trace_async(self):
        @trace
        async def async_func():
            return "async_result"

        result = asyncio.run(async_func())
        self.assertEqual(result, "async_result")
        self.assertEqual(self.client.pending_spans, 1)

    def test_trace_async_with_error(self):
        @trace
        async def async_failing():
            raise RuntimeError("async boom")

        with self.assertRaises(RuntimeError):
            asyncio.run(async_failing())

        self.assertEqual(self.client.pending_spans, 1)

    def test_trace_async_without_client(self):
        shutdown()

        @trace
        async def async_func():
            return 42

        result = asyncio.run(async_func())
        self.assertEqual(result, 42)


if __name__ == "__main__":
    unittest.main()
