module github.com/argus-platform/argus/services/telemetry

go 1.22.0

require (
	github.com/argus-platform/argus/pkg v0.0.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.62.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
	golang.org/x/net v0.24.0 // indirect
	golang.org/x/sys v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240123012728-ef4313101c80 // indirect
)

replace github.com/argus-platform/argus/pkg => ../../pkg
