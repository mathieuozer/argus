"""Tests for the Argus predictive failure model service."""

import math
import unittest

from main import (
    PredictionRequest,
    _heuristic_predict,
    _sigmoid,
)


class TestSigmoid(unittest.TestCase):
    def test_zero(self):
        self.assertAlmostEqual(_sigmoid(0), 0.5)

    def test_large_positive(self):
        self.assertAlmostEqual(_sigmoid(10), 1.0, places=4)

    def test_large_negative(self):
        self.assertAlmostEqual(_sigmoid(-10), 0.0, places=4)

    def test_symmetry(self):
        for x in [0.5, 1.0, 2.0, 3.0]:
            self.assertAlmostEqual(_sigmoid(x) + _sigmoid(-x), 1.0, places=10)


class TestHeuristicPredict(unittest.TestCase):
    def _make_request(self, **kwargs):
        defaults = {
            "latency_p99_ratio": 1.0,
            "token_velocity": 0.0,
            "retry_rate": 0.0,
            "error_rate_delta": 0.0,
            "context_fill_pct": 0.0,
            "tool_call_depth": 1.0,
            "consecutive_slow": 0.0,
            "cost_acceleration": 0.0,
        }
        defaults.update(kwargs)
        return PredictionRequest(**defaults)

    def test_healthy_agent_returns_no_failure(self):
        req = self._make_request()
        resp = _heuristic_predict(req)
        self.assertEqual(resp.precursor_type, "none")
        self.assertEqual(resp.failure_probability, 0.0)
        self.assertEqual(resp.ttf_seconds, 0)

    def test_latency_spike_detected(self):
        req = self._make_request(latency_p99_ratio=5.0, consecutive_slow=8)
        resp = _heuristic_predict(req)
        self.assertEqual(resp.precursor_type, "latency_spike")
        self.assertGreater(resp.failure_probability, 0.5)
        self.assertGreater(resp.ttf_seconds, 0)

    def test_token_escalation_detected(self):
        req = self._make_request(context_fill_pct=0.9, token_velocity=50.0)
        resp = _heuristic_predict(req)
        self.assertEqual(resp.precursor_type, "token_escalation")
        self.assertGreater(resp.failure_probability, 0.5)

    def test_retry_storm_detected(self):
        req = self._make_request(retry_rate=0.6, error_rate_delta=0.3)
        resp = _heuristic_predict(req)
        self.assertEqual(resp.precursor_type, "retry_storm")
        self.assertGreater(resp.failure_probability, 0.3)

    def test_cost_runaway_detected(self):
        req = self._make_request(cost_acceleration=5.0, token_velocity=100.0)
        resp = _heuristic_predict(req)
        self.assertEqual(resp.precursor_type, "cost_runaway")
        self.assertGreater(resp.failure_probability, 0.3)

    def test_probability_clamped_to_1(self):
        req = self._make_request(
            latency_p99_ratio=100.0,
            consecutive_slow=100,
        )
        resp = _heuristic_predict(req)
        self.assertLessEqual(resp.failure_probability, 1.0)

    def test_worst_precursor_wins(self):
        """When multiple precursors fire, the highest probability wins."""
        req = self._make_request(
            latency_p99_ratio=3.0,
            retry_rate=0.9,
            error_rate_delta=0.5,
        )
        resp = _heuristic_predict(req)
        self.assertIn(resp.precursor_type, ["latency_spike", "retry_storm"])
        self.assertGreater(resp.failure_probability, 0.3)

    def test_low_probability_returns_none(self):
        req = self._make_request(latency_p99_ratio=2.0, consecutive_slow=0)
        resp = _heuristic_predict(req)
        if resp.failure_probability < 0.1:
            self.assertEqual(resp.precursor_type, "none")

    def test_ttf_decreases_with_severity(self):
        mild = self._make_request(latency_p99_ratio=2.5)
        severe = self._make_request(latency_p99_ratio=8.0)
        mild_resp = _heuristic_predict(mild)
        severe_resp = _heuristic_predict(severe)
        self.assertGreaterEqual(mild_resp.ttf_seconds, severe_resp.ttf_seconds)

    def test_context_fill_near_100_pct(self):
        req = self._make_request(context_fill_pct=0.99, token_velocity=200.0)
        resp = _heuristic_predict(req)
        self.assertEqual(resp.precursor_type, "token_escalation")
        self.assertGreater(resp.failure_probability, 0.8)


class TestPredictionRequest(unittest.TestCase):
    def test_valid_request(self):
        req = PredictionRequest(
            latency_p99_ratio=1.5,
            token_velocity=10.0,
            retry_rate=0.1,
            error_rate_delta=-0.01,
            context_fill_pct=0.3,
            tool_call_depth=2.0,
            consecutive_slow=0,
            cost_acceleration=0.5,
        )
        self.assertEqual(req.latency_p99_ratio, 1.5)

    def test_retry_rate_bounds(self):
        with self.assertRaises(Exception):
            PredictionRequest(
                latency_p99_ratio=1.0,
                token_velocity=0.0,
                retry_rate=1.5,  # > 1.0, should fail
                error_rate_delta=0.0,
                context_fill_pct=0.0,
                tool_call_depth=1.0,
                consecutive_slow=0,
                cost_acceleration=0.0,
            )


if __name__ == "__main__":
    unittest.main()
