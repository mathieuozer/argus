# Argus Predictor Service
# Multi-stage build: Python builder -> minimal runtime

# -- Stage 1: Build dependencies --
FROM python:3.12-slim AS builder

WORKDIR /app

COPY services/predictor/requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# -- Stage 2: Runtime --
FROM python:3.12-slim

RUN groupadd -r argus && useradd -r -g argus argus

WORKDIR /app

COPY --from=builder /install /usr/local
COPY services/predictor/main.py .

USER argus

EXPOSE 8090

CMD ["python", "main.py"]
