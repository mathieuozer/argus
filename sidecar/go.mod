module github.com/argus-platform/argus/sidecar

go 1.26.1

require (
	github.com/argus-platform/argus/pkg v0.0.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.79.3
)

require (
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/nats-io/nats.go v1.52.0 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/argus-platform/argus/gen/go => ../gen/go
	github.com/argus-platform/argus/pkg => ../pkg
)
