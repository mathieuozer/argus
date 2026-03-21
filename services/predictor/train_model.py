"""Generate a sample ONNX model for development and testing.

Trains a simple regression model on synthetic data that mimics the heuristic
predictor behavior, then exports it to model.onnx for the predictor service.

Usage:
    pip install scikit-learn skl2onnx
    python train_model.py
"""

import numpy as np
from sklearn.ensemble import GradientBoostingRegressor
from sklearn.multioutput import MultiOutputRegressor

FEATURE_NAMES = [
    "latency_p99_ratio",
    "token_velocity",
    "retry_rate",
    "error_rate_delta",
    "context_fill_pct",
    "tool_call_depth",
    "consecutive_slow",
    "cost_acceleration",
]

PRECURSOR_LABELS = ["latency_spike", "token_escalation", "retry_storm", "cost_runaway", "none"]

NUM_SAMPLES = 5000


def sigmoid(x: float) -> float:
    return 1.0 / (1.0 + np.exp(-x))


def generate_synthetic_data(n: int) -> tuple[np.ndarray, np.ndarray]:
    rng = np.random.default_rng(42)

    latency_p99_ratio = rng.uniform(0.5, 10.0, n)
    token_velocity = rng.uniform(0, 500, n)
    retry_rate = rng.uniform(0, 1, n)
    error_rate_delta = rng.uniform(-0.5, 1.0, n)
    context_fill_pct = rng.uniform(0, 1, n)
    tool_call_depth = rng.uniform(0, 20, n)
    consecutive_slow = rng.uniform(0, 30, n)
    cost_acceleration = rng.uniform(0, 10, n)

    X = np.column_stack([
        latency_p99_ratio,
        token_velocity,
        retry_rate,
        error_rate_delta,
        context_fill_pct,
        tool_call_depth,
        consecutive_slow,
        cost_acceleration,
    ])

    probabilities = np.zeros(n)
    ttf_seconds = np.zeros(n)
    precursor_indices = np.full(n, 4, dtype=float)  # default: "none"

    for i in range(n):
        candidates = []

        if latency_p99_ratio[i] >= 2:
            base = sigmoid((latency_p99_ratio[i] - 3) * 2)
            slow_boost = min(consecutive_slow[i] / 10, 0.3)
            prob = min(base + slow_boost, 1.0)
            ttf = 60 if latency_p99_ratio[i] > 5 else int(600 / max(latency_p99_ratio[i], 1))
            candidates.append((0, prob, ttf))

        if context_fill_pct[i] >= 0.5:
            fill_risk = sigmoid((context_fill_pct[i] - 0.7) * 8)
            velocity_factor = min(token_velocity[i] / 100, 1.0) * 0.3
            prob = min(fill_risk + velocity_factor, 1.0)
            remaining = 1.0 - context_fill_pct[i]
            if token_velocity[i] > 0 and remaining > 0:
                ttf = int(max(remaining * 1000 / token_velocity[i], 30))
            else:
                ttf = 30
            candidates.append((1, prob, ttf))

        if retry_rate[i] >= 0.1:
            base = sigmoid((retry_rate[i] - 0.3) * 5)
            error_boost = min(error_rate_delta[i] * 2, 0.3)
            prob = min(base + error_boost, 1.0)
            ttf = 30 if retry_rate[i] > 0.8 else int(300 / max(retry_rate[i] * 5, 1))
            candidates.append((2, prob, ttf))

        if cost_acceleration[i] > 1.0:
            base = sigmoid((cost_acceleration[i] - 2) * 2)
            token_factor = min(token_velocity[i] / 200, 0.2)
            prob = min(base + token_factor, 1.0)
            ttf = 120 if cost_acceleration[i] > 5 else int(600 / max(cost_acceleration[i], 1))
            candidates.append((3, prob, ttf))

        if candidates:
            best = max(candidates, key=lambda c: c[1])
            probabilities[i] = max(0.0, min(1.0, best[1]))
            ttf_seconds[i] = best[2]
            if probabilities[i] >= 0.1:
                precursor_indices[i] = best[0]

        # Add slight noise to make the model learn non-trivially
        probabilities[i] = np.clip(probabilities[i] + rng.normal(0, 0.02), 0, 1)
        ttf_seconds[i] = max(0, ttf_seconds[i] + rng.normal(0, 5))

    y = np.column_stack([probabilities, ttf_seconds, precursor_indices])
    return X.astype(np.float32), y.astype(np.float32)


def main():
    print(f"Generating {NUM_SAMPLES} synthetic samples...")
    X, y = generate_synthetic_data(NUM_SAMPLES)

    print("Training multi-output gradient boosting model...")
    model = MultiOutputRegressor(
        GradientBoostingRegressor(
            n_estimators=100,
            max_depth=5,
            learning_rate=0.1,
            random_state=42,
        )
    )
    model.fit(X, y)

    # Evaluate on training data
    y_pred = model.predict(X)
    prob_mae = np.mean(np.abs(y[:, 0] - y_pred[:, 0]))
    ttf_mae = np.mean(np.abs(y[:, 1] - y_pred[:, 1]))
    print(f"Training MAE - probability: {prob_mae:.4f}, ttf_seconds: {ttf_mae:.1f}")

    print("Exporting to ONNX...")
    from skl2onnx import convert_sklearn
    from skl2onnx.common.data_types import FloatTensorType

    initial_type = [("features", FloatTensorType([None, 8]))]
    onnx_model = convert_sklearn(model, initial_types=initial_type, target_opset=17)

    output_path = "model.onnx"
    with open(output_path, "wb") as f:
        f.write(onnx_model.SerializeToString())

    print(f"Model saved to {output_path}")

    # Verify the exported model
    import onnxruntime as ort

    session = ort.InferenceSession(output_path, providers=["CPUExecutionProvider"])
    test_input = X[:3]
    result = session.run(None, {"features": test_input})
    print(f"Verification - input shape: {test_input.shape}, output shape: {result[0].shape}")
    for i in range(3):
        prob = np.clip(result[0][i][0], 0, 1)
        ttf = max(0, int(result[0][i][1]))
        idx = int(np.clip(result[0][i][2], 0, 4))
        print(f"  Sample {i}: probability={prob:.4f}, ttf={ttf}s, precursor={PRECURSOR_LABELS[idx]}")


if __name__ == "__main__":
    main()
