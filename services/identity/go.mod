module github.com/argus-platform/argus/services/identity

go 1.26.1

require (
	github.com/argus-platform/argus/gen/go v0.0.0
	github.com/argus-platform/argus/pkg v0.0.0
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)

replace (
	github.com/argus-platform/argus/gen/go => ../../gen/go
	github.com/argus-platform/argus/pkg => ../../pkg
)
