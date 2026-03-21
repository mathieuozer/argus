"""Argus Predictive Failure Model Service.

FastAPI microservice serving the ONNX predictive failure model.
Falls back to sigmoid-based heuristics when no ONNX model file is present.

Called by the Go telemetry service at localhost:8090.
"""

import logging
import math
import os
from pathlib import Path

import numpy as np
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

logger = logging.getLogger("argus.predictor")
logging.basicConfig(
    level=getattr(logging, os.getenv("ARGUS_LOG_LEVEL", "INFO").upper(), logging.INFO),
    format="%(asctime)s %(levelname)s %(name)s %(message)s",
)

MODEL_PATH = Path(os.getenv("ARGUS_MODEL_PATH", "./model.onnx"))

app = FastAPI(title="Argus Predictor", version="1.0.0")

onnx_session = None


def _load_onnx_model():
    global onnx_session
    if MODEL_PATH.exists():
        try:
            import onnxruntime as ort

            onnx_session = ort.InferenceSession(
                str(MODEL_PATH),
                providers=["CPUExecutionProvider"],
            )
            input_name = onnx_session.get_inputs()[0].name
            logger.info("ONNX model loaded from %s (input: %s)", MODEL_PATH, input_name)
        except Exception:
            logger.exception("Failed to load ONNX model, falling back to heuristics")
            onnx_session = None
    else:
        logger.info("No ONNX model at %s, using heuristic fallback", MODEL_PATH)


@app.on_event("startup")
async def startup():
    _load_onnx_model()


class PredictionRequest(BaseModel):
    latency_p99_ratio: float = Field(ge=0, description="p99/p50 latency ratio over 5-min window")
    token_velocity: float = Field(description="Tokens/sec rate of change over last 10 tasks")
    retry_rate: float = Field(ge=0, le=1, description="Retries / total calls over last 5 min")
    error_rate_delta: float = Field(description="Error rate change vs 1h baseline")
    context_fill_pct: float = Field(ge=0, le=1, description="Estimated context window utilization")
    tool_call_depth: float = Field(ge=0, description="Average tool call nesting depth")
    consecutive_slow: float = Field(ge=0, description="Count of consecutive >2s tool calls")
    cost_acceleration: float = Field(description="Cost/task rate of change")


class PredictionResponse(BaseModel):
    failure_probability: float = Field(ge=0, le=1)
    ttf_seconds: int = Field(ge=0)
    precursor_type: str


PRECURSOR_LABELS = ["latency_spike", "token_escalation", "retry_storm", "cost_runaway", "none"]


def _sigmoid(x: float) -> float:
    return 1.0 / (1.0 + math.exp(-x))


def _heuristic_predict(req: PredictionRequest) -> PredictionResponse:
    candidates = []

    # Latency spike detection
    if req.latency_p99_ratio >= 2:
        base = _sigmoid((req.latency_p99_ratio - 3) * 2)
        slow_boost = min(req.consecutive_slow / 10, 0.3)
        prob = min(base + slow_boost, 1.0)
        ttf = 60 if req.latency_p99_ratio > 5 else int(600 / max(req.latency_p99_ratio, 1))
        candidates.append(("latency_spike", prob, ttf))

    # Token escalation detection
    if req.context_fill_pct >= 0.5:
        fill_risk = _sigmoid((req.context_fill_pct - 0.7) * 8)
        velocity_factor = min(req.token_velocity / 100, 1.0) * 0.3
        prob = min(fill_risk + velocity_factor, 1.0)
        remaining = 1.0 - req.context_fill_pct
        if req.token_velocity > 0 and remaining > 0:
            ttf = int(max(remaining * 1000 / req.token_velocity, 30))
        else:
            ttf = 30
        candidates.append(("token_escalation", prob, ttf))

    # Retry storm detection
    if req.retry_rate >= 0.1:
        base = _sigmoid((req.retry_rate - 0.3) * 5)
        error_boost = min(req.error_rate_delta * 2, 0.3)
        prob = min(base + error_boost, 1.0)
        ttf = 30 if req.retry_rate > 0.8 else int(300 / max(req.retry_rate * 5, 1))
        candidates.append(("retry_storm", prob, ttf))

    # Cost runaway detection
    if req.cost_acceleration > 1.0:
        base = _sigmoid((req.cost_acceleration - 2) * 2)
        token_factor = min(req.token_velocity / 200, 0.2)
        prob = min(base + token_factor, 1.0)
        ttf = 120 if req.cost_acceleration > 5 else int(600 / max(req.cost_acceleration, 1))
        candidates.append(("cost_runaway", prob, ttf))

    if not candidates:
        return PredictionResponse(failure_probability=0.0, ttf_seconds=0, precursor_type="none")

    best = max(candidates, key=lambda c: c[1])
    probability = max(0.0, min(1.0, best[1]))

    if probability < 0.1:
        return PredictionResponse(failure_probability=round(probability, 4), ttf_seconds=0, precursor_type="none")

    return PredictionResponse(
        failure_probability=round(probability, 4),
        ttf_seconds=best[2],
        precursor_type=best[0],
    )


def _onnx_predict(req: PredictionRequest) -> PredictionResponse:
    features = np.array(
        [[
            req.latency_p99_ratio,
            req.token_velocity,
            req.retry_rate,
            req.error_rate_delta,
            req.context_fill_pct,
            req.tool_call_depth,
            req.consecutive_slow,
            req.cost_acceleration,
        ]],
        dtype=np.float32,
    )

    input_name = onnx_session.get_inputs()[0].name
    outputs = onnx_session.run(None, {input_name: features})

    failure_probability = float(np.clip(outputs[0][0][0], 0.0, 1.0))
    ttf_seconds = max(0, int(outputs[0][0][1]))
    precursor_idx = int(np.clip(outputs[0][0][2], 0, len(PRECURSOR_LABELS) - 1))
    precursor_type = PRECURSOR_LABELS[precursor_idx]

    if failure_probability < 0.1:
        precursor_type = "none"
        ttf_seconds = 0

    return PredictionResponse(
        failure_probability=round(failure_probability, 4),
        ttf_seconds=ttf_seconds,
        precursor_type=precursor_type,
    )


@app.post("/predict", response_model=PredictionResponse)
async def predict(req: PredictionRequest):
    try:
        if onnx_session is not None:
            return _onnx_predict(req)
        return _heuristic_predict(req)
    except Exception as e:
        logger.exception("Prediction failed")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "model": "onnx" if onnx_session is not None else "heuristic",
        "model_path": str(MODEL_PATH),
    }


if __name__ == "__main__":
    import uvicorn

    port = int(os.getenv("ARGUS_PREDICTOR_PORT", "8090"))
    uvicorn.run(app, host="0.0.0.0", port=port, log_level="info")
