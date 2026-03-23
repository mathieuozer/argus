# Argus Simulator — generates live data for all dashboard pages
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache ca-certificates

WORKDIR /app

ENV GOWORK=off

COPY demo/simulator/go.mod demo/simulator/

RUN cd demo/simulator && go mod download

COPY demo/simulator/ demo/simulator/

RUN cd demo/simulator && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /app/simulator ./cmd/main

FROM alpine:3.23

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata

RUN addgroup -S argus && adduser -S argus -G argus
USER argus

COPY --from=builder /app/simulator /usr/local/bin/simulator

ENTRYPOINT ["/usr/local/bin/simulator"]
