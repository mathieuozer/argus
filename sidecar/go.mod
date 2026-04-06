module github.com/argus-platform/argus/sidecar

go 1.26.1

require (
	github.com/argus-platform/argus/pkg v0.0.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.80.0
)

require (
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/nats-io/nats.go v1.49.0 // indirect
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/argus-platform/argus/gen/go => ../gen/go
	github.com/argus-platform/argus/pkg => ../pkg
)
